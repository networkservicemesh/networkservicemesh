// Copyright (c) 2018 Cisco and/or its affiliates.
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

package netmesh

import (
	"encoding/json"
	"sync"

	"github.com/ligato/cn-infra/db/keyval"

	"github.com/golang/protobuf/proto"

	"github.com/ligato/cn-infra/datasync"
)

// Error message if not data is found for a given key
const (
	noDataForKey = "No data assigned to key "
)

// KeyProtoValBroker defines KSR's interface to the key-value data store. It
// defines a subset of operations from a generic cn-infra broker interface
// (keyval.ProtoBroker in cn-infra).
type KeyProtoValBroker interface {
	// Put <data> to ETCD or to any other key-value based data source.
	Put(key string, data proto.Message, opts ...datasync.PutOption) error

	// Delete data under the <key> in ETCD or in any other key-value based data
	// source.
	Delete(key string, opts ...datasync.DelOption) (existed bool, err error)

	// GetValue reads a value from etcd stored under the given key.
	GetValue(key string, reqObj proto.Message) (found bool, revision int64, err error)

	// List values stored in etcd under the given prefix.
	ListValues(prefix string) (keyval.ProtoKeyValIterator, error)
}

// mockKeyProtoValBroker is a mock implementation of KSR's interface to the
// key-value data store.
type mockKeyProtoValBroker struct {
	numRwErr   int
	rwErr      error
	numListErr int
	listErr    error
	ds         map[string]dataStoreItem
	dsMtx      sync.RWMutex
}

// dataStoreItem defines the structure of values stored in the key-value data
// store mock.
type dataStoreItem struct {
	val proto.Message
	rev int64
}

// newMockKeyProtoValBroker returns a new instance of mockKeyProtoValBroker.
func newMockKeyProtoValBroker() *mockKeyProtoValBroker {
	return &mockKeyProtoValBroker{
		numRwErr:   0,
		numListErr: 0,
		rwErr:      nil,
		listErr:    nil,
		ds:         make(map[string]dataStoreItem),
	}
}

// injectReadWriteError sets the error value to be returned from read and write
// operations. The error will be returned from 'numRwErr' calls to Put, Delete,
// and GetValue operations, then cleared.
func (mock *mockKeyProtoValBroker) injectReadWriteError(err error, numErr int) {
	mock.numRwErr = numErr
	mock.rwErr = err
}

// injectListError sets the error value to be returned from list operations. The
// operations. The error will be returned from 'numRwErr' calls to ListValues(),
// and GetValue operations, then cleared.
func (mock *mockKeyProtoValBroker) injectListError(err error, numErr int) {
	mock.numListErr = numErr
	mock.listErr = err
}

// clearReadWriteError resets the error value returned from Put, Delete and
// GetValue operations.
func (mock *mockKeyProtoValBroker) clearReadWriteError() {
	mock.injectReadWriteError(nil, 0)
}

// clearListError resets the error value returned from Put, Delete and
// GetValue operations.
func (mock *mockKeyProtoValBroker) clearListError() {
	mock.injectListError(nil, 0)
}

// Put puts data into an in-memory map simulating a key-value datastore.
func (mock *mockKeyProtoValBroker) Put(key string, data proto.Message, opts ...datasync.PutOption) error {
	mock.dsMtx.Lock()
	defer mock.dsMtx.Unlock()

	if mock.numRwErr > 0 {
		mock.numRwErr--
		return mock.rwErr
	}

	newData := dataStoreItem{val: data, rev: 1}
	oldData, found := mock.ds[key]
	if found {
		newData.rev = oldData.rev + 1
	}
	mock.ds[key] = newData
	return nil
}

// Delete removes data from an in-memory map simulating a key-value datastore.
func (mock *mockKeyProtoValBroker) Delete(key string, opts ...datasync.DelOption) (existed bool, err error) {
	mock.dsMtx.Lock()
	defer mock.dsMtx.Unlock()

	if mock.numRwErr > 0 {
		mock.numRwErr--
		return false, mock.rwErr
	}

	_, existed = mock.ds[key]
	if !existed {
		return false, nil
	}
	delete(mock.ds, key)
	return true, nil
}

// GetValue is a helper for unit tests to get value stored under a given key.
func (mock *mockKeyProtoValBroker) GetValue(key string, out proto.Message) (found bool, revision int64, err error) {
	mock.dsMtx.Lock()
	defer mock.dsMtx.Unlock()

	if mock.numRwErr > 0 {
		mock.numRwErr--
		return false, 0, mock.rwErr
	}

	data, exists := mock.ds[key]
	if !exists {
		return false, 0, nil
	}
	proto.Merge(out, data.val)
	return true, data.rev, nil
}

// ClearDs is a helper which allows to clear the in-memory map simulating
// a key-value datastore.
func (mock *mockKeyProtoValBroker) ClearDs() {
	mock.dsMtx.Lock()
	defer mock.dsMtx.Unlock()

	for key := range mock.ds {
		delete(mock.ds, key)
	}
	mock.numRwErr = 0
	mock.numListErr = 0
	mock.rwErr = nil
	mock.listErr = nil
}

// ListValues returns the mockProtoKeyValIterator which will contain some
// mock values down the road
func (mock *mockKeyProtoValBroker) ListValues(prefix string) (keyval.ProtoKeyValIterator, error) {
	mock.dsMtx.RLock()
	defer mock.dsMtx.RUnlock()

	if mock.numListErr > 0 {
		mock.numListErr--
		return nil, mock.rwErr
	}

	var values []keyval.ProtoKeyVal
	for key, dsItem := range mock.ds {
		pkv := mockProtoKeyval{
			key: key,
			msg: dsItem.val,
		}
		values = append(values, &pkv)
	}
	return &mockProtoKeyValIterator{
		values: values,
		idx:    0,
	}, nil
}

// mockProtoKeyValIterator is a mock implementation of ProtoKeyValIterator
// used in unit tests.
type mockProtoKeyValIterator struct {
	values []keyval.ProtoKeyVal
	idx    int
}

type mockProtoKeyval struct {
	key string
	msg proto.Message
}

func (pkv *mockProtoKeyval) GetKey() string {
	return pkv.key
}

func (pkv *mockProtoKeyval) GetPrevValue(prevValue proto.Message) (prevValueExist bool, err error) {
	return false, nil
}

func (pkv *mockProtoKeyval) GetValue(value proto.Message) error {
	buf, err := json.Marshal(pkv.msg)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, value)
}

func (pkv *mockProtoKeyval) GetRevision() (rev int64) {
	return 0
}

// GetNext getting the next mocked keyval.ProtoKeyVal value from
// mockProtoKeyValIterator
func (it *mockProtoKeyValIterator) GetNext() (kv keyval.ProtoKeyVal, stop bool) {
	if it.idx == len(it.values) {
		return nil, true
	}
	kv = it.values[it.idx]
	it.idx++
	return kv, stop
}

// Close is a mock for mockProtoKeyValIterator
func (it *mockProtoKeyValIterator) Close() error {
	return nil
}
