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
)

const (
	// NSMkeyNamespace defines the name of the key namespace in parameters map (mandatory)
	NSMkeyNamespace = "namespace"
	// NSMkeyIPv4 defines the name of the key ipv4 address in parameters map (optional)
	NSMkeyIPv4 = "ipv4"
	// NSMkeyIPv4PrefixLength defines the name of the key ipv4 prefix length in parameters map (optional)
	NSMkeyIPv4PrefixLength = "ipv4prefixlength"
	//NSMSocketFile defines socket name which will be used for memif connection
	NSMSocketFile = "socketfile"
	//NSMMaster if true, than role is master
	NSMMaster = "master"
	//NSMSlave if true, than role is slave
	NSMSlave = "slave"
	//NSMPerPodDirectory defines directory that is mounted to pod (relative to /var/lib/networkservicemesh)
	NSMPerPodDirectory = "directory"
)

// Keys is a map of all Keys which are supported in the connection Parameters map
type KeyProperties struct {
	Mandatory bool
	Validator func(value string) error
}
type Keys map[string]KeyProperties

// ValidateParameters checks all required amd optional parameters
// and attempts to check them
func ValidateParameters(parameters map[string]string, keyList Keys) error {
	// Check for any Unknown Keys if found return error
	for key := range parameters {
		if _, ok := keyList[key]; !ok {
			// Found a key in parameters which is not in the list of supported Keys
			return fmt.Errorf("found unknown key %s in the parameters", key)
		}
	}

	// Check mandatory parameters first
	for key, properties := range keyList {
		if properties.Mandatory {
			if _, ok := parameters[key]; !ok {
				return fmt.Errorf("missing mandatory %s key", key)
			}
		}
	}

	// Check sanity for all passed parameters
	for key, value := range parameters {
		if err := keyList[key].Validator(value); err != nil {
			return fmt.Errorf("key %s has invalid value %s, error: %+v", key, value, err)
		}
	}

	return nil
}

// Keys validator functions, for each new Keys there should be a validator function.
func Any(value string) error {
	return nil
}

func Ipv4(value string) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return fmt.Errorf("invalid value %s of ipv4 parameter", value)
	}
	// TODO (sbezverk) It will pass for both ipv4 and ipv6 addresses
	// need to add a function to differentiate
	return nil
}

func Ipv4prefixlength(value string) error {
	prefixLength, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if !(prefixLength > 1 && prefixLength < 32) {
		return fmt.Errorf("invalid value %d of ipv4 prefix parameter", prefixLength)
	}
	return nil
}

func Empty(value string) error {
	return nil
}
func Bool(value string) error {
	if _, err := strconv.ParseBool(value); err != nil {
		return err
	}
	return nil
}
