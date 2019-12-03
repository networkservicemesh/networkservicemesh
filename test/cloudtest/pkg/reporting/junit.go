package reporting

import "encoding/xml"

// JUnitFile - JUnitFile
type JUnitFile struct {
	XMLName xml.Name `xml:"testsuites"`
	Suites  []*Suite
}

// Suite - Suite
type Suite struct {
	XMLName    xml.Name    `xml:"testsuite"`
	Tests      int         `xml:"tests,attr"`
	Failures   int         `xml:"failures,attr"`
	Time       string      `xml:"time,attr"`
	Name       string      `xml:"name,attr"`
	Properties []*Property `xml:"properties>property,omitempty"`
	TestCases  []*TestCase
	Suite      []*Suite
}

// TestCase - TestCase
type TestCase struct {
	XMLName     xml.Name     `xml:"testcase"`
	Classname   string       `xml:"classname,attr"`
	Name        string       `xml:"name,attr"`
	Time        string       `xml:"time,attr"`
	Cluster     string       `xml:"cluster,attr"`
	SkipMessage *SkipMessage `xml:"skipped,omitempty"`
	Failure     *Failure     `xml:"failure,omitempty"`
}

// SkipMessage - JUnitSkipMessage contains the reason why a testcase was skipped.
type SkipMessage struct {
	Message string `xml:"message,attr"`
}

// Property -  represents a key/value pair used to define properties.
type Property struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// Failure -  contains data related to a failed test.
type Failure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}
