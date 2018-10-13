package errtools

import "github.com/go-errors/errors"

func Wrap(e interface{}) error {
	if e == nil {
		return nil
	}
	return errors.Wrap(e, 1)
}
