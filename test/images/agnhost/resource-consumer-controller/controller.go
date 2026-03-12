/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resconsumerctrl

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"k8s.io/kubernetes/test/images/agnhost/dns"
	"k8s.io/kubernetes/test/images/resource-consumer/common"
)

// CmdResourceConsumerController is used by agnhost Cobra.
var CmdResourceConsumerController = &cobra.Command{
	Use:   "resource-consumer-controller",
	Short: "Starts a HTTP server that spreads requests around resource consumers",
	Long:  "Starts a HTTP server that spreads requests around resource consumers. The HTTP server has the same endpoints and usage as the one spawned by the \"resource-consumer\" subcommand.",
	Args:  cobra.MaximumNArgs(0),
	Run:   main,
}

var (
	port                     int
	consumerPort             int
	consumerTargetPort       int
	consumerServiceName      string
	consumerServiceNamespace string
	dnsDomain                string
)

// getDNSDomain walks through DNS configuration and looks for "svc.foo" record
// where "foo" is currently configured DNS suffix. Then picks that 'foo' part up
// and returns to a caller.
func getDNSDomain() string {
	if dnsDomain != "" {
		return dnsDomain
	}
	dnsSuffixList := dns.GetDNSSuffixList()
	r, _ := regexp.Compile("^svc.")
	for _, currentDNSSuffix := range dnsSuffixList {
		if r.MatchString(currentDNSSuffix) {
			// Save DNS suffix without the 'svc.' part
			dnsDomain = currentDNSSuffix[4:]
			break
		}
	}
	if dnsDomain == "" {
		panic("Could not find DNS suffix starting with 'svc.' substring")
	}
	return dnsDomain
}

func init() {
	CmdResourceConsumerController.Flags().IntVar(&port, "port", 8080, "Port number.")
	CmdResourceConsumerController.Flags().IntVar(&consumerPort, "consumer-port", 8080, "Port number of consumers.")
	CmdResourceConsumerController.Flags().IntVar(&consumerTargetPort, "consumer-target-port", 8080, "Target (container) port for direct pod connections via headless DNS.")
	CmdResourceConsumerController.Flags().StringVar(&consumerServiceName, "consumer-service-name", "resource-consumer", "Name of service containing resource consumers.")
	CmdResourceConsumerController.Flags().StringVar(&consumerServiceNamespace, "consumer-service-namespace", "default", "Namespace of service containing resource consumers.")
}

func main(cmd *cobra.Command, args []string) {
	mgr := newController()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), mgr))
}

type controller struct {
	responseWriterLock sync.Mutex
	httpClient         *http.Client
	waitGroup          sync.WaitGroup
}

func newController() *controller {
	return &controller{
		// 30s timeout prevents hung goroutines on stale pod IPs
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *controller) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(w, common.BadRequest, http.StatusBadRequest)
		return
	}
	// parsing POST request data and URL data
	if err := req.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// handle consumeCPU
	if req.URL.Path == common.ConsumeCPUAddress {
		c.handleConsumeCPU(w, req.Form)
		return
	}
	// handle consumeMem
	if req.URL.Path == common.ConsumeMemAddress {
		c.handleConsumeMem(w, req.Form)
		return
	}
	// handle bumpMetric
	if req.URL.Path == common.BumpMetricAddress {
		c.handleBumpMetric(w, req.Form)
		return
	}
	http.Error(w, common.UnknownFunction, http.StatusNotFound)
}

