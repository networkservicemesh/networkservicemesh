package errtools

import "github.com/pkg/errors"

func Collect(errs ...error) error {
	var result error

	for _, err := range errs {
		if err == nil {
			continue
		}
		if result == nil {
			result = err
			continue
		}
		result = errors.Wrap(result, err.Error())
	}

	return result
}
