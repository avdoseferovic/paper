package translate

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const (
	pseudoNot   = "not"
	pseudoWhere = "where"
)

var (
	errEmptySelector                    = errors.New("empty selector")
	errInvalidSelectorCombinator        = errors.New("invalid selector combinator")
	errEmptySelectorStep                = errors.New("empty selector step")
	errIncompleteSelector               = errors.New("incomplete selector")
	errInvalidIDSelector                = errors.New("invalid ID selector")
	errInvalidClassSelector             = errors.New("invalid class selector")
	errUnterminatedAttributeSelector    = errors.New("unterminated attribute selector")
	errPseudoElementNotSplit            = errors.New("pseudo-element must be split before matching")
	errInvalidPseudoSelector            = errors.New("invalid pseudo selector")
	errUnterminatedPseudoSelector       = errors.New("unterminated pseudo selector")
	errEmptyPseudoSelectorArgument      = errors.New("empty pseudo selector argument")
	errUnsupportedSelectorCharacter     = errors.New("unsupported selector character")
	errEmptyCompoundSelector            = errors.New("empty compound selector")
	errInvalidAttributeSelector         = errors.New("invalid attribute selector")
	errInvalidAttributeSelectorOperator = errors.New("invalid attribute operator")
	errMissingAttributeSelectorValue    = errors.New("missing attribute value")
	errUnterminatedAttributeValue       = errors.New("unterminated attribute value")
	errInvalidAttributeSelectorSuffix   = errors.New("invalid attribute selector suffix")
)

type selectorSpecificity struct {
	id, class, tag int
}

func (specificity selectorSpecificity) Less(other selectorSpecificity) bool {
	if specificity.id != other.id {
		return specificity.id < other.id
	}
	if specificity.class != other.class {
		return specificity.class < other.class
	}
	return specificity.tag < other.tag
}

type selectorMatcher struct {
	steps       []selectorStep
	specificity selectorSpecificity
}

type selectorStep struct {
	compound compoundSelector
	relation selectorRelation // relationship from the previous step to this one
}

type selectorRelation uint8

const (
	noRelation selectorRelation = iota
	descendantRelation
	childRelation
	adjacentSiblingRelation
	generalSiblingRelation
)

type compoundSelector struct {
	tag     string
	id      string
	classes []string
	attrs   []attributeSelector
	pseudos []pseudoSelector
}

type attributeSelector struct {
	name  string
	op    string
	value string
	flag  byte
}

type pseudoSelector struct {
	name     string
	argument string
	matchers []selectorMatcher
}

func compileSelector(value string) (selectorMatcher, error) {
	steps, err := parseSelectorSteps(strings.TrimSpace(value))
	if err != nil {
		return selectorMatcher{}, err
	}
	matcher := selectorMatcher{steps: steps}
	for _, step := range steps {
		matcher.specificity = addSpecificity(matcher.specificity, step.compound.specificity())
	}
	return matcher, nil
}

func (matcher selectorMatcher) Match(node *html.Node) bool {
	if node == nil || len(matcher.steps) == 0 {
		return false
	}
	return matcher.matchStep(node, len(matcher.steps)-1)
}

func (matcher selectorMatcher) Specificity() selectorSpecificity { return matcher.specificity }