func (c *controller) handleConsumeCPU(w http.ResponseWriter, query url.Values) {
	durationSecString := query.Get(common.DurationSecQuery)
	millicoresString := query.Get(common.MillicoresQuery)
	requestSizeInMillicoresString := query.Get(common.RequestSizeInMillicoresQuery)
	if durationSecString == "" || millicoresString == "" || requestSizeInMillicoresString == "" {
		http.Error(w, common.NotGivenFunctionArgument, http.StatusBadRequest)
		return
	}

	durationSec, durationSecError := strconv.Atoi(durationSecString)
	millicores, millicoresError := strconv.Atoi(millicoresString)
	requestSizeInMillicores, requestSizeInMillicoresError := strconv.Atoi(requestSizeInMillicoresString)
	if durationSecError != nil || millicoresError != nil || requestSizeInMillicoresError != nil || requestSizeInMillicores <= 0 {
		http.Error(w, common.IncorrectFunctionArgument, http.StatusBadRequest)
		return
	}

	// Attempt headless DNS resolution: <service>-headless.<ns>.svc.<domain>
	// returns one A record per pod, allowing deterministic per-pod distribution.
	// consumerTargetPort (default 8080) is the actual container port not
	// consumerPort (80) which is the ClusterIP service port.
	headlessDNS := fmt.Sprintf("%s-headless.%s.svc.%s",
		consumerServiceName, consumerServiceNamespace, getDNSDomain())
	podAddrs, err := net.LookupHost(headlessDNS)

	if err == nil && len(podAddrs) > 0 {
		perPodMillicores := millicores / len(podAddrs)
		if perPodMillicores == 0 {
			perPodMillicores = 1
		}
		_, _ = fmt.Fprintf(w, "RC manager (per-pod): sending %d millicores to each of %d pods via headless DNS\n",
			perPodMillicores, len(podAddrs))
		c.waitGroup.Add(len(podAddrs))
		for _, addr := range podAddrs {
			// Use consumerTargetPort (8080) not consumerPort (80): pods listen
			// directly on the container port, not the service port.
			podURL := fmt.Sprintf("http://%s:%d%s", addr, consumerTargetPort, common.ConsumeCPUAddress)
			go c.sendOneCPURequestToURL(w, podURL, perPodMillicores, durationSec)
		}
		c.waitGroup.Wait()
		return
	}

	// Fallback: original ClusterIP behavior. Gracefully degrades to old
	// non-deterministic routing if headless service is unavailable.
	log.Printf("headless DNS lookup for %s failed (%v), using ClusterIP fallback", headlessDNS, err)
	count := millicores / requestSizeInMillicores
	rest := millicores - count*requestSizeInMillicores
	_, _ = fmt.Fprintf(w, "RC manager: sending %v requests to consume %v millicores each and 1 request to consume %v millicores\n",
		count, requestSizeInMillicores, rest)
	if count > 0 {
		c.waitGroup.Add(count)
		c.sendConsumeCPURequests(w, count, requestSizeInMillicores, durationSec)
	}
	if rest > 0 {
		c.waitGroup.Add(1)
		go c.sendOneConsumeCPURequest(w, rest, durationSec)
	}
	c.waitGroup.Wait()
}

func (c *controller) handleConsumeMem(w http.ResponseWriter, query url.Values) {
	// getting string data for consumeMem
	durationSecString := query.Get(common.DurationSecQuery)
	megabytesString := query.Get(common.MegabytesQuery)
	requestSizeInMegabytesString := query.Get(common.RequestSizeInMegabytesQuery)
	if durationSecString == "" || megabytesString == "" || requestSizeInMegabytesString == "" {
		http.Error(w, common.NotGivenFunctionArgument, http.StatusBadRequest)
		return
	}

	// convert data (strings to ints) for consumeMem
	durationSec, durationSecError := strconv.Atoi(durationSecString)
	megabytes, megabytesError := strconv.Atoi(megabytesString)
	requestSizeInMegabytes, requestSizeInMegabytesError := strconv.Atoi(requestSizeInMegabytesString)
	if durationSecError != nil || megabytesError != nil || requestSizeInMegabytesError != nil || requestSizeInMegabytes <= 0 {
		http.Error(w, common.IncorrectFunctionArgument, http.StatusBadRequest)
		return
	}

	count := megabytes / requestSizeInMegabytes
	rest := megabytes - count*requestSizeInMegabytes
	fmt.Fprintf(w, "RC manager: sending %v requests to consume %v MB each and 1 request to consume %v MB\n", count, requestSizeInMegabytes, rest)
	if count > 0 {
		c.waitGroup.Add(count)
		c.sendConsumeMemRequests(w, count, requestSizeInMegabytes, durationSec)
	}
	if rest > 0 {
		c.waitGroup.Add(1)
		go c.sendOneConsumeMemRequest(w, rest, durationSec)
	}
	c.waitGroup.Wait()
}

func (c *controller) handleBumpMetric(w http.ResponseWriter, query url.Values) {
	// getting string data for handleBumpMetric
	metric := query.Get(common.MetricNameQuery)
	deltaString := query.Get(common.DeltaQuery)
	durationSecString := query.Get(common.DurationSecQuery)
	requestSizeCustomMetricString := query.Get(common.RequestSizeCustomMetricQuery)
	if durationSecString == "" || metric == "" || deltaString == "" || requestSizeCustomMetricString == "" {
		http.Error(w, common.NotGivenFunctionArgument, http.StatusBadRequest)
		return
	}

	// convert data (strings to ints/floats) for handleBumpMetric
	durationSec, durationSecError := strconv.Atoi(durationSecString)
	delta, deltaError := strconv.Atoi(deltaString)
	requestSizeCustomMetric, requestSizeCustomMetricError := strconv.Atoi(requestSizeCustomMetricString)
	if durationSecError != nil || deltaError != nil || requestSizeCustomMetricError != nil || requestSizeCustomMetric <= 0 {
		http.Error(w, common.IncorrectFunctionArgument, http.StatusBadRequest)
		return
	}

	count := delta / requestSizeCustomMetric
	rest := delta - count*requestSizeCustomMetric
	fmt.Fprintf(w, "RC manager: sending %v requests to bump custom metric by %v each and 1 request to bump by %v\n", count, requestSizeCustomMetric, rest)
	if count > 0 {
		c.waitGroup.Add(count)
		c.sendConsumeCustomMetric(w, metric, count, requestSizeCustomMetric, durationSec)
	}
	if rest > 0 {
		c.waitGroup.Add(1)
		go c.sendOneConsumeCustomMetric(w, metric, rest, durationSec)
	}
	c.waitGroup.Wait()
}

