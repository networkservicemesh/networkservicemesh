package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type resolvConfFile struct {
	path string
}

type resolvConfProperty struct {
	name   string
	values []string
}

func (r resolvConfFile) Searches() []string {
	return r.readAllByProperty(searchProperty)
}

func (r resolvConfFile) Nameservers() []string {
	return r.readAllByProperty(nameserverProperty)
}

func (r resolvConfFile) Options() []string {
	return r.readAllByProperty(optionsProperty)
}

func (r resolvConfFile) ReplaceProperties(properties []resolvConfProperty) {
	sb := strings.Builder{}
	for _, property := range properties {
		if len(property.values) == 0 {
			continue
		}
		_, _ = sb.WriteString(property.name)
		_, _ = sb.WriteString(" ")
		_, _ = sb.WriteString(strings.Join(property.values, " "))
		_, _ = sb.WriteString("\n")
	}
	err := ioutil.WriteFile(r.path, []byte(sb.String()), os.ModePerm)
	if err != nil {
		log.Printf("An error during write file. Path: %v, err: %v\n", r.path, err.Error())
	}
}

func (r resolvConfFile) readAllByProperty(propertyKey string) []string {
	bytes, err := ioutil.ReadFile(r.path)
	if err != nil {
		log.Printf("An error during ioutil.ReadFile... Path: %v, err: %v\n", r.path, err.Error())
	}
	source := string(bytes)
	var result []string
	for _, line := range strings.Split(source, "\n") {
		if strings.HasPrefix(line, propertyKey) {
			result = append(result, strings.Split(line[len(propertyKey)+1:], " ")...)
		}
	}
	return result
}