func (matcher selectorMatcher) matchStep(node *html.Node, index int) bool {
	if !matcher.steps[index].compound.match(node) {
		return false
	}
	if index == 0 {
		return true
	}
	switch matcher.steps[index].relation {
	case noRelation:
		return false
	case descendantRelation:
		for parent := node.Parent; parent != nil; parent = parent.Parent {
			if matcher.matchStep(parent, index-1) {
				return true
			}
		}
		return false
	case childRelation:
		return node.Parent != nil && matcher.matchStep(node.Parent, index-1)
	case adjacentSiblingRelation:
		return matcher.matchStep(previousElementSibling(node), index-1)
	case generalSiblingRelation:
		for sibling := previousElementSibling(node); sibling != nil; sibling = previousElementSibling(sibling) {
			if matcher.matchStep(sibling, index-1) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (compound compoundSelector) specificity() selectorSpecificity {
	result := selectorSpecificity{}
	if compound.id != "" {
		result.id++
	}
	result.class += len(compound.classes) + len(compound.attrs)
	if compound.tag != "" && compound.tag != "*" {
		result.tag++
	}
	for _, pseudo := range compound.pseudos {
		switch pseudo.name {
		case pseudoWhere:
			// :where is expanded before compilation by the stylesheet parser.
		case pseudoNot, "is", "has":
			if len(pseudo.matchers) > 0 {
				result = addSpecificity(result, maxSelectorSpecificity(pseudo.matchers))
			} else {
				result.class++
			}
		default:
			result.class++
		}
	}
	return result
}

func addSpecificity(left, right selectorSpecificity) selectorSpecificity {
	return selectorSpecificity{left.id + right.id, left.class + right.class, left.tag + right.tag}
}

func (compound compoundSelector) match(node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode {
		return false
	}
	if compound.tag != "" && compound.tag != "*" && !strings.EqualFold(compound.tag, node.Data) {
		return false
	}
	if compound.id != "" && nodeAttribute(node, "id") != compound.id {
		return false
	}
	for _, class := range compound.classes {
		if !hasClass(node, class) {
			return false
		}
	}
	for _, attribute := range compound.attrs {
		if !attribute.match(node) {
			return false
		}
	}
	for _, pseudo := range compound.pseudos {
		if !pseudo.match(node) {
			return false
		}
	}
	return true
}

func (attribute attributeSelector) match(node *html.Node) bool {
	value, ok := nodeAttributeOK(node, attribute.name)
	if !ok {
		return false
	}
	if attribute.op == "" {
		return true
	}
	needle := attribute.value
	if attribute.flag == 'i' {
		value = strings.ToLower(value)
		needle = strings.ToLower(needle)
	}
	switch attribute.op {
	case "=":
		return value == needle
	case "~=":
		return strings.Contains(" "+strings.Join(strings.Fields(value), " ")+" ", " "+needle+" ")
	case "|=":
		return value == needle || strings.HasPrefix(value, needle+"-")
	case "^=":
		return strings.HasPrefix(value, needle)
	case "$=":
		return strings.HasSuffix(value, needle)
	case "*=":
		return strings.Contains(value, needle)
	default:
		return false
	}
}

//nolint:gocyclo // This direct dispatch keeps each supported pseudo-class easy to audit.
func (pseudo pseudoSelector) match(node *html.Node) bool {
	switch pseudo.name {
	case "root":
		return node.Parent != nil && node.Parent.Type == html.DocumentNode
	case "first-child":
		return previousElementSibling(node) == nil
	case "last-child":
		return nextElementSibling(node) == nil
	case "only-child":
		return previousElementSibling(node) == nil && nextElementSibling(node) == nil
	case "first-of-type":
		return previousElementOfType(node) == nil
	case "last-of-type":
		return nextElementOfType(node) == nil
	case "only-of-type":
		return previousElementOfType(node) == nil && nextElementOfType(node) == nil
	case "empty":
		return node.FirstChild == nil
	case "nth-child":
		return matchNth(pseudo.argument, elementIndex(node, false, false))
	case "nth-last-child":
		return matchNth(pseudo.argument, elementIndex(node, false, true))
	case "nth-of-type":
		return matchNth(pseudo.argument, elementIndex(node, true, false))
	case "nth-last-of-type":
		return matchNth(pseudo.argument, elementIndex(node, true, true))
	case "not":
		for _, matcher := range pseudo.matchers {
			if matcher.Match(node) {
				return false
			}
		}
		return len(pseudo.matchers) > 0
	case "is", "where":
		for _, matcher := range pseudo.matchers {
			if matcher.Match(node) {
				return true
			}
		}
		return false
	case "checked":
		return hasAttribute(node, "checked") || hasAttribute(node, "selected")
	case "disabled":
		return hasAttribute(node, "disabled")
	case "enabled":
		return !hasAttribute(node, "disabled")
	case "required":
		return hasAttribute(node, "required")
	case "optional":
		return !hasAttribute(node, "required")
	case "link":
		return strings.EqualFold(node.Data, "a") && hasAttribute(node, "href")
	case "hover", "active", "focus", "focus-visible", "focus-within", "visited", "target", "has":
		// Paper renders a static document: interaction and navigation state do
		// not exist. :has is intentionally unsupported until a bounded relative
		// selector implementation is needed.
		return false
	default:
		return false
	}
}

//nolint:gocognit,gocyclo // CSS selector combinators require delimiter/quote state while scanning a selector.
func parseSelectorSteps(value string) ([]selectorStep, error) {
	if value == "" {
		return nil, errEmptySelector
	}
	var steps []selectorStep
	position := 0
	relation := noRelation
	for position < len(value) {
		for position < len(value) && selectorSpace(value[position]) {
			position++
		}
		if position >= len(value) {
			break
		}
		if value[position] == '>' || value[position] == '+' || value[position] == '~' {
			if len(steps) == 0 || relation != noRelation {
				return nil, errInvalidSelectorCombinator
			}
			relation = explicitRelation(value[position])
			position++
			continue
		}
		start := position
		depthBracket, depthParen := 0, 0
		quote := byte(0)
		for position < len(value) {
			current := value[position]
			if quote != 0 {
				if current == '\\' && position+1 < len(value) {
					position += 2
					continue
				}
				if current == quote {
					quote = 0
				}
				position++
				continue
			}
			switch current {
			case '\'', '"':
				quote = current
			case '[':
				depthBracket++
			case ']':
				if depthBracket > 0 {
					depthBracket--
				}
			case '(':
				depthParen++
			case ')':
				if depthParen > 0 {
					depthParen--
				}
			case '>', '+', '~':
				if depthBracket == 0 && depthParen == 0 {
					goto compoundDone
				}
			default:
				if selectorSpace(current) && depthBracket == 0 && depthParen == 0 {
					goto compoundDone
				}
			}
			position++
		}
	compoundDone:
		if start == position {
			return nil, errEmptySelectorStep
		}
		compound, err := parseCompoundSelector(value[start:position])
		if err != nil {
			return nil, err
		}
		steps = append(steps, selectorStep{compound: compound, relation: relation})
		if len(steps) > 1 && relation == noRelation {
			steps[len(steps)-1].relation = descendantRelation
		}
		relation = noRelation

		space := false
		for position < len(value) && selectorSpace(value[position]) {
			position++
			space = true
		}
		if position < len(value) && (value[position] == '>' || value[position] == '+' || value[position] == '~') {
			relation = explicitRelation(value[position])
			position++
			continue
		}
		if space && position < len(value) {
			relation = descendantRelation
		}
	}
	if len(steps) == 0 || relation != noRelation {
		return nil, errIncompleteSelector
	}
	return steps, nil
}

//nolint:gocognit,gocyclo,nestif // A compound selector is a short, explicit grammar production.
func parseCompoundSelector(value string) (compoundSelector, error) {
	compound := compoundSelector{}
	position := 0
	if position < len(value) && value[position] == '*' {
		compound.tag = "*"
		position++
	} else if position < len(value) && isSelectorNameStart(value[position]) {
		compound.tag, position = readSelectorName(value, position)
	}

	for position < len(value) {
		switch value[position] {
		case '#':
			name, next := readSelectorName(value, position+1)
			if name == "" || compound.id != "" {
				return compoundSelector{}, errInvalidIDSelector
			}
			compound.id, position = name, next
		case '.':
			name, next := readSelectorName(value, position+1)
			if name == "" {
				return compoundSelector{}, errInvalidClassSelector
			}
			compound.classes = append(compound.classes, name)
			position = next
		case '[':
			end, ok := selectorClosing(value, position, '[', ']')
			if !ok {
				return compoundSelector{}, errUnterminatedAttributeSelector
			}
			attribute, err := parseAttributeSelector(value[position+1 : end])
			if err != nil {
				return compoundSelector{}, err
			}
			compound.attrs = append(compound.attrs, attribute)
			position = end + 1
		case ':':
			if position+1 < len(value) && value[position+1] == ':' {
				return compoundSelector{}, errPseudoElementNotSplit
			}
			name, next := readSelectorName(value, position+1)
			if name == "" {
				return compoundSelector{}, errInvalidPseudoSelector
			}
			pseudo := pseudoSelector{name: strings.ToLower(name)}
			position = next
			if position < len(value) && value[position] == '(' {
				end, ok := selectorClosing(value, position, '(', ')')
				if !ok {
					return compoundSelector{}, errUnterminatedPseudoSelector
				}
				pseudo.argument = strings.TrimSpace(value[position+1 : end])
				position = end + 1
				if pseudo.name == pseudoNot || pseudo.name == "is" || pseudo.name == pseudoWhere {
					for _, argument := range splitSelectorFunctionArgs(pseudo.argument) {
						matcher, err := compileSelector(argument)
						if err != nil {
							return compoundSelector{}, err
						}
						pseudo.matchers = append(pseudo.matchers, matcher)
					}
					if len(pseudo.matchers) == 0 {
						return compoundSelector{}, errEmptyPseudoSelectorArgument
					}
				}
			}
			compound.pseudos = append(compound.pseudos, pseudo)
		default:
			return compoundSelector{}, fmt.Errorf("%w: %q", errUnsupportedSelectorCharacter, value[position])
		}
	}
	if compound.tag == "" && compound.id == "" && len(compound.classes) == 0 && len(compound.attrs) == 0 && len(compound.pseudos) == 0 {
		return compoundSelector{}, errEmptyCompoundSelector
	}
	return compound, nil
}

func parseAttributeSelector(value string) (attributeSelector, error) {
	value = strings.TrimSpace(value)
	name, position := readSelectorName(value, 0)
	if name == "" {
		return attributeSelector{}, errInvalidAttributeSelector
	}
	result := attributeSelector{name: strings.ToLower(name)}
	for position < len(value) && selectorSpace(value[position]) {
		position++
	}
	if position == len(value) {
		return result, nil
	}
	switch {
	case position+1 < len(value) && strings.ContainsRune("~|^$*", rune(value[position])) && value[position+1] == '=':
		result.op = value[position : position+2]
		position += 2
	case value[position] == '=':
		result.op = "="
		position++
	default:
		return attributeSelector{}, errInvalidAttributeSelectorOperator
	}
	for position < len(value) && selectorSpace(value[position]) {
		position++
	}
	if position >= len(value) {
		return attributeSelector{}, errMissingAttributeSelectorValue
	}
	if value[position] == '\'' || value[position] == '"' {
		quote := value[position]
		end := position + 1
		for end < len(value) && value[end] != quote {
			if value[end] == '\\' && end+1 < len(value) {
				end += 2
				continue
			}
			end++
		}
		if end == len(value) {
			return attributeSelector{}, errUnterminatedAttributeValue
		}
		result.value = unescapeSelector(value[position+1 : end])
		position = end + 1
	} else {
		start := position
		for position < len(value) && !selectorSpace(value[position]) {
			position++
		}
		result.value = unescapeSelector(value[start:position])
	}
	for position < len(value) && selectorSpace(value[position]) {
		position++
	}
	if position < len(value) {
		if position+1 != len(value) || (value[position] != 'i' && value[position] != 'I' && value[position] != 's' && value[position] != 'S') {
			return attributeSelector{}, errInvalidAttributeSelectorSuffix
		}
		result.flag = strings.ToLower(value[position : position+1])[0]
	}
	return result, nil
}

func matchNth(expression string, index int) bool {
	if index <= 0 {
		return false
	}
	expression = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(expression), " ", ""))
	switch expression {
	case "odd":
		return index%2 == 1
	case "even":
		return index%2 == 0
	}
	if !strings.Contains(expression, "n") {
		value, err := strconv.Atoi(expression)
		return err == nil && index == value
	}
	parts := strings.SplitN(expression, "n", 2)
	coefficient := 1
	switch parts[0] {
	case "", "+":
	case "-":
		coefficient = -1
	default:
		value, err := strconv.Atoi(parts[0])
		if err != nil {
			return false
		}
		coefficient = value
	}
	offset := 0
	if parts[1] != "" {
		value, err := strconv.Atoi(parts[1])
		if err != nil {
			return false
		}
		offset = value
	}
	if coefficient == 0 {
		return index == offset
	}
	delta := index - offset
	return delta%coefficient == 0 && delta/coefficient >= 0
}

func elementIndex(node *html.Node, sameType, fromEnd bool) int {
	index := 1
	if fromEnd {
		for sibling := nextElementSibling(node); sibling != nil; sibling = nextElementSibling(sibling) {
			if !sameType || strings.EqualFold(sibling.Data, node.Data) {
				index++
			}
		}
		return index
	}
	for sibling := previousElementSibling(node); sibling != nil; sibling = previousElementSibling(sibling) {
		if !sameType || strings.EqualFold(sibling.Data, node.Data) {
			index++
		}
	}
	return index
}

func previousElementSibling(node *html.Node) *html.Node {
	if node == nil {
		return nil
	}
	for sibling := node.PrevSibling; sibling != nil; sibling = sibling.PrevSibling {
		if sibling.Type == html.ElementNode {
			return sibling
		}
	}
	return nil
}

func nextElementSibling(node *html.Node) *html.Node {
	if node == nil {
		return nil
	}
	for sibling := node.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		if sibling.Type == html.ElementNode {
			return sibling
		}
	}
	return nil
}

