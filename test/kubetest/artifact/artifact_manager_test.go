// Copyright (c) 2019 Cisco Systems, Inc.
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
	"io/ioutil"
	"os"
	"path"
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
	dir.Set(artifactPath)
	processInAnyCase.Set(true)
	processInToDir.Set(true)
	c := ConfigFromEnv()
	assert.Expect(c.SaveBehavior()).Should(gomega.Equal(SaveAsDir))
	assert.Expect(c.OutputPath()).Should(gomega.Equal(artifactPath))
	assert.Expect(c.SaveInAnyCase()).Should(gomega.Equal(true))
	m := NewManager(c, DefaultPresenterFactory(), []Finder{&testFinder{}}, []Hook{&testHook{}})
	m.ProcessArtifacts()
	files, err := ioutil.ReadDir(artifactPath)
	assert.Expect(err).Should(gomega.BeNil())
	assert.Expect(len(files)).Should(gomega.Equal(2))
	assert.Expect(files[0].Name() == "A")
	assert.Expect(files[1].Name() == "B")
	bytes, err := ioutil.ReadFile(path.Join(artifactPath, files[0].Name()))
	assert.Expect(err).Should(gomega.BeNil())
	assert.Expect(string(bytes)).Should(gomega.Equal("changed"))
	bytes, err = ioutil.ReadFile(path.Join(artifactPath, files[1].Name()))
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

func (t testHook) PreProcess(a Artifact) Artifact {
	if a.Kind() == "log" {
		return ModifyContent(a, []byte("changed"))
	}
	if a.Kind() == "bin" {
		return nil
	}
	return a
}

func (t testHook) PostProcess(Artifact) {
}
func (t testHook) Started() {
}
func (t testHook) Finished() {
}
