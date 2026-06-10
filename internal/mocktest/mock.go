// Package mock provides the small mock API subset used by generated internal
// test mocks.
package mock

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

// Anything matches any argument.
const Anything = "mock.Anything"

// TestingT is the testing surface used by generated mock constructors.
type TestingT interface {
	Logf(format string, args ...any)
	Errorf(format string, args ...any)
	FailNow()
}

// Arguments contains method call arguments or return values.
type Arguments []any

// Get returns the argument at index.
func (args Arguments) Get(index int) any {
	if index >= len(args) {
		panic(fmt.Sprintf("mocktest: cannot call Get(%d); there are %d arguments", index, len(args)))
	}
	return args[index]
}

// Error returns the argument at index as an error.
func (args Arguments) Error(index int) error {
	if args.Get(index) == nil {
		return nil
	}
	err, ok := args.Get(index).(error)
	if !ok {
		panic(fmt.Sprintf("mocktest: argument %d is %T, not error", index, args.Get(index)))
	}
	return err
}

// Mock records expected method calls.
type Mock struct {
	test TestingT

	mu       sync.Mutex
	expected []*Call
	calls    []recordedCall
}

type recordedCall struct {
	method string
	args   Arguments
}

// Test stores the test handle used for expectation failures.
func (m *Mock) Test(t TestingT) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.test = t
}

// On registers an expected method call.
func (m *Mock) On(method string, args ...any) *Call {
	m.mu.Lock()
	defer m.mu.Unlock()

	call := &Call{
		parent: m,
		Method: method,
		args:   append(Arguments(nil), args...),
		min:    1,
		max:    -1,
	}
	m.expected = append(m.expected, call)
	return call
}

// Called records the caller method and returns the configured return values.
func (m *Mock) Called(args ...any) Arguments {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("mocktest: could not resolve caller")
	}
	name := runtime.FuncForPC(pc).Name()
	parts := strings.Split(name, ".")
	return m.MethodCalled(parts[len(parts)-1], args...)
}

// MethodCalled records method and returns the configured return values.
func (m *Mock) MethodCalled(method string, args ...any) Arguments {
	m.mu.Lock()
	defer m.mu.Unlock()

	call := m.find(method, args)
	if call == nil {
		m.failf("unexpected call %s(%s)", method, formatArgs(args))
		return nil
	}

	call.count++
	m.calls = append(m.calls, recordedCall{method: method, args: append(Arguments(nil), args...)})
	if call.run != nil {
		call.run(append(Arguments(nil), args...))
	}
	return append(Arguments(nil), call.returns...)
}

// AssertExpectations verifies that all required calls happened.
func (m *Mock) AssertExpectations(t TestingT) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	ok := true
	for _, call := range m.expected {
		if call.max < 0 && call.min > 0 && m.wasCalled(call.Method, call.args) {
			continue
		}
		if call.count < call.min {
			t.Errorf("mocktest: expected %s to be called at least %d time(s), called %d", call.Method, call.min, call.count)
			ok = false
		}
		if call.max >= 0 && call.count != call.max {
			t.Errorf("mocktest: expected %s to be called %d time(s), called %d", call.Method, call.max, call.count)
			ok = false
		}
	}
	return ok
}

// AssertNotCalled verifies that method was not called with matching arguments.
func (m *Mock) AssertNotCalled(t TestingT, method string, args ...any) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, call := range m.calls {
		if call.method == method && matchArgs(args, call.args) {
			t.Errorf("mocktest: expected %s not to be called with %s", method, formatArgs(args))
			return false
		}
	}
	return true
}

func (m *Mock) find(method string, args Arguments) *Call {
	for _, call := range m.expected {
		if call.Method != method || !call.canCall() {
			continue
		}
		if call.matches(args) {
			return call
		}
	}
	return nil
}

func (m *Mock) wasCalled(method string, args Arguments) bool {
	for _, call := range m.calls {
		if call.method == method && matchArgs(args, call.args) {
			return true
		}
	}
	return false
}

func (m *Mock) failf(format string, args ...any) {
	if m.test == nil {
		panic(fmt.Sprintf(format, args...))
	}
	m.test.Errorf(format, args...)
	m.test.FailNow()
}

