package main

import _ "embed"

// body is the HTML document rendered by survey-report, embedded at build time
// from assets/body.html.
//
//go:embed assets/body.html
var body string
