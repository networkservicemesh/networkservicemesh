package utils

import (
	"bufio"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
)

func OpenFile( root, fileName string) (string, *os.File, error) {
	// Create folder if it doesn't exists
	if !FileExists(root) {
		_ = os.MkdirAll(root, os.ModePerm)
	}
	fileName = path.Join(root, fileName)

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm )
	return fileName, f, err
}

func ReadFile( fileName string) ([]string, error) {
	// Create folder if it doesn't exists

	f, err := os.OpenFile(fileName, os.O_RDONLY, os.ModePerm )
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	output := []string{}
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		output = append(output, strings.TrimSpace(s))
	}
	return output, nil
}
func WriteFile(root, fileName, content string) {
	fileName, f, err := OpenFile(root, fileName)

	if err != nil {
		logrus.Errorf("Failed to write file: %s %v", fileName, err)
		return
	}
	_, err = f.WriteString(content)
	if err != nil {
		logrus.Errorf("Failed to write content to file, %v", err)
	}
	_ = f.Close()
}

func FileExists(root string) bool {
	_, err := os.Stat(root)
	return !os.IsNotExist(err)
}

func ClearFolder(root string, recreate bool) {
	if FileExists(root) {
		logrus.Infof("Cleaning report folder %s", root)
		_ = os.RemoveAll(root)
	}
	if recreate {
		// Create folder, since we delete is already.
		CreateFolders(root)
	}
}

func CreateFolders(root string) {
	err := os.MkdirAll(root, os.ModePerm)
	if err != nil {
		logrus.Errorf("Failed to create folder %s cause %v", root, err)
	}
}