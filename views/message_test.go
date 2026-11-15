package views

import (
	"errors"
	"testing"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
)

// The service's message is used as it is, but names the view when the
// service's does not.
func TestViewMessage(t *testing.T) {
	cases := []struct {
		name string
		view string
		err  error
		want string
	}{
		{
			"the service does not name the view",
			"V::sys",
			&opensysml.StatusError{Code: opensysml.CodeInvalidArgument, Message: "not a view"},
			"V::sys: not a view",
		},
		{
			"the service names the view",
			"V::partsTable",
			&opensysml.StatusError{Code: opensysml.CodeInvalidArgument,
				Message: "V::partsTable: a table rendering is not written as dot"},
			"V::partsTable: a table rendering is not written as dot",
		},
		{
			"another error is kept as it is",
			"V::sys",
			errors.New("boom"),
			"boom",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := viewMessage(c.err, c.view).Error(); got != c.want {
				t.Errorf("message %q, want %q", got, c.want)
			}
		})
	}
}
