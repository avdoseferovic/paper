package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCustomUTF8FontPDFContainsCIDObjects(t *testing.T) {
	fontBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "assets", "fonts", "arial-unicode-ms.ttf"))
	if err != nil {
		t.Fatalf("read custom font fixture: %v", err)
	}

	pdf := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	pdf.SetCompression(false)
	pdf.AddUTF8FontFromBytes("arial-unicode-ms", "", fontBytes)
	pdf.AddPage()
	pdf.SetFont("arial-unicode-ms", "", 12)
	pdf.Write(5, "Zdravo, ćao 漢字")

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		t.Fatalf("output custom font PDF: %v", err)
	}

	body := out.String()
	for _, marker := range []string{
		"%PDF",
		"/Subtype /CIDFontType2",
		"/W [",
		"/FontFile2",
		"/ToUnicode",
		"/CIDToGIDMap",
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected custom-font PDF output to contain %q", marker)
		}
	}
}

func TestAddUTF8FontFromBytesAcceptsWOFF1(t *testing.T) {
	fontBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "assets", "fonts", "arial-unicode-ms.ttf"))
	if err != nil {
		t.Fatalf("read custom font fixture: %v", err)
	}
	woffBytes := makeWOFF1FromSFNT(t, fontBytes, true)

	pdf := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	pdf.SetCompression(false)
	pdf.AddUTF8FontFromBytes("arial-woff", "", woffBytes)
	pdf.AddPage()
	pdf.SetFont("arial-woff", "", 12)
	pdf.Write(5, "Zdravo, ćao 漢字")
	if err := pdf.Error(); err != nil {
		t.Fatalf("register/render WOFF font: %v", err)
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		t.Fatalf("output WOFF-backed custom font PDF: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, "/Subtype /CIDFontType2") || !strings.Contains(body, "/FontFile2") {
		t.Fatalf("expected WOFF-backed PDF output to contain embedded CID font objects")
	}
}

func TestAddPageAfterLateUTF8FontStyleFallbackReturns(t *testing.T) {
	fontBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "assets", "fonts", "arial-unicode-ms.ttf"))
	if err != nil {
		t.Fatalf("read custom font fixture: %v", err)
	}

	pdf := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	pdf.SetCompression(false)
	pdf.AddPage()
	pdf.AddUTF8FontFromBytes("arial-unicode-ms", "", fontBytes)
	pdf.SetFont("arial-unicode-ms", "B", 5.7)
	pdf.Text(5, 5, "▲")
	if err := pdf.Error(); err != nil {
		t.Fatalf("missing bold style should fall back to regular UTF-8 font: %v", err)
	}

	done := make(chan struct{})
	go func() {
		pdf.AddPage()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("AddPage blocked after late UTF-8 font registration")
	}
}

func TestUTF8FontSemiboldStyleFallbacksToRegisteredFace(t *testing.T) {
	fontBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "assets", "fonts", "arial-unicode-ms.ttf"))
	if err != nil {
		t.Fatalf("read custom font fixture: %v", err)
	}

	pdf := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	pdf.AddUTF8FontFromBytes("arial-unicode-ms", "", fontBytes)
	pdf.AddPage()
	pdf.SetFont("arial-unicode-ms", "M", 12)
	pdf.Text(5, 5, "semibold fallback")
	if err := pdf.Error(); err != nil {
		t.Fatalf("missing semibold style should fall back to regular UTF-8 font: %v", err)
	}
	if pdf.fontStyle != "" {
		t.Fatalf("expected semibold fallback to regular style, got %q", pdf.fontStyle)
	}
	mustOutput(t, pdf)
}

