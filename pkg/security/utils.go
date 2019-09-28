// Copyright (c) 2019 Cisco and/or its affiliates.
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

package security

import (
	"crypto/x509"
	"encoding/pem"
)

func certToPemBlocks(data []byte) ([]byte, error) {
	certs, err := x509.ParseCertificates(data)
	if err != nil {
		return nil, err
	}

	pemData := []byte{}
	for _, cert := range certs {
		b := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}

	return pemData, nil
}

func keyToPem(data []byte) []byte {
	b := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: data,
	}
	return pem.EncodeToMemory(b)
}
