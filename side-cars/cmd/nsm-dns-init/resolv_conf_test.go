package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func createSample(name, source string) (string, error) {
	tmpPath := path.Join(os.TempDir(), name)
	err := ioutil.WriteFile(tmpPath, []byte(source), os.ModePerm)
	if err != nil {
		return "", err
	}
	return tmpPath, nil
}

func TestResolvConfFileReadProperties(t *testing.T) {
	sampleSource := `nameserver 127.0.0.1
search default.svc.cluster.local svc.cluster.local cluster.local
options ndots:5
`
	path, err := createSample("resolv.conf.tmp.test.file", sampleSource)
	if err != nil {
		println(err.Error())
		t.FailNow()
	}
	defer func() {
		_ = os.Remove(path)
	}()
	conf := resolvConfFile{path: path}
	result := conf.Nameservers()
	if len(result) != 1 {
		t.FailNow()
	}
	if result[0] != "127.0.0.1" {
		t.FailNow()
	}
	result = conf.Searches()
	if len(result) != 3 {
		t.FailNow()
	}
	if result[0] != "default.svc.cluster.local" {
		t.FailNow()
	}
	if result[1] != "svc.cluster.local" {
		t.FailNow()
	}
	if result[2] != "cluster.local" {
		t.FailNow()
	}
	result = conf.Options()
	if len(result) != 1 {
		t.FailNow()
	}
	if result[0] != "ndots:5" {
		t.FailNow()
	}
}

func TestResolvConfFileWriteProperties(t *testing.T) {
	sampleSource := `nameserver 127.0.0.1
search default.svc.cluster.local svc.cluster.local cluster.local
options ndots:5
`
	path, err := createSample("resolv.conf.tmp.test.file", "")
	if err != nil {
		println(err.Error())
		t.FailNow()
	}
	defer func() {
		_ = os.Remove(path)
	}()
	conf := resolvConfFile{path: path}
	properties := []resolvConfProperty{
		{nameserverProperty, []string{"127.0.0.1"}},
		{searchProperty, []string{"default.svc.cluster.local", "svc.cluster.local", "cluster.local"}},
		{optionsProperty, []string{"ndots:5"}},
	}
	conf.ReplaceProperties(properties)

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		println(err.Error())
		t.FailNow()
	}
	actual := string(bytes)
	if actual != sampleSource {
		println(actual)
		println(sampleSource)
		t.FailNow()
	}
}
