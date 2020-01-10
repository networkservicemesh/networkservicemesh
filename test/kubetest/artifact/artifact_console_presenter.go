// Copyright (c) 2019-2020 Cisco Systems, Inc.
//
// SPDX-License-Identifier: Apache-2.0
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

package artifact

import (
	"fmt"
	"math"
	"strings"

	"github.com/sirupsen/logrus"
)

type consolePresnter struct {
}

func (c *consolePresnter) Present(artifact Artifact) {
	logTransaction(artifact.Name(), artifact.Content())
}

func logTransaction(name string, content []byte) {
	drawer := transactionWriter{
		buff:       strings.Builder{},
		lineLength: maxTransactionLineWidth,
		drawUnit:   transactionLogUnit,
	}
	drawer.writeLine()
	drawer.writeLineWithText(startLogsOf + " " + name)
	drawer.writeLine()
	drawer.writeBytes(content)
	drawer.writeLine()
	drawer.writeLineWithText(endLogsOf + " " + name)
	drawer.writeLine()
	prettyPrintln(drawer.buff.String())
}

func prettyPrintln(text string) {
	f := logrus.StandardLogger().Formatter
	logrus.SetFormatter(&innerLogFormatter{})
	logrus.Println(text)
	logrus.SetFormatter(f)
}

type transactionWriter struct {
	buff       strings.Builder
	lineLength int
	drawUnit   rune
}

func (t *transactionWriter) writeBytes(bytes []byte) {
	_, _ = t.buff.Write(bytes)
	_, _ = t.buff.WriteRune('\n')
}

func (t *transactionWriter) writeLine() {
	_, _ = t.buff.WriteString(strings.Repeat(string(t.drawUnit), t.lineLength))
	_, _ = t.buff.WriteRune('\n')
}
func (t *transactionWriter) writeLineWithText(test string) {
	sideWidth := int(math.Max(float64(t.lineLength-len(test)), 0)) / 2
	for i := 0; i < sideWidth; i++ {
		_, _ = t.buff.WriteRune(t.drawUnit)
	}
	_, _ = t.buff.WriteString(test)
	for i := sideWidth + len(test); i < maxTransactionLineWidth; i++ {
		_, _ = t.buff.WriteRune(t.drawUnit)
	}
	_, _ = t.buff.WriteRune('\n')
}

type innerLogFormatter struct {
}

func (*innerLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("[%v] %v\n%v", entry.Level.String(), entry.Time, entry.Message)), nil
}
