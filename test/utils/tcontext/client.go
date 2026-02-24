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

package tcontext

import (
	"context"
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/kubernetes/test/utils/ktesting"
)

type data struct {
	restConfig    *rest.Config
	restMapper    *restmapper.DeferredDiscoveryRESTMapper
	client        clientset.Interface
	dynamic       dynamic.Interface
	apiextensions apiextensions.Interface
}

type dataKeyType struct{}

var dataKey dataKeyType

func get(ctx context.Context) data {
	c := ctx.Value(dataKey)
	if c == nil {
		return data{}
	}
	return *c.(*data)
}

func set(tCtx ktesting.TContext, data data) ktesting.TContext {
	return tCtx.WithValue(dataKey, &data)
}

// RESTConfig returns a copy of the config for a rest client with the UserAgent
// set to include the current test name or nil if not available. Several typed
// clients using this config are available through [Client], [Dynamic],
// [APIExtensions].
func RESTConfig(tCtx ktesting.TContext) *rest.Config {
	return rest.CopyConfig(get(tCtx).restConfig)
}

func RESTMapper(tCtx ktesting.TContext) *restmapper.DeferredDiscoveryRESTMapper {
	return get(tCtx).restMapper
}
func Client(tCtx ktesting.TContext) clientset.Interface            { return get(tCtx).client }
func Dynamic(tCtx ktesting.TContext) dynamic.Interface             { return get(tCtx).dynamic }
func APIExtensions(tCtx ktesting.TContext) apiextensions.Interface { return get(tCtx).apiextensions }

// WithRESTConfig initializes all client-go clients with new clients
// created for the config. The current test name gets included in the UserAgent.
func WithRESTConfig(tCtx ktesting.TContext, cfg *rest.Config) ktesting.TContext {
	cfg = rest.CopyConfig(cfg)
	cfg.UserAgent = fmt.Sprintf("%s -- %s", rest.DefaultKubernetesUserAgent(), tCtx.Name())

	var data data
	data.restConfig = cfg
	data.client = clientset.NewForConfigOrDie(cfg)
	data.dynamic = dynamic.NewForConfigOrDie(cfg)
	data.apiextensions = apiextensions.NewForConfigOrDie(cfg)
	cachedDiscovery := memory.NewMemCacheClient(data.client.Discovery())
	data.restMapper = restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscovery)
	return set(tCtx, data)
}

// WithClients uses an existing config and clients.
func WithClients(tCtx ktesting.TContext, cfg *rest.Config, mapper *restmapper.DeferredDiscoveryRESTMapper, client clientset.Interface, dynamic dynamic.Interface, apiextensions apiextensions.Interface) ktesting.TContext {
	return set(tCtx, data{
		restConfig:    cfg,
		restMapper:    mapper,
		client:        client,
		dynamic:       dynamic,
		apiextensions: apiextensions,
	})
}
