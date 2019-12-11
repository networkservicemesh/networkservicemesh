package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func AllFiles(dir string) []string {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Printf("An error during scanning dir: %v. Error: %v", dir, err.Error())
		return nil
	}
	return files
}
func main() {
	files := AllFiles("../")
	filter := []string{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			filter = append(filter, f)
		}
	}
	for _, f := range filter {
		println(f)
	}
}
