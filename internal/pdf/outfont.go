package pdf

import (
	"bytes"
	"strconv"
	"sync"
)

// cellBufferPool reuses the scratch buffer CellFormat assembles each cell in.
// CellFormat runs once per drawn cell and its buffer escapes to the heap, which
// made the declaration alone ~15% of allocated objects. The assembled bytes are
// copied out via String() before the call returns, so nothing outlives the
// pooled buffer. The pool is also what keeps this safe under concurrent page
// rendering, where several documents assemble cells at the same time.
var cellBufferPool = sync.Pool{
	New: func() any {
		return new(fmtBuffer)
	},
}

// takeCellBuffer returns a reset scratch buffer and the function that returns it
// to the pool.
func takeCellBuffer() (*fmtBuffer, func()) {
	buf, ok := cellBufferPool.Get().(*fmtBuffer)
	if !ok {
		buf = new(fmtBuffer)
	}
	buf.Reset()
	return buf, func() { cellBufferPool.Put(buf) }
}

// activeOutBuffer returns the buffer that out/outf would write to: the current
// page's content stream while rendering, or the document buffer otherwise.
func (f *PDF) activeOutBuffer() *bytes.Buffer {
	if f.state == 2 {
		return f.pages[f.page]
	}
	return &f.buffer.Buffer
}

// outFontSelect writes the text font selection operator for fontID at sizePt.
//
// It produces exactly what outf("BT /F%s %.2f Tf ET", fontID, sizePt) would,
// but without going through fmt: the variadic call boxes sizePt into an any,
// which allocates. SetFont runs once per drawn text run, making this one of the
// hottest output lines in the library (profiled at ~18% of all allocated
// objects), so the boxing is worth avoiding here even though outf is fine
// everywhere else.
func (f *PDF) outFontSelect(fontID string, sizePt float64) {
	buf := f.activeOutBuffer()
	buf.WriteString("BT /F")
	buf.WriteString(fontID)
	buf.WriteByte(' ')
	var scratch [32]byte
	buf.Write(strconv.AppendFloat(scratch[:0], sizePt, 'f', 2, 64))
	buf.WriteString(" Tf ET\n")
}

// outTextShow writes the text-showing operator for already-escaped text drawn
// at (x, y) in user units, wrapped in the current text color when the fill and
// text colors differ.
//
// It is byte-for-byte equivalent to the sprintf-based form it replaces in
// PDF.Text, but writes straight into the page buffer. PDF.Text runs once per
// drawn text run, and its intermediate string was the largest remaining single
// allocation site in the render path (~18% of allocated objects).
func (f *PDF) outTextShow(x, y float64, escaped string) {
	buf := f.activeOutBuffer()
	var scratch [32]byte

	if f.colorFlag {
		buf.WriteString("q ")
		buf.WriteString(f.color.text.str)
		buf.WriteByte(' ')
	}

	buf.WriteString("BT ")
	buf.Write(strconv.AppendFloat(scratch[:0], x*f.k, 'f', 2, 64))
	buf.WriteByte(' ')
	buf.Write(strconv.AppendFloat(scratch[:0], (f.h-y)*f.k, 'f', 2, 64))
	buf.WriteString(" Td (")
	buf.WriteString(escaped)
	buf.WriteString(") Tj ET")

	if f.colorFlag {
		buf.WriteString(" Q")
	}
	buf.WriteByte('\n')
}
