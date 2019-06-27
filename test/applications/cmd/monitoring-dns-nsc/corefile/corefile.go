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

//NewCorefileScope - Creates new scope for Corefile
func NewCorefileScope(name string, parent Scope) Scope {
	return &corefileScope{
		records: make([]fmt.Stringer, 0),
		scopes:  make(map[string]Scope),
		name:    name,
		parent:  parent,
	}
}

//Corefile - API for creating and editing the Corefile.
type Corefile interface {
	Scope
	Save() error
}

type corefile struct {
	Scope
	pathToFile string
}

//NewCorefile - Creates new instance of Corefile
func NewCorefile(path string) Corefile {
	return &corefile{
		Scope:      NewCorefileScope("", nil),
		pathToFile: path,
	}
}

//Save - Saves the content of corefile on disk
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

//String - Converts corefile to string
func (c *corefile) String() string {
	sb := strings.Builder{}
	for _, scope := range c.Records() {
		_, _ = sb.WriteString(scope.String())
		_, _ = sb.WriteRune('\n')
	}
	return sb.String()
}

//GetOrCreate - GetOrCreate of corefile
type Scope interface {
	fmt.Stringer
	Write(string) Scope
	WriteScope(string) Scope
	Up() Scope
	GetOrCreate(string) Scope
	Records() []fmt.Stringer
	Remove(string)
	Name() string
	Prioritize() Scope
}

type corefileScope struct {
	records []fmt.Stringer
	scopes  map[string]Scope
	name    string
	parent  Scope
}

//Write - Writes new record in corefile
func (c *corefileScope) Write(str string) Scope {
	c.records = append(c.records, record(str))
	return c
}

//Write - Writes new scope in corefile
func (c *corefileScope) WriteScope(name string) Scope {
	scope := NewCorefileScope(name, c)
	c.scopes[name] = scope
	c.records = append(c.records, scope)
	return scope
}

//Up - Returns parent of current scope. Can return nil if c is root
func (c *corefileScope) Up() Scope {
	return c.parent
}

//Records - Returns all records in scope
func (c *corefileScope) Records() []fmt.Stringer {
	return c.records
}

//Name - Returns name of scope
func (c *corefileScope) Name() string {
	return c.name
}

//GetOrCreate - Creates new scope if scope with name does not exist or returns exists scope.
func (c *corefileScope) GetOrCreate(name string) Scope {
	if _, ok := c.scopes[name]; !ok {
		c.scopes[name] = NewCorefileScope(name, c)
	}
	return c.scopes[name]
}

//Prioritize - Move scope up
func (c *corefileScope) Prioritize() Scope {
	index := indexOf(c.parent.Records(), c)
	if index == -1 {
		return c
	}
	pop := c.parent.Records()[0]
	c.parent.Records()[index] = pop
	c.parent.Records()[0] = c
	return c
}

//Remove - Removes record from current scope
func (c *corefileScope) Remove(name string) {
	delete(c.scopes, name)

	removeIndex := indexOf(c.records, record(name))

	if removeIndex == -1 {
		return
	}
	c.records = append(c.records[:removeIndex], c.records[removeIndex+1:]...)
}

//String - Converts scope to string
func (c *corefileScope) String() string {
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
	if v, ok := s.(Scope); ok {
		return v.Name()
	}
	return s.String()
}
