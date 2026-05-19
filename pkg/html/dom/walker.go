package dom

// blockTags is the set of HTML5 block-level element tag names per the default UA stylesheet.
var blockTags = map[string]bool{
	"address": true, "article": true, "aside": true, "blockquote": true,
	"dd": true, "details": true, "dialog": true, "div": true,
	"dl": true, "dt": true, "fieldset": true, "figcaption": true,
	"figure": true, "footer": true, "form": true, "h1": true,
	"h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"header": true, "hgroup": true, "hr": true, "li": true,
	"main": true, "nav": true, "ol": true, "p": true,
	"pre": true, "section": true, "summary": true, "table": true,
	"ul": true,
}

// IsBlockTag reports whether the given tag name is block-level by the HTML5 default UA stylesheet.
func IsBlockTag(tag string) bool {
	return blockTags[tag]
}
