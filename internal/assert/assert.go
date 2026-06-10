// Package assert provides lightweight test assertions used by the root module.
package assert

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
)

type testingT interface {
	Errorf(format string, args ...any)
}

type helper interface {
	Helper()
}

func markHelper(t testingT) {
	if h, ok := t.(helper); ok {
		h.Helper()
	}
}

func fail(t testingT, msg string, msgAndArgs ...any) bool {
	markHelper(t)
	if len(msgAndArgs) > 0 {
		msg = msg + ": " + fmt.Sprint(msgAndArgs...)
	}
	t.Errorf("%s", msg)
	return false
}

// Equal reports whether expected and actual are deeply equal.
func Equal(t testingT, expected, actual any, msgAndArgs ...any) bool {
	markHelper(t)
	if !reflect.DeepEqual(expected, actual) {
		return fail(t, fmt.Sprintf("not equal\nexpected: %#v\nactual:   %#v", expected, actual), msgAndArgs...)
	}
	return true
}

// NotEqual reports whether expected and actual are not deeply equal.
func NotEqual(t testingT, expected, actual any, msgAndArgs ...any) bool {
	markHelper(t)
	if reflect.DeepEqual(expected, actual) {
		return fail(t, fmt.Sprintf("should not equal %#v", actual), msgAndArgs...)
	}
	return true
}

// Nil reports whether value is nil.
func Nil(t testingT, value any, msgAndArgs ...any) bool {
	markHelper(t)
	if !isNil(value) {
		return fail(t, fmt.Sprintf("expected nil, got %#v", value), msgAndArgs...)
	}
	return true
}

// NotNil reports whether value is not nil.
func NotNil(t testingT, value any, msgAndArgs ...any) bool {
	markHelper(t)
	if isNil(value) {
		return fail(t, "expected non-nil value", msgAndArgs...)
	}
	return true
}

// True reports whether value is true.
func True(t testingT, value bool, msgAndArgs ...any) bool {
	markHelper(t)
	if !value {
		return fail(t, "expected true", msgAndArgs...)
	}
	return true
}

// False reports whether value is false.
func False(t testingT, value bool, msgAndArgs ...any) bool {
	markHelper(t)
	if value {
		return fail(t, "expected false", msgAndArgs...)
	}
	return true
}

// NoError reports whether err is nil.
func NoError(t testingT, err error, msgAndArgs ...any) bool {
	markHelper(t)
	if err != nil {
		return fail(t, fmt.Sprintf("unexpected error: %v", err), msgAndArgs...)
	}
	return true
}

// Error reports whether err is non-nil.
func Error(t testingT, err error, msgAndArgs ...any) bool {
	markHelper(t)
	if err == nil {
		return fail(t, "expected error", msgAndArgs...)
	}
	return true
}

// ErrorIs reports whether err matches target.
func ErrorIs(t testingT, err, target error, msgAndArgs ...any) bool {
	markHelper(t)
	if !errors.Is(err, target) {
		return fail(t, fmt.Sprintf("expected error %v to match %v", err, target), msgAndArgs...)
	}
	return true
}

// Len reports whether object length equals expected.
func Len(t testingT, object any, length int, msgAndArgs ...any) bool {
	markHelper(t)
	got, ok := objectLen(object)
	if !ok {
		return fail(t, fmt.Sprintf("%T has no length", object), msgAndArgs...)
	}
	if got != length {
		return fail(t, fmt.Sprintf("expected length %d, got %d", length, got), msgAndArgs...)
	}
	return true
}

// Empty reports whether object is empty or zero.
func Empty(t testingT, object any, msgAndArgs ...any) bool {
	markHelper(t)
	if !isEmpty(object) {
		return fail(t, fmt.Sprintf("expected empty, got %#v", object), msgAndArgs...)
	}
	return true
}

// NotEmpty reports whether object is not empty.
func NotEmpty(t testingT, object any, msgAndArgs ...any) bool {
	markHelper(t)
	if isEmpty(object) {
		return fail(t, "expected non-empty value", msgAndArgs...)
	}
	return true
}

// Contains reports whether container contains item.
func Contains(t testingT, container, item any, msgAndArgs ...any) bool {
	markHelper(t)
	if !contains(container, item) {
		return fail(t, fmt.Sprintf("%#v does not contain %#v", container, item), msgAndArgs...)
	}
	return true
}

// NotContains reports whether container does not contain item.
func NotContains(t testingT, container, item any, msgAndArgs ...any) bool {
	markHelper(t)
	if contains(container, item) {
		return fail(t, fmt.Sprintf("%#v should not contain %#v", container, item), msgAndArgs...)
	}
	return true
}