func (c *controller) sendConsumeCPURequests(w http.ResponseWriter, requests, millicores, durationSec int) {
	for range requests {
		go c.sendOneConsumeCPURequest(w, millicores, durationSec)
	}
}

// sendOneCPURequestToURL sends a ConsumeCPU POST directly to a pod IP URL.
// Uses c.httpClient (30s timeout) to prevent goroutine leaks on stale pod IPs.
func (c *controller) sendOneCPURequestToURL(w http.ResponseWriter, podURL string, millicores, durationSec int) {
	defer c.waitGroup.Done()
	params := url.Values{}
	params.Set(common.MillicoresQuery, strconv.Itoa(millicores))
	params.Set(common.DurationSecQuery, strconv.Itoa(durationSec))
	params.Set(common.RequestSizeInMillicoresQuery, strconv.Itoa(millicores))
	targetURL := podURL + "?" + params.Encode()

	resp, err := c.httpClient.Post(targetURL, "text/plain", nil)
	if err != nil {
		c.responseWriterLock.Lock()
		defer c.responseWriterLock.Unlock()
		_, _ = fmt.Fprintf(w, "Failed to send to pod at %s: %v\n", podURL, err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
}

func (c *controller) sendConsumeMemRequests(w http.ResponseWriter, requests, megabytes, durationSec int) {
	for range requests {
		go c.sendOneConsumeMemRequest(w, megabytes, durationSec)
	}
}

func (c *controller) sendConsumeCustomMetric(w http.ResponseWriter, metric string, requests, delta, durationSec int) {
	for range requests {
		go c.sendOneConsumeCustomMetric(w, metric, delta, durationSec)
	}
}

func createConsumerURL(suffix string) string {
	// NOTE: full DNS name is used due to the Windows platform restriction where PQDNs are not supported.
	return fmt.Sprintf("http://%s.%s.svc.%s:%d%s", consumerServiceName, consumerServiceNamespace, getDNSDomain(), consumerPort, suffix)
}

// sendOneConsumeCPURequest sends POST request for cpu consumption
func (c *controller) sendOneConsumeCPURequest(w http.ResponseWriter, millicores int, durationSec int) {
	defer c.waitGroup.Done()
	query := createConsumerURL(common.ConsumeCPUAddress)
	_, err := http.PostForm(query, url.Values{common.MillicoresQuery: {strconv.Itoa(millicores)}, common.DurationSecQuery: {strconv.Itoa(durationSec)}})
	c.responseWriterLock.Lock()
	defer c.responseWriterLock.Unlock()
	if err != nil {
		fmt.Fprintf(w, "Failed to connect to consumer: %v\n", err)
		return
	}
	fmt.Fprintf(w, "Consumed %d millicores\n", millicores)
}

// sendOneConsumeMemRequest sends POST request for memory consumption
func (c *controller) sendOneConsumeMemRequest(w http.ResponseWriter, megabytes int, durationSec int) {
	defer c.waitGroup.Done()
	query := createConsumerURL(common.ConsumeMemAddress)
	_, err := http.PostForm(query, url.Values{common.MegabytesQuery: {strconv.Itoa(megabytes)}, common.DurationSecQuery: {strconv.Itoa(durationSec)}})
	c.responseWriterLock.Lock()
	defer c.responseWriterLock.Unlock()
	if err != nil {
		fmt.Fprintf(w, "Failed to connect to consumer: %v\n", err)
		return
	}
	fmt.Fprintf(w, "Consumed %d megabytes\n", megabytes)
}

// sendOneConsumeCustomMetric sends POST request for custom metric consumption
func (c *controller) sendOneConsumeCustomMetric(w http.ResponseWriter, customMetricName string, delta int, durationSec int) {
	defer c.waitGroup.Done()
	query := createConsumerURL(common.BumpMetricAddress)
	_, err := http.PostForm(query,
		url.Values{common.MetricNameQuery: {customMetricName}, common.DurationSecQuery: {strconv.Itoa(durationSec)}, common.DeltaQuery: {strconv.Itoa(delta)}})
	c.responseWriterLock.Lock()
	defer c.responseWriterLock.Unlock()
	if err != nil {
		fmt.Fprintf(w, "Failed to connect to consumer: %v\n", err)
		return
	}
	fmt.Fprintf(w, "Bumped metric %s by %d\n", customMetricName, delta)
}