func TestOutputReturnsErrorWhenUTF8FontSubsettingFails(t *testing.T) {
	fontBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "assets", "fonts", "arial-unicode-ms.ttf"))
	if err != nil {
		t.Fatalf("read custom font fixture: %v", err)
	}

	pdf := NewCustom(&InitType{})
	pdf.AddUTF8FontFromBytes("arial-unicode-ms", "", fontBytes)
	if pdf.Error() != nil {
		t.Fatalf("register custom font: %v", pdf.Error())
	}

	font := pdf.fonts["arial-unicode-ms"]
	font.utf8File.fileReader.array = font.utf8File.fileReader.array[:64]

	pdf.AddPage()
	pdf.SetFont("arial-unicode-ms", "", 12)
	pdf.Write(5, "ćao")

	var out bytes.Buffer
	err = pdf.Output(&out)
	if err == nil {
		t.Fatal("expected Output to report UTF-8 font subsetting failure")
	}
	if !strings.Contains(err.Error(), "font") {
		t.Fatalf("expected font subsetting error, got %v", err)
	}
}

func TestAliasNbPagesUTF8FontRendersDigits(t *testing.T) {
	fontBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "assets", "fonts", "arial-unicode-ms.ttf"))
	if err != nil {
		t.Fatalf("read custom font fixture: %v", err)
	}

	pdf := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	pdf.SetCompression(false)
	pdf.AliasNbPages("")
	pdf.AddUTF8FontFromBytes("arial-unicode-ms", "", fontBytes)
	pdf.AddPage()
	pdf.SetFont("arial-unicode-ms", "", 12)
	pdf.Write(5, "Page 1 of {nb}")
	pdf.AddPage()
	pdf.Write(5, "Page 2 of {nb}")

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		t.Fatalf("output alias PDF: %v", err)
	}
	body := out.Bytes()

	// The alias is replaced with identity UTF-16BE CIDs ("2" -> 0x0032), so
	// the page content must contain the digit CID.
	if !bytes.Contains(body, []byte{0x00, '2'}) {
		t.Fatal("expected page content to contain the identity CID for digit 2")
	}
	// The digit CID must be a real glyph: present in ToUnicode...
	if !bytes.Contains(body, []byte("<0032> <0032>")) {
		t.Fatal("expected ToUnicode CMap to map CID 0x32 to U+0032")
	}
	// ...and mapped to a non-.notdef glyph in the subset font.
	var utf8Font fontDefType
	for _, font := range pdf.fonts {
		if font.utf8File != nil {
			utf8Font = font
		}
	}
	if utf8Font.utf8File == nil {
		t.Fatal("missing UTF-8 font definition")
	}
	if glyph := utf8Font.utf8File.codeSymbolDictionary[0x32]; glyph == 0 {
		t.Fatalf("expected CID 0x32 to map to a real glyph, got glyph %d", glyph)
	}
}

func TestUTF8FontSubsettingOutputIsDeterministic(t *testing.T) {
	fontBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "assets", "fonts", "arial-unicode-ms.ttf"))
	if err != nil {
		t.Fatalf("read custom font fixture: %v", err)
	}

	render := func() []byte {
		t.Helper()
		pdf := NewCustom(&InitType{
			OrientationStr: "P",
			UnitStr:        "mm",
			SizeStr:        "A4",
		})
		pdf.SetCompression(false)
		pdf.SetCatalogSort(true)
		pdf.SetCreationDate(time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC))
		pdf.SetModificationDate(time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC))
		pdf.AddUTF8FontFromBytes("arial-unicode-ms", "", fontBytes)
		pdf.AddPage()
		pdf.SetFont("arial-unicode-ms", "", 12)
		pdf.Write(5, "Brandt, Theresa — Warnsignale: ćao 漢字")

		var out bytes.Buffer
		if err := pdf.Output(&out); err != nil {
			t.Fatalf("output custom font PDF: %v", err)
		}
		return out.Bytes()
	}

	first := render()
	for i := 0; i < 10; i++ {
		if got := render(); !bytes.Equal(first, got) {
			t.Fatalf("UTF-8 font subset output differed on render %d", i+2)
		}
	}
}

func TestUTF8FontFileGlyphDataRejectsOneByteGlyph(t *testing.T) {
	utf := &utf8FontFile{}

	_, ok := utf.glyphData([]byte{0x80}, 0, 1)

	if ok {
		t.Fatal("expected one-byte glyph to be rejected")
	}
	if utf.err == nil {
		t.Fatal("expected glyph error")
	}
}

