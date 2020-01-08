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
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/spire/proto/spire/api/registration"
	"github.com/spiffe/spire/proto/spire/common"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	serverSocketPath = "/run/spire/sockets/registration.sock"
	entriesPath      = "/run/spire/entries/registration.json"
)

func parse(path string) (common.RegistrationEntries, error) {
	f, err := os.Open(path)
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
			return errors.Wrapf(err, "failed to delete entry %v: %v", e.SpiffeId)
		}
	}

	logrus.Infof("successfully deleted %d entries", len(entries.Entries))

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
	if err := cleanup(context.Background(), client); err != nil {
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

	<-c
}
