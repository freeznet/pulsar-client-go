// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package internal

import (
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

// Map: PulsarServiceNameResolver.java

type ServiceNameResolver interface {
	ResolveHost() (*url.URL, error)
	ResolveHostUri() (*PulsarServiceURI, error)
	UpdateServiceUrl(url *url.URL) error
	GetServiceURI() *PulsarServiceURI
	GetServiceURL() *url.URL
	GetAddressList() []*url.URL
}

type PulsarServiceNameResolver struct {
	ServiceURI   *PulsarServiceURI
	ServiceURL   *url.URL
	CurrentIndex int32
	AddressList  []*url.URL
}

func NewPulsarServiceNameResolver(url *url.URL) ServiceNameResolver {
	r := &PulsarServiceNameResolver{}
	err := r.UpdateServiceUrl(url)
	if err != nil {
		log.Errorf("create pulsar service name resolver failed : %v", err)
	}
	return r
}

func (r *PulsarServiceNameResolver) ResolveHost() (*url.URL, error) {
	if r.AddressList == nil {
		return nil, errors.New("no service url is provided yet")
	}
	if len(r.AddressList) == 0 {
		return nil, fmt.Errorf("no hosts found for service url : %v", r.ServiceURL)
	}
	if len(r.AddressList) == 1 {
		return r.AddressList[0], nil
	}
	idx := (r.CurrentIndex + 1) % int32(len(r.AddressList))
	atomic.StoreInt32(&r.CurrentIndex, idx)
	return r.AddressList[idx], nil
}

func (r *PulsarServiceNameResolver) ResolveHostUri() (*PulsarServiceURI, error) {
	host, err := r.ResolveHost()
	if err != nil {
		return nil, err
	}
	hostUrl := host.Scheme + "://" + host.Hostname() + ":" + host.Port()
	return NewPulsarServiceURIFromUriString(hostUrl)
}

func (r *PulsarServiceNameResolver) UpdateServiceUrl(u *url.URL) error {
	uri, err := NewPulsarServiceURIFromUrl(u)
	if err != nil {
		log.Errorf("invalid service-url instance %s provided %v", u, err)
		return err
	}

	hosts := uri.ServiceHosts
	var addresses []*url.URL
	for _, host := range hosts {
		hostUrl := uri.URL.Scheme + "://" + host
		u, err := url.Parse(hostUrl)
		if err != nil {
			log.Errorf("invalid host-url %s provided %v", hostUrl, err)
			return err
		}
		addresses = append(addresses, u)
	}
	r.AddressList = addresses
	r.ServiceURL = u
	r.ServiceURI = uri
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	atomic.StoreInt32(&r.CurrentIndex, int32(rand.Intn(len(addresses))))
	return nil
}

func (r *PulsarServiceNameResolver) GetServiceURI() *PulsarServiceURI {
	return r.ServiceURI
}

func (r *PulsarServiceNameResolver) GetServiceURL() *url.URL {
	return r.ServiceURL
}

func (r *PulsarServiceNameResolver) GetAddressList() []*url.URL {
	return r.AddressList
}
