package kubetest

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/jaeger"
)

// MakeLogsSnapshot prints logs from containers in case of fail/panic or enabled logging in file
func MakeLogsSnapshot(k8s *K8s, t *testing.T) {
	if r := recover(); r != nil {
		makeLogsSnapshot(k8s, t)
		panic(r)
	} else if t.Failed() || shouldShowLogs() {
		makeLogsSnapshot(k8s, t)
	}
}
func makeLogsSnapshot(k8s *K8s, t *testing.T) {
	pods := k8s.ListPods()
	for i := 0; i < len(pods); i++ {
		showPodLogs(k8s, t, &pods[i])
	}
	if jaeger.ShouldStoreJaegerTraces() {
		for _, jaegerPod := range FindJaegerPods(k8s) {
			dir := filepath.Join(logsDir(), t.Name())
			traces := GetJaegerTraces(k8s, jaegerPod)
			for k, v := range traces {
				logFileExt(k, dir, v, "json")
			}
		}
	}
	if shouldStoreLogsInFiles() && t != nil {
		archiveLogs(t.Name())
	}
}

func archiveLogs(testName string) {
	buff := new(bytes.Buffer)
	writer := zip.NewWriter(buff)
	dir := filepath.Join(logsDir(), testName)
	logFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		logrus.Errorf("Can not read dir %v", dir)
		return
	}
	for _, file := range logFiles {
		if file.IsDir() {
			continue
		}
		filePath := filepath.Join(logsDir(), testName, file.Name())
		var bytes []byte
		bytes, err = ioutil.ReadFile(filePath)
		if err != nil {
			logrus.Errorf("Can not read file %v, err: %v", filePath, err)
			continue
		}
		var h *zip.FileHeader
		h, err = zip.FileInfoHeader(file)
		if err != nil {
			logrus.Errorf("Can not get header %v, err: %v", h, err)
			continue
		}
		h.Method = zip.Deflate
		var w io.Writer
		w, err = writer.CreateHeader(h)
		if err != nil {
			logrus.Errorf("Can not create writer, err: %v", err)
			continue
		}
		_, err = w.Write(bytes)
		if err != nil {
			logrus.Errorf("Can not zip write, err: %v", err)
		}
	}
	_ = os.RemoveAll(dir)
	outfile := filepath.Join(logsDir(), testName+".zip")
	if _, err := os.Stat(outfile); err == nil {
		zr, err := zip.OpenReader(outfile)
		if err != nil {
			logrus.Errorf("Can not zip write, err: %v", err)
		} else {
			distill(writer, zr)
			_ = zr.Close()
		}
	}

	err = writer.Flush()
	if err != nil {
		logrus.Errorf("An error during writer.Flush(), err: %v", err)
	}
	err = writer.Close()
	if err != nil {
		logrus.Errorf("An error during zip writer.Close(), err: %v", err)
	}
	err = ioutil.WriteFile(outfile, buff.Bytes(), os.ModePerm)
	if err != nil {
		logrus.Errorf("An error during zip file.Close(), err: %v", err)
	}
}

func distill(w *zip.Writer, r *zip.ReadCloser) {
	for _, f := range r.File {
		header := f.FileHeader
		hw, err := w.CreateHeader(&header)
		if err != nil {
			logrus.Errorf("An error during create zip header, err: %v", err)
			continue
		}
		fr, err := f.Open()
		if err != nil {
			logrus.Errorf("An error during open zip file, err: %v", err)
			continue
		}
		_, err = io.Copy(hw, fr)
		if err != nil {
			logrus.Errorf("An error during io.Copy(...), err: %v", err)
		}
	}
}

func showPodLogs(k8s *K8s, t *testing.T, pod *v1.Pod) {
	for i := 0; i < len(pod.Spec.Containers); i++ {
		c := &pod.Spec.Containers[i]
		savePodContainerLog(k8s, pod, c, t)
	}
	for i := 0; i < len(pod.Spec.InitContainers); i++ {
		c := &pod.Spec.Containers[i]
		savePodContainerLog(k8s, pod, c, t)
	}
}

func savePodContainerLog(k8s *K8s, pod *v1.Pod, c *v1.Container, t *testing.T) {
	name := strings.Join([]string{pod.Name, c.Name}, "-")
	logs, err := k8s.GetFullLogs(pod, c.Name, false)
	writeLogFunc := logTransaction

	if shouldStoreLogsInFiles() && t != nil {
		writeLogFunc = func(name string, content string) {
			logErr := logFile(name, filepath.Join(logsDir(), t.Name()), content)
			if logErr != nil {
				logrus.Errorf("Can't log in file, reason %v", logErr)
				logTransaction(name, content)
			} else {
				logrus.Infof("Saved log for %v. Check archive %v.zip in path %v", name, t.Name(), logsDir())
			}
		}
	}

	if err == nil {
		writeLogFunc(name, logs)
	}
	logs, err = k8s.GetFullLogs(pod, c.Name, true)
	if err == nil {
		writeLogFunc(name+"-previous", logs)
	}
}

func logsDir() string {
	return StorePodLogsDir.GetStringOrDefault(DefaultLogDir)
}

func shouldStoreLogsInFiles() bool {
	return StorePodLogsInFiles.GetBooleanOrDefault(false)
}

func shouldShowLogs() bool {
	return StoreLogsInAnyCases.GetBooleanOrDefault(false)
}

func logFile(name, dir, content string) error {
	return logFileExt(name, dir, content, "log")
}

func logFileExt(name, dir, content, ext string) error {
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
	file, err := os.Create(fmt.Sprintf("%v.%v", path, ext))
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

func logTransaction(name, content string) {
	f := logrus.StandardLogger().Formatter
	logrus.SetFormatter(&innerLogFormatter{})

	drawer := transactionWriter{
		buff:       strings.Builder{},
		lineLength: MaxTransactionLineWidth,
		drawUnit:   TransactionLogUnit,
	}
	drawer.writeLine()
	drawer.writeLineWithText(StartLogsOf + " " + name)
	drawer.writeLine()
	drawer.writeText(content)
	drawer.writeLine()
	drawer.writeLineWithText(EndLogsOf + " " + name)
	drawer.writeLine()
	logrus.Println(drawer.buff.String())
	logrus.SetFormatter(f)
}

type transactionWriter struct {
	buff       strings.Builder
	lineLength int
	drawUnit   rune
}

func (t *transactionWriter) writeText(text string) {
	_, _ = t.buff.WriteString(text)
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
	for i := sideWidth + len(test); i < MaxTransactionLineWidth; i++ {
		_, _ = t.buff.WriteRune(t.drawUnit)
	}
	_, _ = t.buff.WriteRune('\n')
}

type innerLogFormatter struct {
}

func (*innerLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("[%v] %v\n%v", entry.Level.String(), entry.Time, entry.Message)), nil
}
