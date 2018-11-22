package errtools

import (
	stderrors "errors"
	"github.com/go-errors/errors"
	"testing"
)

func TestWrap(t *testing.T) {
	type args struct {
		e interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "nil", args: args{nil}, wantErr: false},
		{name: "string", args: args{"hello"}, wantErr: true},
		{name: "err", args: args{stderrors.New("hello")}, wantErr: true},
		{name: "error", args: args{"hello"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Wrap(tt.args.e)
			if (err != nil) != tt.wantErr {
				t.Errorf("Wrap() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				if _, ok := err.(*errors.Error); !ok {
					t.Errorf("Error must be of type *Error")
				}
			}
		})
	}
}
