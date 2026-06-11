package pdf

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// synthFontDefJSON builds a font definition JSON document equivalent to the
// output of the makefont utility for a synthetic embedded font.
func synthFontDefJSON(t *testing.T, tp, name, file, diff string, size1, size2, originalSize int) []byte {
	t.Helper()
	cw := make([]int, 256)
	for i := range cw {
		cw[i] = 500
	}
	def := map[string]any{
		"Tp":   tp,
		"Name": name,
		"Up":   -100,
		"Ut":   50,
		"Cw":   cw,
		"Enc":  "cp1252",
		"Desc": map[string]any{
			"Ascent":    800,
			"Descent":   -200,
			"CapHeight": 700,
			"Flags":     32,
			"FontBBox": map[string]int{
				"Xmin": -100, "Ymin": -200, "Xmax": 1000, "Ymax": 900,
			},
			"ItalicAngle":  0,
			"StemV":        80,
			"MissingWidth": 500,
		},
	}
	if file != "" {
		def["File"] = file
	}
	if diff != "" {
		def["Diff"] = diff
	}
	if size1 > 0 {
		def["Size1"] = size1
		def["Size2"] = size2
	}
	if originalSize > 0 {
		def["OriginalSize"] = originalSize
	}
	b, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("marshal synthetic font definition: %v", err)
	}
	return b
}

func TestAddFontFromBytesEmbeddedTrueType(t *testing.T) {
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.SetCompression(false)

	zBytes := bytes.Repeat([]byte{0xAB}, 34)
	jsonBytes := synthFontDefJSON(t, "TrueType", "SynthTT", "synthtt.z", "32 /space", 0, 0, 34)
	f.AddFontFromBytes("SynthTT", "", jsonBytes, zBytes)
	if f.Err() {
		t.Fatalf("AddFontFromBytes errored: %v", f.Error())
	}

	// A second font sharing the same encoding differences exercises the diff
	// de-duplication path.
	jsonBytes2 := synthFontDefJSON(t, "TrueType", "SynthTTB", "", "32 /space", 0, 0, 0)
	f.AddFontFromBytes("SynthTT", "B", jsonBytes2, nil)
	if f.Err() {
		t.Fatalf("AddFontFromBytes for bold style errored: %v", f.Error())
	}
	if len(f.diffs) != 1 {
		t.Fatalf("expected shared encoding diff to be registered once, got %d entries", len(f.diffs))
	}

	f.AddPage()
	f.SetFont("SynthTT", "", 12)
	f.Cell(40, 10, "embedded truetype")
	f.SetFont("SynthTT", "B", 12)
	f.Cell(40, 10, "bold")

	out := string(mustOutput(t, f))
	for _, marker := range []string{
		"/Subtype /TrueType",
		"/FontFile2",
		"/Differences [32 /space]",
		"/BaseFont /SynthTT",
		"/MissingWidth 500",
	} {
		if !strings.Contains(out, marker) {
			t.Fatalf("expected output to contain %q", marker)
		}
	}
}

func TestAddFontFromReaderType1LoadsFontFileFromDisk(t *testing.T) {
	dir := t.TempDir()
	fontFile := make([]byte, 40)
	for i := range fontFile {
		fontFile[i] = byte(i)
	}
	if err := os.WriteFile(filepath.Join(dir, "synth1.pfb"), fontFile, 0o644); err != nil {
		t.Fatalf("write synthetic Type1 font file: %v", err)
	}

	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.SetCompression(false)
	f.SetFontLocation(dir)

	jsonBytes := synthFontDefJSON(t, "Type1", "SynthT1", "synth1.pfb", "", 10, 30, 0)
	f.AddFontFromReader("SynthT1", "", bytes.NewReader(jsonBytes))
	if f.Err() {
		t.Fatalf("AddFontFromReader errored: %v", f.Error())
	}

	f.AddPage()
	f.SetFont("SynthT1", "", 12)
	f.Cell(40, 10, "type1")

	out := string(mustOutput(t, f))
	for _, marker := range []string{
		"/Subtype /Type1",
		"/Length2 30 /Length3 0",
		"/FontFile ",
	} {
		if !strings.Contains(out, marker) {
			t.Fatalf("expected output to contain %q", marker)
		}
	}
}

func TestAddFontLoadsDefinitionFromFontDirectory(t *testing.T) {
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.SetFontLocation(filepath.Join("embedded", "fonts"))
	f.AddFont("MyHelv", "", "helvetica.json")
	if f.Err() {
		t.Fatalf("AddFont errored: %v", f.Error())
	}

	f.AddPage()
	f.SetFont("MyHelv", "", 12)
	f.Cell(40, 10, "from disk definition")
	mustOutput(t, f)
}

func TestAddFontMissingDefinitionFileSetsError(t *testing.T) {
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.AddFont("No Such Font", "B", "")
	if !f.Err() {
		t.Fatal("expected error when font definition file is missing")
	}
}

func TestAddUTF8FontMissingFontFileSetsError(t *testing.T) {
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.AddUTF8Font("No Such Font", "", "")
	if !f.Err() {
		t.Fatal("expected error when UTF-8 font file is missing")
	}
}

func TestAddUTF8FontLoadsFromFontDirectory(t *testing.T) {
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.SetFontLocation(filepath.Join("..", "..", "docs", "assets", "fonts"))
	f.AddUTF8Font("ArialUni", "", "arial-unicode-ms.ttf")
	if f.Err() {
		t.Fatalf("AddUTF8Font errored: %v", f.Error())
	}

	// Adding the same family/style again is a no-op.
	f.AddUTF8Font("ArialUni", "", "arial-unicode-ms.ttf")
	if f.Err() {
		t.Fatalf("repeated AddUTF8Font errored: %v", f.Error())
	}

	f.AddPage()
	f.SetFont("ArialUni", "", 12)
	f.Write(5, "Zdravo")
	mustOutput(t, f)
}

