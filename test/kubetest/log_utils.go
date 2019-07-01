package kubetest

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"math"
	"os"
	"path/filepath"
	"strings"
)

func LogsDir() string {
	logDir := DefaultLogDir
	if dir, ok := os.LookupEnv(WritePodLogsDir); ok {
		logDir = dir
	}
	return logDir
}

func LogInFiles() bool {
	if v, ok := os.LookupEnv(WritePodLogsInFile); ok {
		if v == "true" {
			return true
		}
	}
	return false
}

func LogFile(name, dir, content string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
	path := filepath.Join(dir, name)
	var _, err = os.Stat(path)
	if os.IsExist(err) {
		os.Remove(path)
	}
	file, err := os.Create(name)
	file.WriteString(content)
	err = file.Close()
	if err != nil {
		panic(err)
	}
}

func LogTransaction(name, content string) {
	f := logrus.StandardLogger().Formatter
	logrus.SetFormatter(&innerLogFormatter{})

	drawer := transactionDrawer{
		buff:       strings.Builder{},
		lineLength: MaxTransactionLineWidth,
		drawUnit:   TransactionLogUnit,
	}
	drawer.drawLine()
	drawer.drawLineWithName(StartLogsOf + name)
	drawer.drawLine()
	drawer.drawText(content)
	drawer.drawLine()
	drawer.drawLineWithName(EndLogsOf + name)
	drawer.drawLine()
	logrus.Println(drawer.buff.String())
	logrus.SetFormatter(f)
}

type transactionDrawer struct {
	buff       strings.Builder
	lineLength int
	drawUnit   rune
}

func (t *transactionDrawer) drawText(text string) {
	t.buff.WriteString(text)
}

func (t *transactionDrawer) drawLine() {
	t.buff.WriteString(strings.Repeat(string(t.drawUnit), t.lineLength))
	t.buff.WriteRune('\n')
}
func (t *transactionDrawer) drawLineWithName(name string) {
	sideWidth := int(math.Max(float64(t.lineLength-len(name)), 0)) / 2
	for i := 0; i < sideWidth; i++ {
		t.buff.WriteRune(t.drawUnit)
	}
	t.buff.WriteString(name)
	for i := 0; i < sideWidth; i++ {
		t.buff.WriteRune(t.drawUnit)
	}
	t.buff.WriteRune('\n')
}

type innerLogFormatter struct {
}

func (*innerLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("[%v] %v\n%v", entry.Level.String(), entry.Time, entry.Message)), nil
}