// InDelta reports whether expected and actual differ by at most delta.
func InDelta(t testingT, expected, actual, delta any, msgAndArgs ...any) bool {
	markHelper(t)
	exp, ok := asFloat(expected)
	if !ok {
		return fail(t, fmt.Sprintf("%T is not numeric", expected), msgAndArgs...)
	}
	act, ok := asFloat(actual)
	if !ok {
		return fail(t, fmt.Sprintf("%T is not numeric", actual), msgAndArgs...)
	}
	d, ok := asFloat(delta)
	if !ok {
		return fail(t, fmt.Sprintf("%T is not numeric", delta), msgAndArgs...)
	}
	if math.Abs(exp-act) > d {
		return fail(t, fmt.Sprintf("expected %v and %v to differ by <= %v", expected, actual, delta), msgAndArgs...)
	}
	return true
}

// InDeltaSlice reports whether numeric slices differ element-wise by at most delta.
func InDeltaSlice(t testingT, expected, actual any, delta float64, msgAndArgs ...any) bool {
	markHelper(t)
	exp := reflect.ValueOf(expected)
	act := reflect.ValueOf(actual)
	if exp.Kind() != reflect.Slice && exp.Kind() != reflect.Array {
		return fail(t, fmt.Sprintf("%T is not a slice", expected), msgAndArgs...)
	}
	if act.Kind() != reflect.Slice && act.Kind() != reflect.Array {
		return fail(t, fmt.Sprintf("%T is not a slice", actual), msgAndArgs...)
	}
	if exp.Len() != act.Len() {
		return fail(t, fmt.Sprintf("expected length %d, got %d", exp.Len(), act.Len()), msgAndArgs...)
	}
	for i := range exp.Len() {
		if !InDelta(t, exp.Index(i).Interface(), act.Index(i).Interface(), delta, msgAndArgs...) {
			return false
		}
	}
	return true
}

// Greater reports whether actual is greater than expected.
func Greater(t testingT, actual, expected any, msgAndArgs ...any) bool {
	return compare(t, actual, expected, func(c int) bool { return c > 0 }, "greater than", msgAndArgs...)
}

// GreaterOrEqual reports whether actual is greater than or equal to expected.
func GreaterOrEqual(t testingT, actual, expected any, msgAndArgs ...any) bool {
	return compare(t, actual, expected, func(c int) bool { return c >= 0 }, "greater than or equal to", msgAndArgs...)
}

// Less reports whether actual is less than expected.
func Less(t testingT, actual, expected any, msgAndArgs ...any) bool {
	return compare(t, actual, expected, func(c int) bool { return c < 0 }, "less than", msgAndArgs...)
}

// LessOrEqual reports whether actual is less than or equal to expected.
func LessOrEqual(t testingT, actual, expected any, msgAndArgs ...any) bool {
	return compare(t, actual, expected, func(c int) bool { return c <= 0 }, "less than or equal to", msgAndArgs...)
}

// Zero reports whether value is the zero value for its type.
func Zero(t testingT, value any, msgAndArgs ...any) bool {
	markHelper(t)
	if value == nil {
		return true
	}
	if !reflect.ValueOf(value).IsZero() {
		return fail(t, fmt.Sprintf("expected zero, got %#v", value), msgAndArgs...)
	}
	return true
}

// Same reports whether expected and actual are the same object.
func Same(t testingT, expected, actual any, msgAndArgs ...any) bool {
	markHelper(t)
	if pointerOf(expected) != pointerOf(actual) {
		return fail(t, "expected same object", msgAndArgs...)
	}
	return true
}

// NotSame reports whether expected and actual are different objects.
func NotSame(t testingT, expected, actual any, msgAndArgs ...any) bool {
	markHelper(t)
	if pointerOf(expected) == pointerOf(actual) {
		return fail(t, "expected different objects", msgAndArgs...)
	}
	return true
}

// IsType reports whether expectedType and object have the same dynamic type.
func IsType(t testingT, expectedType, object any, msgAndArgs ...any) bool {
	markHelper(t)
	if reflect.TypeOf(expectedType) != reflect.TypeOf(object) {
		return fail(t, fmt.Sprintf("expected type %T, got %T", expectedType, object), msgAndArgs...)
	}
	return true
}

// Implements reports whether object implements interfaceObject.
func Implements(t testingT, interfaceObject, object any, msgAndArgs ...any) bool {
	markHelper(t)
	interfaceType := reflect.TypeOf(interfaceObject)
	if interfaceType == nil || interfaceType.Kind() != reflect.Ptr || interfaceType.Elem().Kind() != reflect.Interface {
		return fail(t, "interfaceObject must be a pointer to an interface", msgAndArgs...)
	}
	objectType := reflect.TypeOf(object)
	if objectType == nil || !objectType.Implements(interfaceType.Elem()) {
		return fail(t, fmt.Sprintf("%T does not implement %s", object, interfaceType.Elem()), msgAndArgs...)
	}
	return true
}

