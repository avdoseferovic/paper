// Package dom provides a Maroto-friendly wrapper over golang.org/x/net/html.
package dom

import (
	"strings"

	"golang.org/x/net/html"
)

// Document is the parsed HTML document.
type Document struct {
	root      *html.Node
	styleText string
	linkHrefs []string
}

// Parse parses an HTML string and returns a Document.
func Parse(src string) (*Document, error) {
	root, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	doc := &Document{root: root}
	doc.styleText, doc.linkHrefs = extractStylesAndLinks(root)
	return doc, nil
}

// StyleText returns all concatenated <style> block contents.
// Backward-compatible: returns only inline <style> text, not the contents of
// <link rel="stylesheet"> sheets. Callers needing both should use StyleSources.
func (d *Document) StyleText() string { return d.styleText }

// StyleSources returns both the concatenated inline <style> text and the
// ordered list of <link rel="stylesheet"> href values (in DOM order).
// External stylesheet content is NOT loaded here; the caller is responsible
// for resolving each href via a safe resolver.
func (d *Document) StyleSources() (inlineCSS string, linkHrefs []string) {
	return d.styleText, d.linkHrefs
}

// Walk performs a depth-first traversal starting from the document body.
// The callback returns true to continue traversal, false to stop.
func (d *Document) Walk(fn func(*Node) bool) {
	body := findTag(d.root, "body")
	if body == nil {
		body = d.root
	}
	walkNode(body, fn)
}

// Root returns the underlying html.Node for direct access when needed.
func (d *Document) Root() *html.Node { return d.root }

// Node wraps a raw html.Node with Maroto-friendly accessors.
type Node struct {
	raw          *html.Node
	preformatted bool // whitespace preserved
}

// RawNode returns the underlying golang.org/x/net/html.Node.
// Callers needing cascadia selector matching can use this directly.
func (n *Node) RawNode() *html.Node { return n.raw }

// Tag returns the element tag name (lowercase) or "" for text nodes.
func (n *Node) Tag() string {
	if n.raw.Type != html.ElementNode {
		return ""
	}
	return n.raw.Data
}

// Attr returns the value of the named attribute, or "" if absent.
func (n *Node) Attr(name string) string {
	for _, a := range n.raw.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

// InlineStyle returns the value of the style="" attribute.
func (n *Node) InlineStyle() string { return n.Attr("style") }

// IsBlock returns whether this element is block-level by HTML5 default.
func (n *Node) IsBlock() bool { return IsBlockTag(n.Tag()) }

// IsInline returns whether this element renders inline by HTML5 default.
func (n *Node) IsInline() bool { return !n.IsBlock() && n.Tag() != "" }

// TextContent returns the concatenated text of all descendant text nodes.
// Whitespace is collapsed unless the element is inside <pre>.
func (n *Node) TextContent() string {
	return extractText(n.raw, isPreformatted(n.raw))
}

// Children returns the direct child Nodes.
func (n *Node) Children() []*Node {
	pre := isPreformatted(n.raw)
	var out []*Node
	for c := n.raw.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode || c.Type == html.TextNode {
			out = append(out, &Node{raw: c, preformatted: pre})
		}
	}
	return out
}

func walkNode(n *html.Node, fn func(*Node) bool) {
	if n.Type == html.ElementNode {
		node := &Node{raw: n}
		if !fn(node) {
			return
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkNode(c, fn)
	}
}

func findTag(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findTag(c, tag); found != nil {
			return found
		}
	}
	return nil
}

// extractStylesAndLinks walks the DOM once and returns the concatenated text
// of all <style> blocks together with the ordered list of href values found
// on every <link rel="stylesheet"> element.
func extractStylesAndLinks(n *html.Node) (string, []string) {
	var sb strings.Builder
	var links []string
	var walk func(*html.Node)
	walk = func(cur *html.Node) {
		if cur.Type == html.ElementNode {
			switch cur.Data {
			case "style":
				for c := cur.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.TextNode {
						sb.WriteString(c.Data)
					}
				}
			case "link":
				// Honour rel=stylesheet (case-insensitive) with href.
				var rel, href string
				for _, a := range cur.Attr {
					switch a.Key {
					case "rel":
						rel = strings.ToLower(a.Val)
					case "href":
						href = a.Val
					}
				}
				if rel == "stylesheet" && href != "" {
					links = append(links, href)
				}
			}
		}
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String(), links
}

func extractText(n *html.Node, pre bool) string {
	var sb strings.Builder
	var walk func(*html.Node, bool)
	walk = func(cur *html.Node, inPre bool) {
		if cur.Type == html.TextNode {
			text := cur.Data
			if !inPre {
				text = collapseWhitespace(text)
			}
			sb.WriteString(text)
			return
		}
		childPre := inPre
		if cur.Type == html.ElementNode && (cur.Data == "pre" || cur.Data == "code") {
			childPre = true
		}
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			walk(c, childPre)
		}
	}
	walk(n, pre)
	result := sb.String()
	if !pre {
		result = collapseWhitespace(result)
	}
	return result
}

func isPreformatted(n *html.Node) bool {
	for cur := n; cur != nil; cur = cur.Parent {
		if cur.Type == html.ElementNode && (cur.Data == "pre" || cur.Data == "code") {
			return true
		}
	}
	return false
}

func collapseWhitespace(s string) string {
	if s == "" {
		return ""
	}
	leading := isASCIISpace(s[0])
	trailing := isASCIISpace(s[len(s)-1])
	fields := strings.Fields(s)
	if len(fields) == 0 {
		if leading || trailing {
			return " "
		}
		return ""
	}
	result := strings.Join(fields, " ")
	if leading {
		result = " " + result
	}
	if trailing {
		result += " "
	}
	return result
}

func isASCIISpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}
