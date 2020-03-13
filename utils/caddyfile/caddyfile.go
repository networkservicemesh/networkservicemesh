// Package caddyfile provides API for creating and editing caddyfile
package caddyfile

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

//Caddyfile provides API for creating and editing caddyfile
type Caddyfile interface {
	Scope
	Save() error
}

type caddyfile struct {
	Scope
	pathToFile string
}

//NewCaddyfile creates new instance of Caddyfile
func NewCaddyfile(path string) Caddyfile {
	return &caddyfile{
		Scope:      newCaddyfileScope("", nil),
		pathToFile: path,
	}
}

//Save saves the content of caddyfile on disk
func (c *caddyfile) Save() error {
	_, err := os.Stat(c.pathToFile)
	if os.IsNotExist(err) {
		f, err := os.Create(c.pathToFile)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
	}
	return ioutil.WriteFile(c.pathToFile, []byte(c.String()), 0)
}

//String converts caddyfile to string
func (c *caddyfile) String() string {
	sb := strings.Builder{}
	for _, scope := range c.Records() {
		_, _ = sb.WriteString(scope.String())
		_, _ = sb.WriteRune('\n')
	}
	return sb.String()
}

//Scope is part of caddyfile, represented as a block of records
type Scope interface {
	fmt.Stringer
	Write(string) Scope
	WriteScope(string) Scope
	Up() Scope
	HasScope(string) bool
	GetOrCreate(string) Scope
	Records() []fmt.Stringer
	Remove(string)
	Name() string
}

type caddyfileScope struct {
	records []fmt.Stringer
	scopes  map[string]Scope
	name    string
	parent  Scope
}

//Write writes new record in caddyfile
func (c *caddyfileScope) Write(str string) Scope {
	c.records = append(c.records, record(str))
	return c
}

//Write writes new scope in caddyfile
func (c *caddyfileScope) WriteScope(name string) Scope {
	scope := newCaddyfileScope(name, c)
	c.scopes[name] = scope
	c.records = append(c.records, scope)
	return scope
}

//Up returns parent of current scope. Can return nil if c is root
func (c *caddyfileScope) Up() Scope {
	return c.parent
}

//Records returns all records in scope
func (c *caddyfileScope) Records() []fmt.Stringer {
	return c.records
}

//Name returns name of scope
func (c *caddyfileScope) Name() string {
	return c.name
}

//GetOrCreate creates new scope if scope with name does not exist or returns exists scope.
func (c *caddyfileScope) GetOrCreate(name string) Scope {
	if _, ok := c.scopes[name]; !ok {
		c.scopes[name] = newCaddyfileScope(name, c)
	}
	return c.scopes[name]
}

//Remove removes record from current scope
func (c *caddyfileScope) Remove(name string) {
	delete(c.scopes, name)

	removeIndex := indexOf(c.records, record(name))

	if removeIndex == -1 {
		return
	}
	c.records = append(c.records[:removeIndex], c.records[removeIndex+1:]...)
}

//String converts scope to string
func (c *caddyfileScope) String() string {
	sb := strings.Builder{}
	_, _ = sb.WriteString(c.name)
	_, _ = sb.WriteString(" {\n")
	tab := strings.Repeat("\t", c.level())
	for _, scope := range c.records {
		_, _ = sb.WriteString(tab)
		_, _ = sb.WriteString(scope.String())
		_, _ = sb.WriteString("\n")
	}
	if tab != "" {
		_, _ = sb.WriteString(tab[:len(tab)-1])
	}
	_, _ = sb.WriteString("}")
	return sb.String()
}

//HasScope returns true if current scope contain specific scope
func (c *caddyfileScope) HasScope(name string) bool {
	return c.scopes[name] != nil
}

func (c *caddyfileScope) level() int {
	lvl := 0
	p := c
	for p.parent != nil {
		lvl++
		p = p.parent.(*caddyfileScope)
	}
	return lvl
}
func indexOf(records []fmt.Stringer, rec fmt.Stringer) int {
	name := nameOf(rec)
	removeIndex := -1
	for i, rec := range records {
		if nameOf(rec) == name {
			removeIndex = i
			break
		}
	}
	return removeIndex
}

func nameOf(s fmt.Stringer) string {
	if v, ok := s.(Scope); ok {
		return v.Name()
	}
	return s.String()
}

func newCaddyfileScope(name string, parent Scope) Scope {
	return &caddyfileScope{
		records: make([]fmt.Stringer, 0),
		scopes:  make(map[string]Scope),
		name:    name,
		parent:  parent,
	}
}
