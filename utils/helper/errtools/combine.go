package errtools

import (
	"errors"
	"fmt"
	"strings"
)

//Combine means collect a array of errors to the error
func Combine(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	sb := strings.Builder{}
	id := 0
	for i, err := range errs {
		if err == nil {
			continue
		}
		id++
		_, _ = sb.WriteString(fmt.Sprintf("%v. %v", id, err.Error()))

		if i+1 != len(errs) {
			_, _ = sb.WriteRune('\n')
		}
	}
	if sb.Len() == 0 {
		return nil
	}
	return errors.New(sb.String())
}
