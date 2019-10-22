// Copyright (c) 2019 Cisco Systems, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"net/http"
	"net/url"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

func createHTTPClient(roundTripper func(*http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(roundTripper),
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type restClient struct {
	Client               *http.Client
	NegotiatedSerializer runtime.NegotiatedSerializer
	GroupVersion         schema.GroupVersion
}

func (c *restClient) Get() *rest.Request {
	return c.request("GET")
}

func (c *restClient) Put() *rest.Request {
	return c.request("PUT")
}

func (c *restClient) Patch(pt types.PatchType) *rest.Request {
	return c.request("PATCH").SetHeader("Content-Type", string(pt))
}

func (c *restClient) Post() *rest.Request {
	return c.request("POST")
}

func (c *restClient) Delete() *rest.Request {
	return c.request("DELETE")
}

func (c *restClient) Verb(verb string) *rest.Request {
	return c.request(verb)
}

func (c *restClient) APIVersion() schema.GroupVersion {
	return c.GroupVersion
}

func (c *restClient) GetRateLimiter() flowcontrol.RateLimiter {
	return nil
}

func (c *restClient) request(verb string) *rest.Request {
	config := rest.ContentConfig{
		ContentType:          runtime.ContentTypeJSON,
		GroupVersion:         &c.GroupVersion,
		NegotiatedSerializer: c.NegotiatedSerializer,
	}

	ns := c.NegotiatedSerializer
	info, _ := runtime.SerializerInfoForMediaType(ns.SupportedMediaTypes(), runtime.ContentTypeJSON)
	serializers := rest.Serializers{
		// TODO this was hardcoded before, but it doesn't look right
		Encoder: ns.EncoderForVersion(info.Serializer, c.GroupVersion),
		Decoder: ns.DecoderToVersion(info.Serializer, c.GroupVersion),
	}
	if info.StreamSerializer != nil {
		serializers.StreamingSerializer = info.StreamSerializer.Serializer
		serializers.Framer = info.StreamSerializer.Framer
	}
	return rest.NewRequest(c, verb, &url.URL{Host: "localhost"}, "", config, serializers, nil, nil, 0)
}

func (c *restClient) Do(req *http.Request) (*http.Response, error) {
	if c.Client != nil {
		return c.Client.Do(req)
	}
	return nil, nil
}
