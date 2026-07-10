package pdf

// SetTaggedPDF enables tagged PDF foundation output.
func (f *PDF) SetTaggedPDF(enabled bool) {
	f.taggedPDF = enabled
	if !enabled {
		f.structTreeRoot = 0
	}
}

func (f *PDF) taggedPageContent(_ int, content []byte) []byte {
	if !f.taggedPDF {
		return content
	}
	const prefix = "/P << /MCID 0 >> BDC\n"
	wrapped := make([]byte, 0, len(prefix)+len(content)+6)
	wrapped = append(wrapped, prefix...)
	wrapped = append(wrapped, content...)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		wrapped = append(wrapped, '\n')
	}
	wrapped = append(wrapped, []byte("EMC\n")...)
	return wrapped
}

func (f *PDF) putTaggedStructure(pageCount int) {
	f.structTreeRoot = 0
	if !f.taggedPDF || pageCount <= 0 {
		return
	}

	rootRef := f.n + 1
	docRef := f.n + 2
	elemStart := f.n + 3
	arrayStart := elemStart + pageCount
	parentTreeRef := arrayStart + pageCount

	f.newobjExpected(rootRef)
	if f.err != nil {
		return
	}
	f.structTreeRoot = rootRef
	f.out("<<")
	f.out("/Type /StructTreeRoot")
	f.outf("/K %d 0 R", docRef)
	f.outf("/ParentTree %d 0 R", parentTreeRef)
	f.out(">>")
	f.out("endobj")

	f.newobjExpected(docRef)
	if f.err != nil {
		return
	}
	f.out("<<")
	f.out("/Type /StructElem")
	f.out("/S /Document")
	f.outf("/P %d 0 R", rootRef)
	f.out("/K [")
	for pageIndex := range pageCount {
		f.outf("%d 0 R", elemStart+pageIndex)
	}
	f.out("]")
	f.out(">>")
	f.out("endobj")

	for pageIndex := range pageCount {
		elemRef := elemStart + pageIndex
		pageRef := formPageObjectNumber(pageIndex)
		f.newobjExpected(elemRef)
		if f.err != nil {
			return
		}
		f.out("<<")
		f.out("/Type /StructElem")
		f.out("/S /P")
		f.outf("/P %d 0 R", docRef)
		f.outf("/Pg %d 0 R", pageRef)
		f.outf("/K << /Type /MCR /Pg %d 0 R /MCID 0 >>", pageRef)
		f.out(">>")
		f.out("endobj")
	}

	for pageIndex := range pageCount {
		arrayRef := arrayStart + pageIndex
		elemRef := elemStart + pageIndex
		f.newobjExpected(arrayRef)
		if f.err != nil {
			return
		}
		f.outf("[%d 0 R]", elemRef)
		f.out("endobj")
	}

	f.newobjExpected(parentTreeRef)
	if f.err != nil {
		return
	}
	f.out("<<")
	f.out("/Nums [")
	for pageIndex := range pageCount {
		f.outf("%d %d 0 R", pageIndex, arrayStart+pageIndex)
	}
	f.out("]")
	f.out(">>")
	f.out("endobj")
}
