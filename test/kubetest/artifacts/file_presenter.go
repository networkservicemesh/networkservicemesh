// Copyright (c) 2019-2020 Cisco Systems, Inc.
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

package artifacts

import (
	"os"
	"path"

	"github.com/sirupsen/logrus"
)

type filePresenter struct {
	path string
}

func (f *filePresenter) Present(artifact Artifact) {
	name := artifact.Name()
	bytes := artifact.Content()
	if _, err := os.Stat(f.path); os.IsNotExist(err) {
		err = os.MkdirAll(f.path, os.ModePerm)
		if err != nil {
			logrus.Error(err.Error())
		}
	}
	filePath := path.Join(f.path, name)
	file, err := os.Create(filePath)
	if err != nil {
		logrus.Errorf("Can not save artifact:%v, in path: %v. Error: %v", name, filePath, err.Error())
		return
	}
	if _, err = file.Write(bytes); err != nil {
		logrus.Errorf("An error during write to file: %v. Error: %v", filePath, err)
	}
	if err = file.Close(); err != nil {
		logrus.Errorf("An error during close file: %v. Error: %v", filePath, err)
	}
}
