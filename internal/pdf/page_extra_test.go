package pdf

import "testing"

func TestMarginsGetSet(t *testing.T) {
	f := NewCustom(&InitType{UnitStr: "mm"})
	f.SetMargins(15, 20, 25)
	f.SetLeftMargin(11)
	f.SetTopMargin(12)
	f.SetRightMargin(13)
	l, top, r, b := f.GetMargins()
	if l != 11 || top != 12 || r != 13 {
		t.Fatalf("GetMargins = %v,%v,%v,%v", l, top, r, b)
	}
}

func TestCellMarginGetSet(t *testing.T) {
	f := NewCustom(&InitType{})
	f.SetCellMargin(3.5)
	if got := f.GetCellMargin(); got != 3.5 {
		t.Fatalf("GetCellMargin = %v", got)
	}
}

func TestGetPageSizeReflectsOrientation(t *testing.T) {
	p := NewCustom(&InitType{OrientationStr: "P", UnitStr: "pt", SizeStr: "A4"})
	pw, ph := p.GetPageSize()
	if pw >= ph {
		t.Fatalf("portrait should have width < height, got %v x %v", pw, ph)
	}
	l := NewCustom(&InitType{OrientationStr: "L", UnitStr: "pt", SizeStr: "A4"})
	lw, lh := l.GetPageSize()
	if lw <= lh {
		t.Fatalf("landscape should have width > height, got %v x %v", lw, lh)
	}
}

func TestAutoPageBreakGetSet(t *testing.T) {
	f := NewCustom(&InitType{})
	f.SetAutoPageBreak(true, 17)
	auto, margin := f.GetAutoPageBreak()
	if !auto || margin != 17 {
		t.Fatalf("GetAutoPageBreak = %v, %v", auto, margin)
	}
}

func TestPageCountAndNo(t *testing.T) {
	f := NewCustom(&InitType{})
	f.AddPage()
	f.AddPage()
	if got := f.PageCount(); got != 2 {
		t.Fatalf("PageCount = %d", got)
	}
	if got := f.PageNo(); got != 2 {
		t.Fatalf("PageNo = %d", got)
	}
}

func TestSetPageMovesCurrentPage(t *testing.T) {
	f := NewCustom(&InitType{})
	f.AddPage()
	f.AddPage()
	f.SetPage(1)
	if f.PageNo() != 1 {
		t.Fatalf("SetPage(1) -> PageNo = %d", f.PageNo())
	}
	if f.Err() {
		t.Fatalf("SetPage errored: %v", f.Error())
	}
}

func TestPageSizePerPage(t *testing.T) {
	f := NewCustom(&InitType{UnitStr: "pt"})
	f.AddPageFormat("P", f.GetPageSizeStr("A5"))
	w, h, _ := f.PageSize(1)
	if w <= 0 || h <= 0 {
		t.Fatalf("PageSize = %v x %v", w, h)
	}
}

func TestGetPageSizeStrKnownSize(t *testing.T) {
	f := NewCustom(&InitType{UnitStr: "pt"})
	sz := f.GetPageSizeStr("A4")
	if sz.Wd <= 0 || sz.Ht <= 0 {
		t.Fatalf("A4 size = %+v", sz)
	}
}

func TestSetPageBox(t *testing.T) {
	f := NewCustom(&InitType{UnitStr: "pt"})
	f.AddPage()
	f.SetPageBox("crop", 10, 10, 100, 100)
	if f.Err() {
		t.Fatalf("SetPageBox errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestLnDefaultAndExplicit(t *testing.T) {
	f := readyPDF(t)
	f.Cell(40, 10, "line")
	yBefore := f.GetY()
	f.Ln(-1) // last cell height
	if f.GetY() <= yBefore {
		t.Fatal("Ln(-1) should advance Y")
	}
	f.Ln(20)
	if f.GetX() != f.lMargin {
		t.Fatal("Ln should reset X to left margin")
	}
}

func TestHeaderFooterFuncsInvoked(t *testing.T) {
	f := NewCustom(&InitType{UnitStr: "mm"})
	headerCalled := false
	footerCalled := false
	f.SetHeaderFunc(func() { headerCalled = true })
	f.SetFooterFunc(func() { footerCalled = true })
	f.SetAcceptPageBreakFunc(func() bool { return false })
	f.AddPage()
	f.SetFont("Helvetica", "", 12)
	f.Cell(40, 10, "body")
	mustOutput(t, f)
	if !headerCalled {
		t.Error("header func not invoked")
	}
	if !footerCalled {
		t.Error("footer func not invoked")
	}
}
