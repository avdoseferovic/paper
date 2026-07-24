package svg

import (
	"strconv"
	"strings"
)

//nolint:gocognit,gocyclo // SVG paths are command-driven; keeping their state transitions together prevents divergence.
func (path *svgPath) parse(data string) bool {
	tokens := scanPathTokens(data)
	if len(tokens) == 0 {
		return false
	}
	index := 0
	command := byte(0)
	previousCommand := byte(0)
	lastCubic, lastQuadratic := svgPoint{}, svgPoint{}
	for index < len(tokens) {
		if tokens[index].command != 0 {
			command = tokens[index].command
			index++
		} else if command == 0 {
			return false
		}
		relative := command >= 'a' && command <= 'z'
		kind := command
		if kind >= 'a' && kind <= 'z' {
			kind -= 'a' - 'A'
		}
		read := func(count int) ([]float64, bool) {
			if index+count > len(tokens) {
				return nil, false
			}
			values := make([]float64, count)
			for offset := range values {
				if tokens[index+offset].command != 0 {
					return nil, false
				}
				values[offset] = tokens[index+offset].number
			}
			index += count
			return values, true
		}
		point := func(x, y float64) svgPoint {
			if relative && path.hasCurrent {
				return svgPoint{path.current.x + x, path.current.y + y}
			}
			return svgPoint{x, y}
		}
		switch kind {
		case 'M':
			values, ok := read(2)
			if !ok {
				return false
			}
			path.moveTo(point(values[0], values[1]).x, point(values[0], values[1]).y)
			if relative {
				command = 'l'
			} else {
				command = 'L'
			}
		case 'L':
			values, ok := read(2)
			if !ok {
				return false
			}
			end := point(values[0], values[1])
			path.lineTo(end.x, end.y)
		case 'H':
			values, ok := read(1)
			if !ok || !path.hasCurrent {
				return false
			}
			x := values[0]
			if relative {
				x += path.current.x
			}
			path.lineTo(x, path.current.y)
		case 'V':
			values, ok := read(1)
			if !ok || !path.hasCurrent {
				return false
			}
			y := values[0]
			if relative {
				y += path.current.y
			}
			path.lineTo(path.current.x, y)
		case 'C':
			values, ok := read(6)
			if !ok {
				return false
			}
			control1, control2, end := point(values[0], values[1]), point(values[2], values[3]), point(values[4], values[5])
			path.cubeTo(control1, control2, end)
			lastCubic = control2
		case 'S':
			values, ok := read(4)
			if !ok || !path.hasCurrent {
				return false
			}
			control1 := path.current
			if previousCommand == 'C' || previousCommand == 'c' || previousCommand == 'S' || previousCommand == 's' {
				control1 = svgPoint{2*path.current.x - lastCubic.x, 2*path.current.y - lastCubic.y}
			}
			control2, end := point(values[0], values[1]), point(values[2], values[3])
			path.cubeTo(control1, control2, end)
			lastCubic = control2
		case 'Q':
			values, ok := read(4)
			if !ok {
				return false
			}
			control, end := point(values[0], values[1]), point(values[2], values[3])
			path.quadTo(control, end)
			lastQuadratic = control
		case 'T':
			values, ok := read(2)
			if !ok || !path.hasCurrent {
				return false
			}
			control := path.current
			if previousCommand == 'Q' || previousCommand == 'q' || previousCommand == 'T' || previousCommand == 't' {
				control = svgPoint{2*path.current.x - lastQuadratic.x, 2*path.current.y - lastQuadratic.y}
			}
			end := point(values[0], values[1])
			path.quadTo(control, end)
			lastQuadratic = control
		case 'A':
			values, ok := read(7)
			if !ok {
				return false
			}
			// Arc endpoint semantics are preserved; this compact renderer draws
			// the chord until elliptical-arc flattening is required by Paper.
			end := point(values[5], values[6])
			path.lineTo(end.x, end.y)
		case 'Z':
			path.close()
		default:
			return false
		}
		previousCommand = command
	}
	return path.hasCurrent
}

type pathToken struct {
	command byte
	number  float64
}

//nolint:gocognit // The scanner deliberately accepts compact SVG number syntax in one bounded pass.
func scanPathTokens(value string) []pathToken {
	var tokens []pathToken
	for index := 0; index < len(value); {
		if strings.ContainsRune(" \t\r\n,", rune(value[index])) {
			index++
			continue
		}
		if value[index] >= 'A' && value[index] <= 'Z' || value[index] >= 'a' && value[index] <= 'z' {
			tokens = append(tokens, pathToken{command: value[index]})
			index++
			continue
		}
		start := index
		if value[index] == '+' || value[index] == '-' {
			index++
		}
		digits := false
		for index < len(value) && value[index] >= '0' && value[index] <= '9' {
			index++
			digits = true
		}
		if index < len(value) && value[index] == '.' {
			index++
			for index < len(value) && value[index] >= '0' && value[index] <= '9' {
				index++
				digits = true
			}
		}
		if index < len(value) && (value[index] == 'e' || value[index] == 'E') {
			exponent := index + 1
			if exponent < len(value) && (value[exponent] == '+' || value[exponent] == '-') {
				exponent++
			}
			end := exponent
			for end < len(value) && value[end] >= '0' && value[end] <= '9' {
				end++
			}
			if end > exponent {
				index = end
			}
		}
		if !digits {
			index++
			continue
		}
		number, err := strconv.ParseFloat(value[start:index], 64)
		if err == nil {
			tokens = append(tokens, pathToken{number: number})
		}
	}
	return tokens
}
