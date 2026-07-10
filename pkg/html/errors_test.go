package html_test

import (
	"errors"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/html"
)

func TestParseErrorWrapsCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("bad markup")
	err := &html.ParseError{Err: cause}

	assert.Contains(t, err.Error(), "bad markup")
	assert.ErrorIs(t, err, cause)

	var empty *html.ParseError
	assert.NotEmpty(t, empty.Error(), "nil receiver must not panic")
}
