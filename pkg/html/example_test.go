package html_test

import (
	"bytes"
	"testing"

	maroto "github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/html"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExample_FromString_GeneratesPDF(t *testing.T) {
	t.Parallel()

	htmlInput := `
<html>
<head><style>h1 { color: #003366 }</style></head>
<body>
  <h1>Invoice #123</h1>
  <p>Hello <b>world</b>! This is a <i>test</i> document.</p>
  <table>
    <tr><th>Item</th><th>Price</th></tr>
    <tr><td>Widget</td><td>$10</td></tr>
    <tr><td>Gadget</td><td>$25</td></tr>
  </table>
  <ul>
    <li>First item</li>
    <li>Second item with <a href="https://example.com">a link</a></li>
  </ul>
</body>
</html>`

	rows, err := html.FromString(htmlInput)
	require.NoError(t, err)
	assert.NotEmpty(t, rows)

	m := maroto.New()
	require.NoError(t, m.AddHTML(htmlInput))

	doc, err := m.Generate()
	require.NoError(t, err)

	pdfBytes := doc.GetBytes()
	require.NotEmpty(t, pdfBytes)
	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-")), "output should start with PDF magic bytes")
	assert.Greater(t, len(pdfBytes), 1000, "PDF should be larger than 1KB")
}