func previousElementOfType(node *html.Node) *html.Node {
	for sibling := previousElementSibling(node); sibling != nil; sibling = previousElementSibling(sibling) {
		if strings.EqualFold(sibling.Data, node.Data) {
			return sibling
		}
	}
	return nil
}

func nextElementOfType(node *html.Node) *html.Node {
	for sibling := nextElementSibling(node); sibling != nil; sibling = nextElementSibling(sibling) {
		if strings.EqualFold(sibling.Data, node.Data) {
			return sibling
		}
	}
	return nil
}

func nodeAttribute(node *html.Node, name string) string {
	value, _ := nodeAttributeOK(node, name)
	return value
}

func nodeAttributeOK(node *html.Node, name string) (string, bool) {
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, name) {
			return attribute.Val, true
		}
	}
	return "", false
}

func hasAttribute(node *html.Node, name string) bool {
	_, ok := nodeAttributeOK(node, name)
	return ok
}

func hasClass(node *html.Node, class string) bool {
	return slices.Contains(strings.Fields(nodeAttribute(node, "class")), class)
}

func selectorClosing(value string, start int, open, closing byte) (int, bool) {
	depth := 0
	quote := byte(0)
	for position := start; position < len(value); position++ {
		current := value[position]
		if quote != 0 {
			if current == '\\' {
				position++
				continue
			}
			if current == quote {
				quote = 0
			}
			continue
		}
		if current == '\'' || current == '"' {
			quote = current
			continue
		}
		if current == open {
			depth++
		}
		if current == closing {
			depth--
			if depth == 0 {
				return position, true
			}
		}
	}
	return 0, false
}

