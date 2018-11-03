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

	"github.com/ligato/cn-infra/datasync"
	"github.com/ligato/cn-infra/datasync/syncbase"
	"github.com/ligato/cn-infra/db/keyval"
)

// NewBytesTxn is a constructor.
func NewBytesTxn(commit func(map[string] /*key*/ datasync.ChangeValue) error) *BytesTxn {
	return &BytesTxn{items: map[string] /*key*/ *BytesTxnItem{}, commit: commit}
}

// BytesTxn is just a concurrent map of Bytes messages.
// The intent is to collect the user data and propagate them when commit happens.
type BytesTxn struct {
	items  map[string] /*key*/ *BytesTxnItem
	access sync.Mutex
	commit func(map[string] /*key*/ datasync.ChangeValue) error
}

//Put adds store operation into transaction.
func (txn *BytesTxn) Put(key string, data []byte) keyval.BytesTxn {
	txn.access.Lock()
	defer txn.access.Unlock()

	txn.items[key] = &BytesTxnItem{data, false}

	return txn
}

//Delete add delete operation into transaction.
func (txn *BytesTxn) Delete(key string) keyval.BytesTxn {
	txn.access.Lock()
	defer txn.access.Unlock()

	txn.items[key] = &BytesTxnItem{nil, true}

	return txn
}

//Commit executes the transaction.
func (txn *BytesTxn) Commit() error {
	txn.access.Lock()
	defer txn.access.Unlock()

	kvs := map[string] /*key*/ datasync.ChangeValue{}
	for key, item := range txn.items {
		changeType := datasync.Put
		if item.Delete {
			changeType = datasync.Delete
		}

		kvs[key] = syncbase.NewChangeBytes(key, item.Data, 0, changeType)
	}
	return txn.commit(kvs)
}

// BytesTxnItem is used in BytesTxn.
type BytesTxnItem struct {
	Data   []byte
	Delete bool
}
