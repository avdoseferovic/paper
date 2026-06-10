package main

import _ "embed"

// body is the HTML document rendered by html-demo, embedded at build time
// from assets/body.html so the binary stays self-contained without inlining
// the HTML as a string literal.
//
//go:embed assets/body.html
var body string
