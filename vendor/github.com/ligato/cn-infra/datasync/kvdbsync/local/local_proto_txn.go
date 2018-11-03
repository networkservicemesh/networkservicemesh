// Copyright (c) 2017 Cisco and/or its affiliates.
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

package local

import (
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/ligato/cn-infra/datasync"
	"github.com/ligato/cn-infra/datasync/syncbase"
	"github.com/ligato/cn-infra/db/keyval"
)

// NewProtoTxn is a constructor.
func NewProtoTxn(commit func(map[string] /*key*/ datasync.ChangeValue) error) *ProtoTxn {
	return &ProtoTxn{items: map[string] /*key*/ *ProtoTxnItem{}, commit: commit}
}

// ProtoTxn is a concurrent map of proto messages.
// The intent is to collect the user data and propagate them when commit happens.
type ProtoTxn struct {
	items  map[string] /*key*/ *ProtoTxnItem
	access sync.Mutex
	commit func(map[string] /*key*/ datasync.ChangeValue) error
}

//Put adds store operation into transaction.
func (txn *ProtoTxn) Put(key string, data proto.Message) keyval.ProtoTxn {
	txn.access.Lock()
	defer txn.access.Unlock()

	txn.items[key] = &ProtoTxnItem{data, false}

	return txn
}

//Delete adds delete operation into transaction.
func (txn *ProtoTxn) Delete(key string) keyval.ProtoTxn {
	txn.access.Lock()
	defer txn.access.Unlock()

	txn.items[key] = &ProtoTxnItem{nil, true}

	return txn
}

//Commit executes the transaction.
func (txn *ProtoTxn) Commit() error {
	txn.access.Lock()
	defer txn.access.Unlock()

	kvs := map[string] /*key*/ datasync.ChangeValue{}
	for key, item := range txn.items {
		changeType := datasync.Put
		if item.Delete {
			changeType = datasync.Delete
		}

		kvs[key] = syncbase.NewChange(key, item.Data, 0, changeType)
	}
	return txn.commit(kvs)
}

// ProtoTxnItem is used in ProtoTxn.
type ProtoTxnItem struct {
	Data   proto.Message
	Delete bool
}

// GetValue returns the value of the pair.
func (lazy *ProtoTxnItem) GetValue(out proto.Message) error {
	if lazy.Data != nil {
		proto.Merge(out, lazy.Data)
	}
	return nil
}
