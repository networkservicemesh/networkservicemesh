package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type record string

func (r record) String() string {
	return string(r)
}

func NewCorefileScope(name string, parent CorefileScope) CorefileScope {
	return &corefileScope{
		records: make([]fmt.Stringer, 0),
		scopes:  make(map[string]CorefileScope),
		name:    name,
		parent:  parent,
	}
}

type Corefile interface {
	CorefileScope
	Save()
}

type corefile struct {
	CorefileScope
	pathToFile string
}

func NewCorefile(path string) Corefile {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			os.Create(path)
		} else {
			panic(err)
		}
	}
	return &corefile{
		CorefileScope: NewCorefileScope("", nil),
		pathToFile:    path,
	}
}

func (c *corefile) Save() {
	err := ioutil.WriteFile(c.pathToFile, []byte(c.String()), 0)
	if err != nil {
		panic(err)
	}
}
func (c *corefile) String() string {
	sb := strings.Builder{}
	for _, scope := range c.Records() {
		sb.WriteString(scope.String())
		sb.WriteRune('\n')
	}
	return sb.String()
}

type CorefileScope interface {
	fmt.Stringer
	Write(string) CorefileScope
	WriteScope(string) CorefileScope
	Up() CorefileScope
	Scope(string) CorefileScope
	Records() []fmt.Stringer
	Remove(string)
}

type corefileScope struct {
	records []fmt.Stringer
	scopes  map[string]CorefileScope
	name    string
	parent  CorefileScope
}

func (c *corefileScope) Write(str string) CorefileScope {
	c.records = append(c.records, record(str))
	return c
}

func (c *corefileScope) WriteScope(name string) CorefileScope {
	scope := NewCorefileScope(name, c)
	c.scopes[name] = scope
	c.records = append(c.records, scope)
	return scope
}
func (c *corefileScope) Up() CorefileScope {
	return c.parent
}
func (c *corefileScope) Records() []fmt.Stringer {
	return c.records
}

func (c *corefileScope) Scope(name string) CorefileScope {
	if _, ok := c.scopes[name]; !ok {
		c.scopes[name] = NewCorefileScope(name, c)
	}
	return c.scopes[name]
}

func (c *corefileScope) Remove(name string) {
	delete(c.scopes, name)
}

func (c *corefileScope) String() string {
	sb := strings.Builder{}
	sb.WriteString(c.name)
	sb.WriteString(" {\n")
	tab := strings.Repeat("\t", c.level())
	for _, scope := range c.records {
		sb.WriteString(tab)
		sb.WriteString(scope.String())
		sb.WriteString("\n")
	}
	if tab != "" {
		sb.WriteString(tab[:len(tab)-1])
	}
	sb.WriteString("}")
	return sb.String()
}

func (c *corefileScope) level() int {
	lvl := 0
	p := c
	for p.parent != nil {
		lvl++
		p = p.parent.(*corefileScope)
	}
	return lvl
}
