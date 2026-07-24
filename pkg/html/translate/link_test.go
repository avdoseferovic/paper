package translate

import "testing"

func TestSanitizedHyperlink(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		href     string
		wantLink string
		wantOK   bool
	}{
		"https":                    {"https://example.com/a?b=1", "https://example.com/a?b=1", true},
		"http":                     {"http://example.com", "http://example.com", true},
		"mailto":                   {"mailto:a@example.com", "mailto:a@example.com", true},
		"tel":                      {"tel:+15551234", "tel:+15551234", true},
		"uppercase scheme":         {"HTTPS://example.com", "HTTPS://example.com", true},
		"surrounding space":        {"  https://example.com  ", "https://example.com", true},
		"relative path":            {"docs/page.html", "docs/page.html", true},
		"root relative":            {"/docs/page.html", "/docs/page.html", true},
		"path with colon":          {"docs/a:b", "docs/a:b", true},
		"javascript":               {"javascript:alert(1)", "", false},
		"mixed case javascript":    {"JaVaScRiPt:alert(1)", "", false},
		"padded javascript":        {"   javascript:alert(1)", "", false},
		"control char smuggling":   {"java\tscript:alert(1)", "", false},
		"newline smuggling":        {"java\nscript:alert(1)", "", false},
		"file":                     {"file:///etc/passwd", "", false},
		"smb credential leak":      {"smb://attacker.example/share", "", false},
		"data":                     {"data:text/html,<script>x</script>", "", false},
		"vbscript":                 {"vbscript:msgbox", "", false},
		"empty":                    {"", "", false},
		"whitespace only":          {"   ", "", false},
		"bare colon is relative":   {":nope", ":nope", true},
		"scheme starting in digit": {"1http://example.com", "1http://example.com", true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			link, ok := sanitizedHyperlink(tc.href)

			if ok != tc.wantOK || link != tc.wantLink {
				t.Fatalf("sanitizedHyperlink(%q) = (%q, %v), want (%q, %v)", tc.href, link, ok, tc.wantLink, tc.wantOK)
			}
		})
	}
}

func TestURIScheme(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		reference  string
		wantScheme string
		wantOK     bool
	}{
		"simple":            {"https://example.com", "https", true},
		"lowercased":        {"MAILTO:a@b.c", "mailto", true},
		"with plus":         {"view-source+x:y", "view-source+x", true},
		"no colon":          {"example.com/path", "", false},
		"colon after slash": {"a/b:c", "", false},
		"leading colon":     {":x", "", false},
		"digit start":       {"1a:b", "", false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			scheme, ok := uriScheme(tc.reference)

			if ok != tc.wantOK || scheme != tc.wantScheme {
				t.Fatalf("uriScheme(%q) = (%q, %v), want (%q, %v)", tc.reference, scheme, ok, tc.wantScheme, tc.wantOK)
			}
		})
	}
}
