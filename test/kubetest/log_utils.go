package kubetest

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"math"
	"os"
	"path/filepath"
	"strings"
)

//LogsDir - returns dir where contains logs
func LogsDir() string {
	logDir := DefaultLogDir
	if dir, ok := os.LookupEnv(WritePodLogsDir); ok {
		logDir = dir
	}
	return logDir
}

//LogInFiles - returns if the logs from pods should be stored as files
func LogInFiles() bool {
	if v, ok := os.LookupEnv(WritePodLogsInFile); ok {
		if v == "true" {
			return true
		}
	}
	return false
}

//LogFile - saves logs in specific dir as file
func LogFile(name, dir, content string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	path := filepath.Join(dir, name)
	var _, err = os.Stat(path)
	if os.IsExist(err) {
		err = os.Remove(path)
		if err != nil {
			return err
		}
	}
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	_, err = file.WriteString(content)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}
	return nil
}

//LogTransaction - writes in log transaction with name and specific content
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
	_, _ = t.buff.WriteString(text)
}

func (t *transactionDrawer) drawLine() {
	_, _ = t.buff.WriteString(strings.Repeat(string(t.drawUnit), t.lineLength))
	_, _ = t.buff.WriteRune('\n')
}
func (t *transactionDrawer) drawLineWithName(name string) {
	sideWidth := int(math.Max(float64(t.lineLength-len(name)), 0)) / 2
	for i := 0; i < sideWidth; i++ {
		_, _ = t.buff.WriteRune(t.drawUnit)
	}
	_, _ = t.buff.WriteString(name)
	for i := 0; i < sideWidth; i++ {
		_, _ = t.buff.WriteRune(t.drawUnit)
	}
	_, _ = t.buff.WriteRune('\n')
}

type innerLogFormatter struct {
}

func (*innerLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("[%v] %v\n%v", entry.Level.String(), entry.Time, entry.Message)), nil
}
