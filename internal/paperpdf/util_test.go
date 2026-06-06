package paperpdf

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSliceUncompressInvalidDataReturnsError(t *testing.T) {
	t.Parallel()

	out, err := sliceUncompress([]byte("not zlib data"))
	if err == nil {
		t.Fatal("expected invalid zlib data to return an error")
	}
	if out != nil {
		t.Fatalf("expected no output for invalid zlib data, got %q", out)
	}
}

func TestRemoveIntPreservesSliceWhenValueIsMissing(t *testing.T) {
	t.Parallel()

	input := []int{10, 20, 30}
	got := removeInt(input, 99)

	if !reflect.DeepEqual(got, input) {
		t.Fatalf("expected missing value removal to preserve slice; got %v", got)
	}
}

func TestUTF8ToUTF16SupplementaryPlaneUsesSurrogatePair(t *testing.T) {
	t.Parallel()

	got := []byte(utf8toutf16("😀", false))
	want := []byte{0xD8, 0x3D, 0xDE, 0x00}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestTemplateKeyListSortsWhenRequested(t *testing.T) {
	t.Parallel()

	got := templateKeyList(map[string]Template{
		"z": nil,
		"a": nil,
		"m": nil,
	}, true)

	want := []string{"a", "m", "z"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected sorted keys %v, got %v", want, got)
	}
}

func TestAddUTF8FontFromBytesRecordsParseErrorWithoutStdout(t *testing.T) {
	f := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})

	stdout := captureStdout(t, func() {
		f.AddUTF8FontFromBytes("broken", "", []byte{0, 0, 0, 0})
	})

	if stdout != "" {
		t.Fatalf("expected no stdout while parsing a bad font, got %q", stdout)
	}
	if f.Error() == nil {
		t.Fatal("expected bad font parse to be recorded on Fpdf")
	}
	if !strings.Contains(f.Error().Error(), "not a TrueType font") {
		t.Fatalf("expected TrueType parse error, got %q", f.Error())
	}
}

func TestAddUTF8FontFromBytesTruncatedTrueTypeDoesNotPanicOrWriteStdout(t *testing.T) {
	f := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})

	var recovered any
	stdout := captureStdout(t, func() {
		defer func() {
			recovered = recover()
		}()
		f.AddUTF8FontFromBytes("broken", "", []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x01})
	})

	if recovered != nil {
		t.Fatalf("expected truncated TrueType-like bytes not to panic, got %v", recovered)
	}
	if stdout != "" {
		t.Fatalf("expected no stdout while parsing a bad font, got %q", stdout)
	}
	if f.Error() == nil {
		t.Fatal("expected bad font parse to be recorded on Fpdf")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return string(out)
}
