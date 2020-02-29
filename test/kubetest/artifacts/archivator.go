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
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

//Archivator hook for artifacts manager. Archives artifact content.
func Archivator(c Config) Hook {
	return &archivator{c: c}
}

type archivator struct {
	c      Config
	buff   *bytes.Buffer
	writer *zip.Writer
	m      sync.Mutex
}

func (a *archivator) Present(artifact Artifact) {
}

func (a *archivator) OnStart() {
	a.buff = new(bytes.Buffer)
	a.writer = zip.NewWriter(a.buff)
}

func (a *archivator) OnFinish() {
	if _, err := os.Stat(a.c.OutputPath()); os.IsNotExist(err) {
		_ = os.MkdirAll(a.c.OutputPath(), os.ModePerm)
	}
	_, name := path.Split(a.c.OutputPath())
	outfile := filepath.Join(a.c.OutputPath(), name+".zip")
	if _, err := os.Stat(outfile); err == nil {
		zr, err := zip.OpenReader(outfile)
		if err != nil {
			logrus.Errorf("Can not zip write, err: %v", err)
		} else {
			distill(a.writer, zr)
			_ = zr.Close()
		}
	}

	err := a.writer.Flush()
	if err != nil {
		logrus.Errorf("An error during writer.Flush(), err: %v", err)
	}
	err = a.writer.Close()
	if err != nil {
		logrus.Errorf("An error during zip writer.Close(), err: %v", err)
	}
	err = ioutil.WriteFile(outfile, a.buff.Bytes(), os.ModePerm)
	if err != nil {
		logrus.Errorf("An error during zip file.Close(), err: %v", err)
	}
}

func (a *archivator) OnPresented(artifact Artifact) {
	a.m.Lock()
	defer a.m.Unlock()
	writer, err := a.writer.Create(artifact.Name())
	if err != nil {
		logrus.Errorf("Can not create zip writer for artifact %v, error: %v", artifact.Name(), err)
	}
	_, err = writer.Write(artifact.Content())
	if err != nil {
		logrus.Errorf("An error during write artifact content %v, error: %v", artifact.Name(), err)
	}
}

func (a *archivator) OnPresent(artifact Artifact) Artifact {
	return artifact
}

func distill(w *zip.Writer, r *zip.ReadCloser) {
	for _, f := range r.File {
		header := f.FileHeader
		hw, err := w.CreateHeader(&header)
		if err != nil {
			logrus.Errorf("An error during create zip header, err: %v", err)
			continue
		}
		fr, err := f.Open()
		if err != nil {
			logrus.Errorf("An error during open zip file, err: %v", err)
			continue
		}
		_, err = io.Copy(hw, fr)
		if err != nil {
			logrus.Errorf("An error during io.Copy(...), err: %v", err)
		}
	}
}
