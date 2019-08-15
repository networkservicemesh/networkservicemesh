package errtools

import (
	"fmt"
	"strings"
)

//Combine means collect a array of errors to the error
func Combine(errs ...error) error {
	var notNillErrs []error
	for _, err := range errs {
		if err == nil {
			continue
		}
		if cErr, ok := err.(*combinedError); ok {
			notNillErrs = append(notNillErrs, cErr.errs...)

		} else {
			notNillErrs = append(notNillErrs, err)
		}
	}
	if len(notNillErrs) == 0 {
		return nil
	}
	if len(notNillErrs) == 1 {
		return notNillErrs[0]
	}

	return &combinedError{notNillErrs}
}

type combinedError struct {
	errs []error
}

func (c *combinedError) Error() string {
	sb := strings.Builder{}
	id := 0
	for i, err := range c.errs {
		id++
		_, _ = sb.WriteString(fmt.Sprintf("%v. %v", id, err.Error()))

		if i+1 != len(c.errs) {
			_, _ = sb.WriteRune('\n')
		}
	}
	return sb.String()
}
