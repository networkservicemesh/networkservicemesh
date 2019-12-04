package utils

import (
	"bufio"
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"
)

//ToValidFileName converts string to valid file name
func ToValidFileName(s string) string {
	if s == "" {
		return "unknown"
	}
	sb := strings.Builder{}

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' {
			_, _ = sb.WriteRune(r)
		} else {
			_, _ = sb.WriteRune('_')
		}
	}

	return sb.String()
}

// OpenFile - opens a file in folder and make folder parents if required.
func OpenFile(root, fileName string) (string, *os.File, error) {
	// Create folder if it doesn't exists
	joinedRoot := path.Join(root, fileName)
	root, fileName = path.Split(joinedRoot)
	if !FileExists(root) {
		_ = os.MkdirAll(root, os.ModePerm)
	}
	fileName = path.Join(root, fileName)

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	return fileName, f, err
}

// ReadFile - read a file contents and return as array of strings
func ReadFile(fileName string) ([]string, error) {
	// Create folder if it doesn't exists

	f, err := os.OpenFile(fileName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

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

// WriteFile - write content to file inside folder
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

// FileExists - check if file are exists.
func FileExists(root string) bool {
	_, err := os.Stat(root)
	return !os.IsNotExist(err)
}

// ClearFolder - If folder exists it will be removed with all subfolders and if recreate is passed it will be created
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

// CreateFolders - Create folder and all parents.
func CreateFolders(root string) {
	err := os.MkdirAll(root, os.ModePerm)
	if err != nil {
		logrus.Errorf("Failed to create folder %s cause %v", root, err)
	}
}
