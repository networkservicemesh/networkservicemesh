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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
)

func TestManager_ProcessArtifacts(t *testing.T) {
	assert := gomega.NewWithT(t)
	artifactPath := path.Join(os.TempDir(), t.Name())
	assert.Expect(os.MkdirAll(artifactPath, os.ModePerm)).Should(gomega.BeNil())
	defer func() {
		_ = os.Remove(artifactPath)
	}()
	outputDirectory.Set(artifactPath)
	saveInAnyCase.Set(true)
	saveAsFiles.Set(true)
	c := ConfigFromEnv()
	assert.Expect(c.SaveOption()).Should(gomega.Equal(SaveAsFiles))
	assert.Expect(c.OutputPath()).Should(gomega.Equal(artifactPath))
	assert.Expect(c.SaveInAnyCase()).Should(gomega.Equal(true))
	m := NewManager(c, DefaultPresenterFactory(), []Finder{&testFinder{}}, []Hook{&testHook{}})
	m.SaveArtifacts()
	files, err := ioutil.ReadDir(artifactPath)
	assert.Expect(err).Should(gomega.BeNil())
	assert.Expect(len(files)).Should(gomega.Equal(2))
	assert.Expect(files[0].Name() == "A")
	assert.Expect(files[1].Name() == "B")
	filePath1 := path.Join(artifactPath, files[0].Name())
	bytes, err := ioutil.ReadFile(filepath.Clean(filePath1))
	assert.Expect(err).Should(gomega.BeNil())
	assert.Expect(string(bytes)).Should(gomega.Equal("changed"))
	filePath2 := path.Join(artifactPath, files[1].Name())
	bytes, err = ioutil.ReadFile(filepath.Clean(filePath2))
	assert.Expect(err).Should(gomega.BeNil())
	assert.Expect(string(bytes)).Should(gomega.Equal("{}"))
}

type testFinder struct {
}

func (t *testFinder) Find() []Artifact {
	return []Artifact{
		New("A", "log", []byte("content1")),
		New("B", "json", []byte("{}")),
		New("C", "bin", []byte("...")),
	}
}

type testHook struct {
}

func (t testHook) OnPresent(a Artifact) Artifact {
	if a.Kind() == "log" {
		return modifyContent(a, []byte("changed"))
	}
	if a.Kind() == "bin" {
		return nil
	}
	return a
}

func (t testHook) OnPresented(Artifact) {
}
func (t testHook) OnStart() {
}
func (t testHook) OnFinish() {
}
func modifyContent(a Artifact, newContent []byte) Artifact {
	return New(a.Name(), a.Kind(), newContent)
}