// mapFontLoader serves font resources from memory. Open returns an
// io.ReadCloser so the loader close path is exercised.
type mapFontLoader struct {
	files map[string][]byte
}

func (l mapFontLoader) Open(name string) (io.Reader, error) {
	b, ok := l.files[name]
	if !ok {
		return nil, errors.New("font resource not found")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// plainFontLoader serves font resources without implementing io.Closer.
type plainFontLoader struct {
	files map[string][]byte
}

func (l plainFontLoader) Open(name string) (io.Reader, error) {
	b, ok := l.files[name]
	if !ok {
		return nil, errors.New("font resource not found")
	}
	return bytes.NewReader(b), nil
}

func TestSetFontLoaderLoadsDefinitionAndFallsBackToDisk(t *testing.T) {
	helv, err := os.ReadFile(filepath.Join("embedded", "fonts", "helvetica.json"))
	if err != nil {
		t.Fatalf("read helvetica definition fixture: %v", err)
	}

	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.SetFontLoader(mapFontLoader{files: map[string][]byte{"helvetica.json": helv}})
	f.AddFont("LoadedHelv", "", "helvetica.json")
	if f.Err() {
		t.Fatalf("AddFont via loader errored: %v", f.Error())
	}

	// The loader does not know courier.json, so AddFont falls back to the
	// configured font directory.
	f.SetFontLocation(filepath.Join("embedded", "fonts"))
	f.AddFont("DiskCour", "", "courier.json")
	if f.Err() {
		t.Fatalf("AddFont with loader fallback errored: %v", f.Error())
	}

	f.AddPage()
	f.SetFont("LoadedHelv", "", 12)
	f.Cell(40, 10, "loader")
	f.SetFont("DiskCour", "", 12)
	f.Cell(40, 10, "fallback")
	mustOutput(t, f)
}

func TestOutputLoadsEmbeddedFontFileFromLoader(t *testing.T) {
	zBytes := bytes.Repeat([]byte{0xCD}, 25)

	for name, loader := range map[string]FontLoader{
		"closer":    mapFontLoader{files: map[string][]byte{"fromloader.z": zBytes}},
		"nonCloser": plainFontLoader{files: map[string][]byte{"fromloader.z": zBytes}},
	} {
		t.Run(name, func(t *testing.T) {
			f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
			f.SetCompression(false)
			f.SetFontLoader(loader)

			jsonBytes := synthFontDefJSON(t, "TrueType", "LoaderTT", "fromloader.z", "", 0, 0, 25)
			f.AddFontFromReader("LoaderTT", "", bytes.NewReader(jsonBytes))
			if f.Err() {
				t.Fatalf("AddFontFromReader errored: %v", f.Error())
			}

			f.AddPage()
			f.SetFont("LoaderTT", "", 12)
			f.Cell(40, 10, "loader font file")

			out := string(mustOutput(t, f))
			if !strings.Contains(out, "/Length1 25") {
				t.Fatal("expected font file stream object with original length")
			}
		})
	}
}

func TestAddFontFromReaderDeduplicatesAndRegistersDiff(t *testing.T) {
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	jsonBytes := synthFontDefJSON(t, "TrueType", "DiffTT", "", "32 /space", 0, 0, 0)
	f.AddFontFromReader("DiffTT", "", bytes.NewReader(jsonBytes))
	if f.Err() {
		t.Fatalf("AddFontFromReader errored: %v", f.Error())
	}
	if len(f.diffs) != 1 {
		t.Fatalf("expected encoding diff to be registered, got %d entries", len(f.diffs))
	}

	// Same family/style again is a no-op, same diff again is de-duplicated.
	f.AddFontFromReader("DiffTT", "", bytes.NewReader(jsonBytes))
	jsonBytes2 := synthFontDefJSON(t, "TrueType", "DiffTT2", "", "32 /space", 0, 0, 0)
	f.AddFontFromReader("DiffTT2", "", bytes.NewReader(jsonBytes2))
	if f.Err() {
		t.Fatalf("AddFontFromReader errored: %v", f.Error())
	}
	if len(f.diffs) != 1 {
		t.Fatalf("expected shared diff to be stored once, got %d entries", len(f.diffs))
	}

	// Once an error is set, AddFontFromReader becomes a no-op.
	f.SetErrorf("sticky error")
	f.AddFontFromReader("DiffTT3", "", bytes.NewReader(jsonBytes))
	if _, ok := f.fonts["difftt3"]; ok {
		t.Fatal("expected no font registration after an error is set")
	}
}

func TestOutputFailsWhenFontFileMissingFromDisk(t *testing.T) {
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.SetFontLocation(t.TempDir())

	jsonBytes := synthFontDefJSON(t, "TrueType", "MissingTT", "missing.z", "", 0, 0, 10)
	f.AddFontFromReader("MissingTT", "", bytes.NewReader(jsonBytes))
	if f.Err() {
		t.Fatalf("AddFontFromReader errored: %v", f.Error())
	}

	f.AddPage()
	f.SetFont("MissingTT", "", 12)
	f.Cell(40, 10, "missing font file")

	var buf bytes.Buffer
	if err := f.Output(&buf); err == nil {
		t.Fatal("expected output to fail when the referenced font file cannot be loaded")
	}
}
