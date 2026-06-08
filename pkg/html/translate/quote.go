package translate

import (
	"strings"

	"github.com/avdoseferovic/paper/pkg/html/css"
)

type quoteState struct {
	depth int
}

type quotePair struct {
	open  string
	close string
}

func newQuoteState() *quoteState {
	return &quoteState{}
}

func (q *quoteState) open(style *css.ComputedStyle) string {
	pairs := quotePairs(style)
	idx := 0
	if q != nil {
		idx = quotePairIndex(q.depth, len(pairs))
		q.depth++
	}
	return pairs[idx].open
}

func (q *quoteState) close(style *css.ComputedStyle) string {
	pairs := quotePairs(style)
	idx := 0
	if q != nil {
		if q.depth > 0 {
			q.depth--
		}
		idx = quotePairIndex(q.depth, len(pairs))
	}
	return pairs[idx].close
}

func (q *quoteState) noOpen() {
	if q != nil {
		q.depth++
	}
}

func (q *quoteState) noClose() {
	if q != nil && q.depth > 0 {
		q.depth--
	}
}

func quotePairIndex(depth, count int) int {
	if count <= 1 || depth <= 0 {
		return 0
	}
	if depth >= count {
		return count - 1
	}
	return depth
}

func quotePairs(style *css.ComputedStyle) []quotePair {
	if style == nil {
		return defaultQuotePairs()
	}
	value := strings.TrimSpace(style.Quotes)
	if value == "" || strings.EqualFold(value, cssValueAuto) {
		return defaultQuotePairs()
	}
	if strings.EqualFold(value, cssValueNone) {
		return []quotePair{{}}
	}
	pairs, ok := parseQuotePairs(value)
	if !ok || len(pairs) == 0 {
		return defaultQuotePairs()
	}
	return pairs
}

func defaultQuotePairs() []quotePair {
	return []quotePair{
		{open: `"`, close: `"`},
		{open: `'`, close: `'`},
	}
}

func parseQuotePairs(value string) ([]quotePair, bool) {
	var pairs []quotePair
	for {
		value = strings.TrimLeft(value, " \t\r\n\f")
		if value == "" {
			return pairs, true
		}
		if value[0] != '"' && value[0] != '\'' {
			return nil, false
		}
		open, rest, ok := readCSSContentString(value)
		if !ok {
			return nil, false
		}
		rest = strings.TrimLeft(rest, " \t\r\n\f")
		if rest == "" || (rest[0] != '"' && rest[0] != '\'') {
			return nil, false
		}
		closeText, rest, ok := readCSSContentString(rest)
		if !ok {
			return nil, false
		}
		pairs = append(pairs, quotePair{open: open, close: closeText})
		value = rest
	}
}
