package utils

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/sirupsen/logrus"
)

// Contains - check if array contains element.
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// NewRandomStr - generates random string of desired length, size should be multiple of two for best result.
func NewRandomStr(size int) string {
	value := make([]byte, size/2)
	_, err := rand.Read(value)
	if err != nil {
		logrus.Errorf("error during random string generation %v", err)
		return ""
	}
	return hex.EncodeToString(value)
}