func readSelectorName(value string, position int) (string, int) {
	start := position
	for position < len(value) {
		if value[position] == '\\' && position+1 < len(value) {
			position += 2
			continue
		}
		if !isSelectorName(value[position]) {
			break
		}
		position++
	}
	return unescapeSelector(value[start:position]), position
}

func unescapeSelector(value string) string {
	return strings.ReplaceAll(value, "\\", "")
}

func isSelectorNameStart(value byte) bool {
	return isSelectorName(value) && value != '-' && value != '_'
}

func isSelectorName(value byte) bool {
	return (value >= 'a' && value <= 'z') ||
		(value >= 'A' && value <= 'Z') ||
		(value >= '0' && value <= '9') ||
		value == '-' || value == '_' || value >= 0x80
}

func selectorSpace(value byte) bool {
	switch value {
	case ' ', '\t', '\r', '\n', '\f':
		return true
	default:
		return false
	}
}

func explicitRelation(value byte) selectorRelation {
	switch value {
	case '>':
		return childRelation
	case '+':
		return adjacentSiblingRelation
	case '~':
		return generalSiblingRelation
	default:
		return noRelation
	}
}

func maxSelectorSpecificity(matchers []selectorMatcher) selectorSpecificity {
	maximum := selectorSpecificity{}
	for _, matcher := range matchers {
		if maximum.Less(matcher.specificity) {
			maximum = matcher.specificity
		}
	}
	return maximum
}
