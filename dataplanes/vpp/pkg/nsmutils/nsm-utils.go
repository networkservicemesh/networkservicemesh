// Copyright (c) 2018 Cisco and/or its affiliates.
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

package nsmutils

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	// NSMkeyNamespace defines the name of the key namespace in parameters map (mandatory)
	NSMkeyNamespace = "namespace"
	// NSMkeyIPv4 defines the name of the key ipv4 address in parameters map (optional)
	NSMkeyIPv4 = "ipv4"
	// NSMkeyIPv4PrefixLength defines the name of the key ipv4 prefix length in parameters map (optional)
	NSMkeyIPv4PrefixLength = "ipv4prefixlength"
)

// keys is a map of all keys which are supported in the connection Parameters map
type keyProperties struct {
	mandatory bool
	validator func(value string) error
}
type keys map[string]keyProperties

// ValidateParameters checks all required amd optional parameters
// and attempts to check them
func ValidateParameters(parameters map[string]string) error {

	keyList := keys{
		NSMkeyNamespace: keyProperties{
			mandatory: true,
			validator: namespace},
		NSMkeyIPv4: keyProperties{
			mandatory: false,
			validator: ipv4},
		NSMkeyIPv4PrefixLength: keyProperties{
			mandatory: false,
			validator: ipv4prefixlength},
	}

	// Check for any Unknown keys if found return error
	for key := range parameters {
		if _, ok := keyList[key]; !ok {
			// Found a key in parameters which is not in the list of supported keys
			return fmt.Errorf("found unknown key %s in the parameters", key)
		}
	}

	// Check mandatory parameters first
	for key, properties := range keyList {
		if properties.mandatory {
			if _, ok := parameters[key]; !ok {
				return fmt.Errorf("missing mandatory %s key", key)
			}
		}
	}

	// Check sanity for all passed parameters
	for key, value := range parameters {
		if err := keyList[key].validator(value); err != nil {
			return fmt.Errorf("key %s has invalid value %s, error: %+v", key, value, err)
		}
	}
	// Check presence of both ipv4 address and prefix length
	_, v1 := parameters[NSMkeyIPv4]
	_, v2 := parameters[NSMkeyIPv4PrefixLength]
	if v1 != v2 {
		return fmt.Errorf("both parameter \"ipv4\" and \"ipv4prefixlength\" must either present or missing")
	}
	return nil
}

// keys validator functions, for each new keys there should be a validator function.
func namespace(value string) error {
	if !strings.HasPrefix(value, "pid:") {
		return fmt.Errorf("malformed namespace %s, must start with \"pid:\" following by the process id of a container", value)
	}
	return nil
}

func ipv4(value string) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return fmt.Errorf("invalid value %s of ipv4 parameter", value)
	}
	// TODO (sbezverk) It will pass for both ipv4 and ipv6 addresses
	// need to add a function to differentiate
	return nil
}

func ipv4prefixlength(value string) error {
	prefixLength, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if !(prefixLength > 1 && prefixLength < 32) {
		return fmt.Errorf("invalid value %d of ipv4 prefix parameter", prefixLength)
	}
	return nil
}
