/*
Copyright The Kubernetes Authors.

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

package metrics

import (
	"strings"
	"testing"

	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/testutil"
)

func TestRequestsProcessed(t *testing.T) {
	registry := metrics.NewKubeRegistry()
	defer registry.Reset()
	registry.MustRegister(RequestsProcessed)

	RequestsProcessed.WithLabelValues("driver-a.example.com").Inc()
	RequestsProcessed.WithLabelValues("driver-a.example.com").Inc()
	RequestsProcessed.WithLabelValues("driver-b.example.com").Inc()

	want := `# HELP resourcepoolstatusrequest_controller_requests_processed_total [ALPHA] Total number of ResourcePoolStatusRequests processed
# TYPE resourcepoolstatusrequest_controller_requests_processed_total counter
resourcepoolstatusrequest_controller_requests_processed_total{driver_name="driver-a.example.com"} 2
resourcepoolstatusrequest_controller_requests_processed_total{driver_name="driver-b.example.com"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want), "resourcepoolstatusrequest_controller_requests_processed_total"); err != nil {
		t.Errorf("unexpected metric output: %v", err)
	}
}

func TestRequestProcessingErrors(t *testing.T) {
	registry := metrics.NewKubeRegistry()
	defer registry.Reset()
	registry.MustRegister(RequestProcessingErrors)

	RequestProcessingErrors.WithLabelValues("driver-a.example.com").Inc()

	want := `# HELP resourcepoolstatusrequest_controller_request_processing_errors_total [ALPHA] Total number of errors encountered while processing ResourcePoolStatusRequests
# TYPE resourcepoolstatusrequest_controller_request_processing_errors_total counter
resourcepoolstatusrequest_controller_request_processing_errors_total{driver_name="driver-a.example.com"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want), "resourcepoolstatusrequest_controller_request_processing_errors_total"); err != nil {
		t.Errorf("unexpected metric output: %v", err)
	}
}

func TestRequestProcessingDuration(t *testing.T) {
	registry := metrics.NewKubeRegistry()
	defer registry.Reset()
	registry.MustRegister(RequestProcessingDuration)

	RequestProcessingDuration.WithLabelValues("driver-a.example.com").Observe(0.5)
	RequestProcessingDuration.WithLabelValues("driver-b.example.com").Observe(1.0)

	// Validate metric metadata and label structure; histogram bucket values
	// are deterministic given the observations above.
	want := `# HELP resourcepoolstatusrequest_controller_request_processing_duration_seconds [ALPHA] Time taken to process a ResourcePoolStatusRequest
# TYPE resourcepoolstatusrequest_controller_request_processing_duration_seconds histogram
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="0.001"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="0.002"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="0.004"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="0.008"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="0.016"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="0.032"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="0.064"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="0.128"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="0.256"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="0.512"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="1.024"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="2.048"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="4.096"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="8.192"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="16.384"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-a.example.com",le="+Inf"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_sum{driver_name="driver-a.example.com"} 0.5
resourcepoolstatusrequest_controller_request_processing_duration_seconds_count{driver_name="driver-a.example.com"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="0.001"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="0.002"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="0.004"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="0.008"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="0.016"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="0.032"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="0.064"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="0.128"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="0.256"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="0.512"} 0
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="1.024"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="2.048"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="4.096"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="8.192"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="16.384"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_bucket{driver_name="driver-b.example.com",le="+Inf"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_sum{driver_name="driver-b.example.com"} 1
resourcepoolstatusrequest_controller_request_processing_duration_seconds_count{driver_name="driver-b.example.com"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want), "resourcepoolstatusrequest_controller_request_processing_duration_seconds"); err != nil {
		t.Errorf("unexpected metric output: %v", err)
	}
}

func TestRegister(t *testing.T) {
	// Verify Register does not panic when called multiple times.
	Register()
	Register()
}