// Call configures a single expected method call.
type Call struct {
	parent *Mock

	// Method is the expected method name.
	Method string

	args    Arguments
	returns Arguments
	run     func(Arguments)
	min     int
	max     int
	count   int
}

// Return configures return values for the expected call.
func (c *Call) Return(values ...any) *Call {
	c.parent.mu.Lock()
	defer c.parent.mu.Unlock()
	c.returns = append(Arguments(nil), values...)
	return c
}

// Run configures a callback that receives actual call arguments.
func (c *Call) Run(fn func(args Arguments)) *Call {
	c.parent.mu.Lock()
	defer c.parent.mu.Unlock()
	c.run = fn
	return c
}

// Once requires exactly one call.
func (c *Call) Once() *Call {
	return c.Times(1)
}

// Twice requires exactly two calls.
func (c *Call) Twice() *Call {
	return c.Times(2)
}

// Times requires exactly n calls.
func (c *Call) Times(n int) *Call {
	c.parent.mu.Lock()
	defer c.parent.mu.Unlock()
	c.min = n
	c.max = n
	return c
}

// Maybe marks the call as optional.
func (c *Call) Maybe() *Call {
	c.parent.mu.Lock()
	defer c.parent.mu.Unlock()
	c.min = 0
	c.max = -1
	return c
}

func (c *Call) canCall() bool {
	return c.max < 0 || c.count < c.max
}

func (c *Call) matches(actual Arguments) bool {
	return matchArgs(c.args, actual)
}

func matchArgs(expected, actual Arguments) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if !matchArg(expected[i], actual[i]) {
			return false
		}
	}
	return true
}

type matcher interface {
	Matches(argument any) bool
}

func matchArg(expected, actual any) bool {
	if expected == Anything {
		return true
	}
	if m, ok := expected.(matcher); ok {
		return m.Matches(actual)
	}
	if matchesAnythingOfType(expected, actual) {
		return true
	}
	return reflect.DeepEqual(expected, actual)
}

func matchesAnythingOfType(expected, actual any) bool {
	v := reflect.ValueOf(expected)
	if v.Kind() != reflect.String {
		return false
	}
	typeName := v.String()
	if typeName == "" || typeName == Anything {
		return false
	}
	actualType := reflect.TypeOf(actual)
	return actualType != nil && actualType.String() == typeName
}

type anythingOfTypeArgument string

// AnythingOfType matches arguments with the provided reflected type string.
func AnythingOfType(t string) any {
	return anythingOfTypeArgument(t)
}

type argumentMatcher struct {
	fn reflect.Value
}

// MatchedBy matches arguments accepted by fn.
func MatchedBy(fn any) any {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		panic(fmt.Sprintf("mocktest: %T is not a matcher function", fn))
	}
	if fnType.NumIn() != 1 || fnType.NumOut() != 1 || fnType.Out(0).Kind() != reflect.Bool {
		panic("mocktest: matcher must be func(T) bool")
	}
	return argumentMatcher{fn: reflect.ValueOf(fn)}
}

// Matches reports whether argument satisfies the matcher function.
func (m argumentMatcher) Matches(argument any) bool {
	expectType := m.fn.Type().In(0)
	argType := reflect.TypeOf(argument)
	if argType == nil {
		if isNilableMatcherKind(expectType.Kind()) {
			return m.fn.Call([]reflect.Value{reflect.Zero(expectType)})[0].Bool()
		}
		return false
	}
	if !argType.AssignableTo(expectType) {
		return false
	}
	return m.fn.Call([]reflect.Value{reflect.ValueOf(argument)})[0].Bool()
}

func isNilableMatcherKind(kind reflect.Kind) bool {
	return kind == reflect.Interface ||
		kind == reflect.Chan ||
		kind == reflect.Func ||
		kind == reflect.Map ||
		kind == reflect.Slice ||
		kind == reflect.Ptr
}

func formatArgs(args Arguments) string {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = fmt.Sprintf("%#v", arg)
	}
	return strings.Join(parts, ", ")
}
