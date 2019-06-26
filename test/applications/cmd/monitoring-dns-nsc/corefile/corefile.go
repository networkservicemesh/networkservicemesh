package corefile

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
	Save() error
}

type corefile struct {
	CorefileScope
	pathToFile string
}

func NewCorefile(path string) Corefile {
	return &corefile{
		CorefileScope: NewCorefileScope("", nil),
		pathToFile:    path,
	}
}

func (c *corefile) Save() error {
	_, err := os.Stat(c.pathToFile)
	if os.IsNotExist(err) {
		_, err = os.Create(c.pathToFile)
		if err != nil {
			return err
		}
	}
	err = ioutil.WriteFile(c.pathToFile, []byte(c.String()), 0)
	if err != nil {
		return err
	}
	return nil
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
	Name() string
	Prioritize() CorefileScope
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
func (c *corefileScope) Name() string {
	return c.name
}

func (c *corefileScope) Scope(name string) CorefileScope {
	if _, ok := c.scopes[name]; !ok {
		c.scopes[name] = NewCorefileScope(name, c)
	}
	return c.scopes[name]
}
func (c *corefileScope) Prioritize() CorefileScope {
	index := indexOf(c.parent.Records(), c)
	if index == -1 {
		return c
	}
	pop := c.parent.Records()[0]
	c.parent.Records()[index] = pop
	c.parent.Records()[0] = c
	return c
}

func (c *corefileScope) Remove(name string) {
	delete(c.scopes, name)

	removeIndex := indexOf(c.records, record(name))

	if removeIndex == -1 {
		return
	}
	c.records = append(c.records[:removeIndex], c.records[removeIndex+1:]...)
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
	if v, ok := s.(CorefileScope); ok {
		return v.Name()
	}
	return s.String()
}