// NotPanics reports whether fn does not panic.
func NotPanics(t testingT, fn func(), msgAndArgs ...any) bool {
	markHelper(t)
	defer func() {
		if r := recover(); r != nil {
			fail(t, fmt.Sprintf("unexpected panic: %v", r), msgAndArgs...)
		}
	}()
	fn()
	return true
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	if isNilableKind(v.Kind()) {
		return v.IsNil()
	}
	return false
}

func objectLen(object any) (int, bool) {
	if object == nil {
		return 0, false
	}
	v := reflect.ValueOf(object)
	if isLengthKind(v.Kind()) {
		return v.Len(), true
	}
	return 0, false
}

func isEmpty(object any) bool {
	if object == nil {
		return true
	}
	v := reflect.ValueOf(object)
	if isLengthKind(v.Kind()) {
		return v.Len() == 0
	}
	return v.IsZero()
}

func contains(container, item any) bool {
	if container == nil {
		return false
	}
	if c, ok := container.(string); ok {
		return strings.Contains(c, fmt.Sprint(item))
	}
	v := reflect.ValueOf(container)
	if v.Kind() == reflect.Map {
		return v.MapIndex(reflect.ValueOf(item)).IsValid()
	}
	if v.Kind() == reflect.Array || v.Kind() == reflect.Slice {
		for i := range v.Len() {
			if reflect.DeepEqual(v.Index(i).Interface(), item) {
				return true
			}
		}
	}
	return false
}

func asFloat(value any) (float64, bool) {
	v := reflect.ValueOf(value)
	if isSignedIntegerKind(v.Kind()) {
		return float64(v.Int()), true
	}
	if isUnsignedIntegerKind(v.Kind()) {
		return float64(v.Uint()), true
	}
	if isFloatKind(v.Kind()) {
		return v.Float(), true
	}
	return 0, false
}

func compare(t testingT, actual, expected any, pass func(int) bool, relation string, msgAndArgs ...any) bool {
	markHelper(t)
	cmp, ok := comparison(actual, expected)
	if !ok {
		return fail(t, fmt.Sprintf("cannot compare %T and %T", actual, expected), msgAndArgs...)
	}
	if !pass(cmp) {
		return fail(t, fmt.Sprintf("expected %#v to be %s %#v", actual, relation, expected), msgAndArgs...)
	}
	return true
}

func comparison(actual, expected any) (int, bool) {
	if a, ok := asFloat(actual); ok {
		e, ok := asFloat(expected)
		if !ok {
			return 0, false
		}
		switch {
		case a < e:
			return -1, true
		case a > e:
			return 1, true
		default:
			return 0, true
		}
	}
	a, ok := actual.(string)
	if !ok {
		return 0, false
	}
	e, ok := expected.(string)
	if !ok {
		return 0, false
	}
	return strings.Compare(a, e), true
}

func pointerOf(value any) uintptr {
	if value == nil {
		return 0
	}
	v := reflect.ValueOf(value)
	if isPointerLikeKind(v.Kind()) {
		return v.Pointer()
	}
	return 0
}

func isNilableKind(kind reflect.Kind) bool {
	return kind == reflect.Chan ||
		kind == reflect.Func ||
		kind == reflect.Interface ||
		kind == reflect.Map ||
		kind == reflect.Ptr ||
		kind == reflect.Slice
}

func isLengthKind(kind reflect.Kind) bool {
	return kind == reflect.Array ||
		kind == reflect.Chan ||
		kind == reflect.Map ||
		kind == reflect.Slice ||
		kind == reflect.String
}

func isSignedIntegerKind(kind reflect.Kind) bool {
	return kind == reflect.Int ||
		kind == reflect.Int8 ||
		kind == reflect.Int16 ||
		kind == reflect.Int32 ||
		kind == reflect.Int64
}

func isUnsignedIntegerKind(kind reflect.Kind) bool {
	return kind == reflect.Uint ||
		kind == reflect.Uint8 ||
		kind == reflect.Uint16 ||
		kind == reflect.Uint32 ||
		kind == reflect.Uint64 ||
		kind == reflect.Uintptr
}

func isFloatKind(kind reflect.Kind) bool {
	return kind == reflect.Float32 || kind == reflect.Float64
}

func isPointerLikeKind(kind reflect.Kind) bool {
	return kind == reflect.Chan ||
		kind == reflect.Func ||
		kind == reflect.Map ||
		kind == reflect.Ptr ||
		kind == reflect.Slice
}

// Regexp reports whether str matches rx.
func Regexp(t testingT, rx any, str any, msgAndArgs ...any) bool {
	markHelper(t)
	pattern, ok := rx.(*regexp.Regexp)
	if !ok {
		pattern = regexp.MustCompile(fmt.Sprint(rx))
	}
	if !pattern.MatchString(fmt.Sprint(str)) {
		return fail(t, fmt.Sprintf("%q does not match %s", str, pattern.String()), msgAndArgs...)
	}
	return true
}
