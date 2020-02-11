// Copyright (c) 2020 Cisco and/or its affiliates.
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	federation "github.com/networkservicemesh/networkservicemesh/applications/federation-server/api"
	"github.com/networkservicemesh/networkservicemesh/utils"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/spire/proto/spire/api/registration"
	"github.com/spiffe/spire/proto/spire/common"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	serverSocketPath        = "/run/spire/sockets/registration.sock"
	entriesPath             = "/run/spire/entries/registration.json"
	federationServerEnv     = "FEDERATION_SERVER"
	podIPEnv                = "POD_IP"
	federationServerDefault = "federation-server.default"
)

func parse(path string) (common.RegistrationEntries, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return common.RegistrationEntries{}, err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return common.RegistrationEntries{}, err
	}

	var rv common.RegistrationEntries
	if err := json.Unmarshal(b, &rv); err != nil {
		return common.RegistrationEntries{}, err
	}

	return rv, nil
}

func cleanup(ctx context.Context, client registration.RegistrationClient) error {
	entries, err := client.FetchEntries(ctx, &common.Empty{})
	if err != nil {
		return err
	}

	for _, e := range entries.Entries {
		_, err = client.DeleteEntry(ctx, &registration.RegistrationEntryID{Id: e.EntryId})
		if err != nil {
			return errors.Wrapf(err, "failed to delete entry %v", e.SpiffeId)
		}
	}

	logrus.Infof("successfully deleted %d entries", len(entries.Entries))

	return nil
}

func listenBundles(ctx context.Context, bundle *common.Bundle, h func(*common.Bundle)) error {
	host := utils.EnvVar(federationServerEnv).GetStringOrDefault(federationServerDefault)
	address, ok := os.LookupEnv(podIPEnv)
	if !ok {
		return errors.New(fmt.Sprintf("env %v is not set", podIPEnv))
	}

	logrus.Infof("Bundle registration in federation-server: %v", host)
	cc, err := grpc.Dial(fmt.Sprintf("%s:%d", host, 7002), grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer func() { _ = cc.Close() }()

	reg := federation.NewRegistrationClient(cc)

	cbundle := proto.Clone(bundle).(*common.Bundle)
	cbundle.TrustDomainId = fmt.Sprintf("%v;%v", bundle.GetTrustDomainId(), address)
	if _, err = reg.CreateFederatedBundle(ctx, cbundle); err != nil {
		return err
	}

	logrus.Infof("Bundle successfully registered")

	logrus.Infof("Fetching list of bundles...")
	stream, err := reg.ListFederatedBundles(ctx, &common.Empty{})
	if err != nil {
		return err
	}

	for {
		m, err := stream.Recv()
		if err != nil {
			return err
		}

		for _, b := range m.GetBundles() {
			logrus.Infof("Received bundle for %v", b.GetTrustDomainId())
			h(b)
		}
	}
}

func patchHosts(address, domain string) error {
	f, err := os.OpenFile("/etc/hosts", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	record := fmt.Sprintf("%v %v\n", address, domain)
	if _, err := f.WriteString(record); err != nil {
		return err
	}
	logrus.Infof("successfully wrote to resolv.conf: %v", record)
	return nil
}

func main() {
	var err error
	c := tools.NewOSSignalChannel()

	var serverConn *grpc.ClientConn
	for {
		serverConn, err = tools.DialUnixInsecure(serverSocketPath)
		if err == nil {
			break
		}
		logrus.Errorf("failed to dial server: %v", err)
	}

	client := registration.NewRegistrationClient(serverConn)
	err = cleanup(context.Background(), client)
	if err != nil {
		logrus.Error(err)
		return
	}

	entries, err := parse(entriesPath)
	if err != nil {
		logrus.Error(err)
		return
	}

	for _, e := range entries.Entries {
		id, err := client.CreateEntry(context.Background(), e)
		if err != nil {
			logrus.Error(err)
			return
		}
		logrus.Infof("successfully register entry %v", id)
	}

	bundle, err := client.FetchBundle(context.Background(), &common.Empty{})
	if err != nil {
		logrus.Error(err)
		return
	}

	logrus.Infof("TrustDomain: %v", bundle.GetBundle().GetTrustDomainId())
	logrus.Infof("Bundle: %v", bundle.GetBundle())

	h := func(b *common.Bundle) {
		s := strings.Split(b.GetTrustDomainId(), ";")
		if len(s) != 2 {
			logrus.Error("wrong trust_id format")
			return
		}

		td, address := s[0], s[1]
		url, err := url.Parse(td)
		if err != nil {
			logrus.Error("wrong trust_id format: %v", err)
			return
		}

		if err := patchHosts(address, url.Host); err != nil {
			logrus.Error(err)
			return
		}

		cbundle := proto.Clone(b).(*common.Bundle)
		cbundle.TrustDomainId = td

		_, err = client.CreateFederatedBundle(context.Background(), &registration.FederatedBundle{
			Bundle: cbundle,
		})
		if err == nil {
			logrus.Infof("Successfully create bundle for %v", td)
		}

		// otherwise update existing bundle
		_, err = client.UpdateFederatedBundle(context.Background(), &registration.FederatedBundle{
			Bundle: cbundle,
		})
		if err != nil {
			logrus.Error(err)
			return
		}
	}

	if err := listenBundles(context.Background(), bundle.GetBundle(), h); err != nil {
		logrus.Error(err)
		return
	}

	<-c
}
