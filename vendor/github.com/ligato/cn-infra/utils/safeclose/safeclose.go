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

package safeclose

import (
	"errors"
	"io"
	"reflect"

	"github.com/ligato/cn-infra/logging/logrus"
)

// CloserWithoutErr is similar interface to GoLang Closer but Close() does not return error
type CloserWithoutErr interface {
	Close()
}

// Close closes closable I/O stream.
func Close(obj interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			logrus.DefaultLogger().Error("Recovered in safeclose: ", r)
		}
	}()

	if reflect.ValueOf(obj).IsValid() && !reflect.ValueOf(obj).IsNil() {
		if closer, ok := obj.(*io.Closer); ok {
			if closer != nil {
				err := (*closer).Close()
				return err
			}
		} else if closer, ok := obj.(*CloserWithoutErr); ok {
			if closer != nil {
				(*closer).Close()
			}
		} else if closer, ok := obj.(io.Closer); ok {
			if closer != nil {
				logrus.DefaultLogger().Debug("closer: ", closer)
				err := closer.Close()
				return err
			}
		} else if closer, ok := obj.(CloserWithoutErr); ok {
			if closer != nil {
				closer.Close()
			}
		} else if reflect.TypeOf(obj).Kind() == reflect.Chan {
			//reflect.ValueOf(nil).

			if x, ok := obj.(chan interface{}); ok {
				close(x)
			}
		}

	}
	return nil
}

// CloseAll tries to close all objects and return all errors (there are nils if there was no errors).
func CloseAll(objs ...interface{}) (details []error, errOccured error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.DefaultLogger().Error("Recovered in safeclose: ", r)
		}
	}()

	details = make([]error, len(objs))
	for i, obj := range objs {
		details[i] = Close(obj)
	}

	if errOccured != nil {
		return details, format(details)
	}

	return details, nil
}

// format squashes multiple errors into one.
func format(errs []error) error {
	errMsg := ""

	for _, err := range errs {
		if err != nil {
			errMsg += ";" + err.Error()
		}
	}

	if len(errMsg) > 0 {
		return errors.New(errMsg)
	}
	return nil
}
