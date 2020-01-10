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

package artifact

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

func Archivator(c Config) Hook {
	return &artifactArchivator{c: c}
}

type artifactArchivator struct {
	c      Config
	buff   *bytes.Buffer
	writer *zip.Writer
	m      sync.Mutex
}

func (aa *artifactArchivator) Present(a Artifact) {
}

func (aa *artifactArchivator) Started() {
	aa.buff = new(bytes.Buffer)
	aa.writer = zip.NewWriter(aa.buff)
}

func (aa *artifactArchivator) Finished() {
	dir, _ := path.Split(aa.c.OutputPath())
	if dir != "" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			_ = os.MkdirAll(dir, os.ModePerm)
		}
	}
	outfile := filepath.Join(aa.c.OutputPath() + ".zip")
	if _, err := os.Stat(outfile); err == nil {
		zr, err := zip.OpenReader(outfile)
		if err != nil {
			logrus.Errorf("Can not zip write, err: %v", err)
		} else {
			distill(aa.writer, zr)
			_ = zr.Close()
		}
	}

	err := aa.writer.Flush()
	if err != nil {
		logrus.Errorf("An error during writer.Flush(), err: %v", err)
	}
	err = aa.writer.Close()
	if err != nil {
		logrus.Errorf("An error during zip writer.Close(), err: %v", err)
	}
	err = ioutil.WriteFile(outfile, aa.buff.Bytes(), os.ModePerm)
	if err != nil {
		logrus.Errorf("An error during zip file.Close(), err: %v", err)
	}
}

func (aa *artifactArchivator) PostProcess(a Artifact) {
	aa.m.Lock()
	defer aa.m.Unlock()
	writer, err := aa.writer.Create(a.Name())
	if err != nil {
		logrus.Errorf("Can not create zip writer for artifact %v, error: %v", a.Name(), err)
	}
	_, err = writer.Write(a.Content())
	if err != nil {
		logrus.Errorf("An error during write artifact content %v, error: %v", a.Name(), err)
	}
}

func (aa *artifactArchivator) PreProcess(a Artifact) Artifact {
	return a
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
