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

package keyval

// BytesTxn allows to group operations into the transaction.
// Transaction executes multiple operations in a more efficient way in contrast
// to executing them one by one.
type BytesTxn interface {
	// Put adds put operation (write raw <data> under the given <key>) into
	// the transaction.
	Put(key string, data []byte) BytesTxn
	// Delete adds delete operation (removal of <data> under the given <key>)
	// into the transaction.
	Delete(key string) BytesTxn
	// Commit tries to execute all the operations of the transaction.
	// In the end, either all of them have been successfully applied or none
	// of them and an error is returned.
	Commit() error
}