func TestUTF8FontFileCompositeGlyphRejectsTruncatedComponent(t *testing.T) {
	utf := &utf8FontFile{}
	data := []byte{
		0x80, 0x00,
		0, 0, 0, 0, 0, 0, 0, 0,
		0x00, symbolContinue,
	}

	utf.rewriteCompositeGlyph(data, 1, map[int]int{})

	if utf.err == nil {
		t.Fatal("expected truncated composite glyph error")
	}
}

func makeWOFF1FromSFNT(t *testing.T, sfnt []byte, compressTables bool) []byte {
	t.Helper()
	if len(sfnt) < 12 {
		t.Fatal("sfnt fixture is truncated")
	}
	numTables := int(binary.BigEndian.Uint16(sfnt[4:6]))
	directoryEnd := 12 + numTables*16
	if numTables < 1 || directoryEnd > len(sfnt) {
		t.Fatalf("invalid sfnt table directory: numTables=%d len=%d", numTables, len(sfnt))
	}

	type table struct {
		tag        []byte
		checksum   []byte
		offset     int
		length     int
		data       []byte
		compData   []byte
		compLength int
	}
	tables := make([]table, 0, numTables)
	for i := 0; i < numTables; i++ {
		record := sfnt[12+i*16 : 12+(i+1)*16]
		offset := int(binary.BigEndian.Uint32(record[8:12]))
		length := int(binary.BigEndian.Uint32(record[12:16]))
		if offset < 0 || offset > len(sfnt) || offset+length > len(sfnt) {
			t.Fatalf("sfnt table %d exceeds font data", i)
		}
		tableData := append([]byte(nil), sfnt[offset:offset+length]...)
		compData := tableData
		if compressTables {
			var buf bytes.Buffer
			zw := zlib.NewWriter(&buf)
			if _, err := zw.Write(tableData); err != nil {
				t.Fatalf("compress table %d: %v", i, err)
			}
			if err := zw.Close(); err != nil {
				t.Fatalf("close compressor for table %d: %v", i, err)
			}
			if buf.Len() < len(tableData) {
				compData = buf.Bytes()
			}
		}
		tables = append(tables, table{
			tag:        append([]byte(nil), record[0:4]...),
			checksum:   append([]byte(nil), record[4:8]...),
			offset:     offset,
			length:     length,
			data:       tableData,
			compData:   append([]byte(nil), compData...),
			compLength: len(compData),
		})
	}

	totalLength := 44 + numTables*20
	for _, table := range tables {
		totalLength = paddedLength(totalLength) + paddedLength(table.compLength)
	}
	out := make([]byte, 44+numTables*20)
	copy(out[0:4], []byte{'w', 'O', 'F', 'F'})
	copy(out[4:8], sfnt[0:4])
	binary.BigEndian.PutUint32(out[8:12], uint32(totalLength)) // #nosec G115 -- test fixture is bounded.
	binary.BigEndian.PutUint16(out[12:14], uint16(numTables))  // #nosec G115 -- table count is from a uint16 field.
	binary.BigEndian.PutUint32(out[16:20], uint32(len(sfnt)))  // #nosec G115 -- test fixture is bounded.

	offset := 44 + numTables*20
	for i, table := range tables {
		offset = paddedLength(offset)
		entry := out[44+i*20 : 44+(i+1)*20]
		copy(entry[0:4], table.tag)
		binary.BigEndian.PutUint32(entry[4:8], uint32(offset))            // #nosec G115 -- test fixture is bounded.
		binary.BigEndian.PutUint32(entry[8:12], uint32(table.compLength)) // #nosec G115 -- test fixture is bounded.
		binary.BigEndian.PutUint32(entry[12:16], uint32(table.length))    // #nosec G115 -- test fixture is bounded.
		copy(entry[16:20], table.checksum)
		if len(out) < offset {
			out = append(out, make([]byte, offset-len(out))...)
		}
		out = append(out, table.compData...)
		if padding := paddedLength(table.compLength) - table.compLength; padding > 0 {
			out = append(out, make([]byte, padding)...)
		}
		offset += paddedLength(table.compLength)
	}
	if len(out) != totalLength {
		t.Fatalf("WOFF length = %d, want %d", len(out), totalLength)
	}
	return out
}
