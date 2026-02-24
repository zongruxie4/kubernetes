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
	"testing"

	"github.com/onsi/gomega"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/kubernetes/test/utils/ktesting"
)

func TestWithRESTConfig(t *testing.T) {
	tCtx := ktesting.Init(t)
	tCtx = WithRESTConfig(tCtx, new(rest.Config))
	config := RESTConfig(tCtx)
	tCtx.Assert(config).NotTo(gomega.BeNil(), "RESTConfig()")
	tCtx.Assert(config.UserAgent).To(gomega.ContainSubstring("TestWithRESTConfig"), "UserAgent")
	mapper := RESTMapper(tCtx)
	tCtx.Assert(mapper).NotTo(gomega.BeNil(), "RESTMapper()")
	client := Client(tCtx)
	tCtx.Assert(client).NotTo(gomega.BeNil(), "Client()")
	dynamic := Dynamic(tCtx)
	tCtx.Assert(dynamic).NotTo(gomega.BeNil(), "Dynamic()")
	extensions := APIExtensions(tCtx)
	tCtx.Assert(extensions).NotTo(gomega.BeNil(), "APIExtensions()")

	otherCtx := tCtx.WithCancel()
	tCtx.Assert(RESTConfig(otherCtx)).To(gomega.Equal(config), "RESTConfig()")
	tCtx.Assert(RESTMapper(otherCtx)).To(gomega.BeIdenticalTo(mapper), "RESTMapper()")
	tCtx.Assert(Client(otherCtx)).To(gomega.BeIdenticalTo(client), "Client()")
	tCtx.Assert(Dynamic(otherCtx)).To(gomega.BeIdenticalTo(dynamic), "Dynamic()")
	tCtx.Assert(APIExtensions(otherCtx)).To(gomega.BeIdenticalTo(extensions), "APIExtensions()")

	tCtx.CleanupCtx(func(tCtx ktesting.TContext) {
		tCtx.Assert(RESTConfig(tCtx)).To(gomega.Equal(config), "RESTConfig()")
		tCtx.Assert(RESTMapper(tCtx)).To(gomega.BeIdenticalTo(mapper), "RESTMapper()")
		tCtx.Assert(Client(tCtx)).To(gomega.BeIdenticalTo(client), "Client()")
		tCtx.Assert(Dynamic(tCtx)).To(gomega.BeIdenticalTo(dynamic), "Dynamic()")
		tCtx.Assert(APIExtensions(tCtx)).To(gomega.BeIdenticalTo(extensions), "APIExtensions()")
	})

	// Cancel, then let testing.T invoke test cleanup.
	tCtx.Cancel("test is complete")
}

func TestWithClients(t *testing.T) {
	tCtx := ktesting.Init(t)
	config := &rest.Config{UserAgent: "my-user-agent"}
	mapper := &restmapper.DeferredDiscoveryRESTMapper{}
	client := clientset.NewForConfigOrDie(config)
	dynamic := dynamic.NewForConfigOrDie(config)
	extensions := apiextensions.NewForConfigOrDie(config)
	tCtx = WithClients(tCtx, config, mapper, client, dynamic, extensions)
	tCtx.Assert(RESTConfig(tCtx)).To(gomega.Equal(config), "RESTConfig()")
	tCtx.Assert(RESTMapper(tCtx)).To(gomega.BeIdenticalTo(mapper), "RESTMapper()")
	tCtx.Assert(Client(tCtx)).To(gomega.BeIdenticalTo(client), "Client()")
	tCtx.Assert(Dynamic(tCtx)).To(gomega.BeIdenticalTo(dynamic), "Dynamic()")
	tCtx.Assert(APIExtensions(tCtx)).To(gomega.BeIdenticalTo(extensions), "APIExtensions()")

	otherCtx := tCtx.WithCancel()
	tCtx.Assert(RESTConfig(otherCtx)).To(gomega.Equal(config), "RESTConfig()")
	tCtx.Assert(RESTMapper(otherCtx)).To(gomega.BeIdenticalTo(mapper), "RESTMapper()")
	tCtx.Assert(Client(otherCtx)).To(gomega.BeIdenticalTo(client), "Client()")
	tCtx.Assert(Dynamic(otherCtx)).To(gomega.BeIdenticalTo(dynamic), "Dynamic()")
	tCtx.Assert(APIExtensions(otherCtx)).To(gomega.BeIdenticalTo(extensions), "APIExtensions()")

	tCtx.CleanupCtx(func(tCtx ktesting.TContext) {
		tCtx.Assert(RESTConfig(tCtx)).To(gomega.Equal(config), "RESTConfig()")
		tCtx.Assert(RESTMapper(tCtx)).To(gomega.BeIdenticalTo(mapper), "RESTMapper()")
		tCtx.Assert(Client(tCtx)).To(gomega.BeIdenticalTo(client), "Client()")
		tCtx.Assert(Dynamic(tCtx)).To(gomega.BeIdenticalTo(dynamic), "Dynamic()")
		tCtx.Assert(APIExtensions(tCtx)).To(gomega.BeIdenticalTo(extensions), "APIExtensions()")
	})

	// Cancel, then let testing.T invoke test cleanup.
	tCtx.Cancel("test is complete")
}
