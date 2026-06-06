package translate

import (
	"strconv"
	"strings"

	"github.com/avdoseferovic/paper/pkg/html/css"
)

type counterState struct {
	values map[string][]int
}

type counterMutation struct {
	name  string
	value int
}

func newCounterState() *counterState {
	return &counterState{values: map[string][]int{}}
}

func (c *counterState) enter(style *css.ComputedStyle) []string {
	if c == nil || style == nil {
		return nil
	}
	pushed := c.reset(style.CounterReset)
	c.increment(style.CounterIncrement)
	return pushed
}

func (c *counterState) exit(pushed []string) {
	if c == nil {
		return
	}
	for i := len(pushed) - 1; i >= 0; i-- {
		name := pushed[i]
		stack := c.values[name]
		if len(stack) <= 1 {
			delete(c.values, name)
			continue
		}
		c.values[name] = stack[:len(stack)-1]
	}
}

func (c *counterState) reset(value string) []string {
	mutations := parseCounterMutations(value, 0)
	if len(mutations) == 0 {
		return nil
	}
	pushed := make([]string, 0, len(mutations))
	for _, mutation := range mutations {
		c.values[mutation.name] = append(c.values[mutation.name], mutation.value)
		pushed = append(pushed, mutation.name)
	}
	return pushed
}

func (c *counterState) increment(value string) {
	mutations := parseCounterMutations(value, 1)
	for _, mutation := range mutations {
		stack := c.values[mutation.name]
		if len(stack) == 0 {
			stack = []int{0}
		}
		stack[len(stack)-1] += mutation.value
		c.values[mutation.name] = stack
	}
}

func (c *counterState) value(name string) int {
	if c == nil {
		return 0
	}
	stack := c.values[name]
	if len(stack) == 0 {
		return 0
	}
	return stack[len(stack)-1]
}

func (c *counterState) allValues(name string) []int {
	if c == nil {
		return []int{0}
	}
	stack := c.values[name]
	if len(stack) == 0 {
		return []int{0}
	}
	out := make([]int, len(stack))
	copy(out, stack)
	return out
}

func parseCounterMutations(value string, defaultValue int) []counterMutation {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "none") {
		return nil
	}
	fields := strings.Fields(value)
	mutations := make([]counterMutation, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		name := strings.TrimSpace(fields[i])
		if name == "" || isCounterKeyword(name) {
			continue
		}
		if _, ok := parseCounterInteger(name); ok {
			continue
		}
		mutation := counterMutation{name: name, value: defaultValue}
		if i+1 < len(fields) {
			if n, ok := parseCounterInteger(fields[i+1]); ok {
				mutation.value = n
				i++
			}
		}
		mutations = append(mutations, mutation)
	}
	return mutations
}

func parseCounterInteger(value string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	return n, err == nil
}

func isCounterKeyword(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none", "inherit", "initial", "unset", "revert":
		return true
	default:
		return false
	}
}

func formatCounterValue(value int, style string) string {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(style), `'"`)) {
	case "", "decimal":
		return strconv.Itoa(value)
	case "decimal-leading-zero":
		if value >= 0 && value < 10 {
			return "0" + strconv.Itoa(value)
		}
		if value < 0 && value > -10 {
			return "-0" + strconv.Itoa(-value)
		}
		return strconv.Itoa(value)
	case "lower-alpha", "lower-latin":
		return formatAlphaCounter(value, false)
	case "upper-alpha", "upper-latin":
		return formatAlphaCounter(value, true)
	case "lower-roman":
		return formatRomanCounter(value, false)
	case "upper-roman":
		return formatRomanCounter(value, true)
	default:
		return strconv.Itoa(value)
	}
}

func formatAlphaCounter(value int, upper bool) string {
	if value <= 0 {
		return strconv.Itoa(value)
	}
	var out []byte
	for value > 0 {
		value--
		ch := byte('a' + value%26)
		if upper {
			ch = byte('A' + value%26)
		}
		out = append([]byte{ch}, out...)
		value /= 26
	}
	return string(out)
}

func formatRomanCounter(value int, upper bool) string {
	if value <= 0 || value > 3999 {
		return strconv.Itoa(value)
	}
	pairs := []struct {
		value int
		upper string
		lower string
	}{
		{1000, "M", "m"}, {900, "CM", "cm"}, {500, "D", "d"}, {400, "CD", "cd"},
		{100, "C", "c"}, {90, "XC", "xc"}, {50, "L", "l"}, {40, "XL", "xl"},
		{10, "X", "x"}, {9, "IX", "ix"}, {5, "V", "v"}, {4, "IV", "iv"},
		{1, "I", "i"},
	}
	var out strings.Builder
	for _, pair := range pairs {
		for value >= pair.value {
			if upper {
				out.WriteString(pair.upper)
			} else {
				out.WriteString(pair.lower)
			}
			value -= pair.value
		}
	}
	return out.String()
}
