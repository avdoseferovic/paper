// Package require provides fail-fast test assertions used by the root module.
package require

import (
	"github.com/avdoseferovic/paper/internal/assert"
)

type testingT interface {
	Errorf(format string, args ...any)
	FailNow()
}

func stop(t testingT, ok bool) bool {
	if !ok {
		t.FailNow()
	}
	return ok
}

// NoError requires err to be nil.
func NoError(t testingT, err error, msgAndArgs ...any) bool {
	return stop(t, assert.NoError(t, err, msgAndArgs...))
}

// Error requires err to be non-nil.
func Error(t testingT, err error, msgAndArgs ...any) bool {
	return stop(t, assert.Error(t, err, msgAndArgs...))
}

// ErrorIs requires err to match target.
func ErrorIs(t testingT, err, target error, msgAndArgs ...any) bool {
	return stop(t, assert.ErrorIs(t, err, target, msgAndArgs...))
}

// Equal requires expected and actual to be deeply equal.
func Equal(t testingT, expected, actual any, msgAndArgs ...any) bool {
	return stop(t, assert.Equal(t, expected, actual, msgAndArgs...))
}

// NotNil requires value to be non-nil.
func NotNil(t testingT, value any, msgAndArgs ...any) bool {
	return stop(t, assert.NotNil(t, value, msgAndArgs...))
}

// Len requires object length to equal length.
func Len(t testingT, object any, length int, msgAndArgs ...any) bool {
	return stop(t, assert.Len(t, object, length, msgAndArgs...))
}

// NotEmpty requires object to be non-empty.
func NotEmpty(t testingT, object any, msgAndArgs ...any) bool {
	return stop(t, assert.NotEmpty(t, object, msgAndArgs...))
}

// Empty requires object to be empty.
func Empty(t testingT, object any, msgAndArgs ...any) bool {
	return stop(t, assert.Empty(t, object, msgAndArgs...))
}

// True requires value to be true.
func True(t testingT, value bool, msgAndArgs ...any) bool {
	return stop(t, assert.True(t, value, msgAndArgs...))
}

// False requires value to be false.
func False(t testingT, value bool, msgAndArgs ...any) bool {
	return stop(t, assert.False(t, value, msgAndArgs...))
}

// Contains requires container to contain item.
func Contains(t testingT, container, item any, msgAndArgs ...any) bool {
	return stop(t, assert.Contains(t, container, item, msgAndArgs...))
}

// GreaterOrEqual requires actual to be greater than or equal to expected.
func GreaterOrEqual(t testingT, actual, expected any, msgAndArgs ...any) bool {
	return stop(t, assert.GreaterOrEqual(t, actual, expected, msgAndArgs...))
}

// Regexp requires str to match rx.
func Regexp(t testingT, rx any, str any, msgAndArgs ...any) bool {
	return stop(t, assert.Regexp(t, rx, str, msgAndArgs...))
}

// PanicsWithValue requires fn to panic with expected.
func PanicsWithValue(t testingT, expected any, fn func(), msgAndArgs ...any) bool {
	var recovered any
	didPanic := false
	func() {
		defer func() {
			recovered = recover()
			didPanic = recovered != nil
		}()
		fn()
	}()
	if !didPanic {
		assert.Equal(t, expected, recovered, msgAndArgs...)
		t.FailNow()
		return false
	}
	return stop(t, assert.Equal(t, expected, recovered, msgAndArgs...))
}

// FailNow marks the test as failed and stops execution.
func FailNow(t testingT, msgAndArgs ...any) {
	if len(msgAndArgs) > 0 {
		t.Errorf("%v", msgAndArgs...)
	}
	t.FailNow()
}
