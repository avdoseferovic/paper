package pdf

// Embedded standard fonts and codepage maps.
//
// The core font metric definitions (Courier/Helvetica/Times/ZapfDingbats) and
// the codepage translation maps live as data files under embedded/ and are
// served via the embed.FS values below. Each file is read on demand — only the
// fonts/codepages actually used by a document are loaded.

import (
	"embed"
	"strings"
)

//go:embed embedded/fonts/*.json
var coreFontFS embed.FS

//go:embed embedded/maps/*.map
var codepageMapFS embed.FS

func (f *PDF) coreFontReader(familyStr, styleStr string) *strings.Reader {
	key := familyStr + styleStr
	data, err := coreFontFS.ReadFile("embedded/fonts/" + key + ".json")
	if err != nil {
		f.SetErrorf("could not locate \"%s\" among embedded core font definition files", key)
		return nil
	}
	return strings.NewReader(string(data))
}

// embeddedCodepageMap returns the embedded .map definition for the given
// codepage and whether one exists.
func embeddedCodepageMap(cpStr string) (string, bool) {
	data, err := codepageMapFS.ReadFile("embedded/maps/" + cpStr + ".map")
	if err != nil {
		return "", false
	}
	return string(data), true
}
