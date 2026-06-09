package pdf

import (
	"bytes"
	"crypto/sha1"
	"encoding/gob"
	"errors"
	"fmt"
	"maps"
	sortpkg "sort"
)

// CreateTemplate defines a new template using the current page size.
func (f *PDF) CreateTemplate(fn func(*Tpl)) Template {
	return newTpl(PointType{0, 0}, f.curPageSize, f.defOrientation, f.unitStr, f.fontDirStr, fn, f)
}

// CreateTemplateCustom starts a template, using the given bounds.
func (f *PDF) CreateTemplateCustom(corner PointType, size SizeType, fn func(*Tpl)) Template {
	return newTpl(corner, size, f.defOrientation, f.unitStr, f.fontDirStr, fn, f)
}

// UseTemplate adds a template to the current page or another template,
// using the size and position at which it was originally written.
func (f *PDF) UseTemplate(t Template) {
	if t == nil {
		f.SetErrorf("template is nil")
		return
	}
	corner, size := t.Size()
	f.UseTemplateScaled(t, corner, size)
}

// UseTemplateScaled adds a template to the current page or another template,
// using the given page coordinates.
func (f *PDF) UseTemplateScaled(t Template, corner PointType, size SizeType) {
	if t == nil {
		f.SetErrorf("template is nil")
		return
	}

	if f.page <= 0 {
		f.SetErrorf("cannot use a template without first adding a page")
		return
	}

	f.templates[t.ID()] = t
	for _, tt := range t.Templates() {
		f.templates[tt.ID()] = tt
	}

	existingImages := map[string]bool{}
	for _, image := range f.images {
		existingImages[image.i] = true
	}

	for name, ti := range t.Images() {
		if _, found := existingImages[ti.i]; found {
			continue
		}
		name = sprintf("t%s-%s", t.ID(), name)
		f.images[name] = ti
	}

	_, templateSize := t.Size()
	scaleX := size.Wd / templateSize.Wd
	scaleY := size.Ht / templateSize.Ht
	tx := corner.X * f.k
	ty := (f.curPageSize.Ht - corner.Y - size.Ht) * f.k

	f.outf("q %.4f 0 0 %.4f %.4f %.4f cm", scaleX, scaleY, tx, ty)
	f.outf("/TPL%s Do Q", t.ID())
}

// Template is an object that can be written to, then used and re-used any number of times within a document.
type Template interface {
	ID() string
	Size() (PointType, SizeType)
	Bytes() []byte
	Images() map[string]*ImageInfoType
	Templates() []Template
	NumPages() int
	FromPage(int) (Template, error)
	FromPages() []Template
	Serialize() ([]byte, error)
	gob.GobDecoder
	gob.GobEncoder
}

func (f *PDF) templateFontCatalog() {
	var keyList []string
	var font fontDefType
	var key string
	f.out("/Font <<")
	for key = range f.fonts {
		keyList = append(keyList, key)
	}
	if f.catalogSort {
		sortpkg.Strings(keyList)
	}
	for _, key = range keyList {
		font = f.fonts[key]
		f.outf("/F%s %d 0 R", font.i, font.N)
	}
	f.out(">>")
}

// putTemplates writes the templates to the PDF
func (f *PDF) putTemplates() {
	filter := ""
	if f.compress {
		filter = "/Filter /FlateDecode "
	}

	templates := sortTemplates(f.templates, f.catalogSort)
	var t Template
	for _, t = range templates {
		corner, size := t.Size()

		f.newobj()
		f.templateObjects[t.ID()] = f.n
		f.outf("<<%s/Type /XObject", filter)
		f.out("/Subtype /Form")
		f.out("/Formtype 1")
		f.outf("/BBox [%.2f %.2f %.2f %.2f]", corner.X*f.k, corner.Y*f.k, (corner.X+size.Wd)*f.k, (corner.Y+size.Ht)*f.k)
		if corner.X != 0 || corner.Y != 0 {
			f.outf("/Matrix [1 0 0 1 %.5f %.5f]", -corner.X*f.k*2, corner.Y*f.k*2)
		}

		f.out("/Resources ")
		f.out("<</ProcSet [/PDF /Text /ImageB /ImageC /ImageI]")

		f.templateFontCatalog()

		tImages := t.Images()
		tTemplates := t.Templates()
		if len(tImages) > 0 || len(tTemplates) > 0 {
			f.out("/XObject <<")
			{
				var key string
				var keyList []string
				var ti *ImageInfoType
				for key = range tImages {
					keyList = append(keyList, key)
				}
				if gl.catalogSort {
					sortpkg.Strings(keyList)
				}
				for _, key = range keyList {
					ti = tImages[key]
					f.outf("/I%s %d 0 R", ti.i, ti.n)
				}
			}
			for _, tt := range tTemplates {
				id := tt.ID()
				if objID, ok := f.templateObjects[id]; ok {
					f.outf("/TPL%s %d 0 R", id, objID)
				}
			}
			f.out(">>")
		}

		f.out(">>")

		buffer := t.Bytes()

		if f.compress {
			buffer = sliceCompress(buffer)
		}
		f.outf("/Length %d >>", len(buffer))
		f.putstream(buffer)
		f.out("endobj")
	}
}

func templateKeyList(mp map[string]Template, sort bool) (keyList []string) {
	var key string
	for key = range mp {
		keyList = append(keyList, key)
	}
	if sort {
		sortpkg.Strings(keyList)
	}
	return
}

// sortTemplates puts templates in a suitable order based on dependices
func sortTemplates(templates map[string]Template, catalogSort bool) []Template {
	chain := make([]Template, 0, len(templates)*2)

	// build a full set of dependency chains
	var keyList []string
	var key string
	var t Template
	keyList = templateKeyList(templates, catalogSort)
	for _, key = range keyList {
		t = templates[key]
		tlist := templateChainDependencies(t)
		for _, tt := range tlist {
			if tt != nil {
				chain = append(chain, tt)
			}
		}
	}

	sorted := make([]Template, 0, len(templates))
chain:
	for _, t := range chain {
		for _, already := range sorted {
			if t == already {
				continue chain
			}
		}
		sorted = append(sorted, t)
	}

	return sorted
}

// templateChainDependencies is a recursive function for determining the full chain of template dependencies
func templateChainDependencies(template Template) []Template {
	requires := template.Templates()
	chain := make([]Template, len(requires)*2)
	for _, req := range requires {
		chain = append(chain, templateChainDependencies(req)...)
	}
	chain = append(chain, template)
	return chain
}

// newTpl creates a template, copying graphics settings from a template if one is given
func newTpl(corner PointType, size SizeType, orientationStr, unitStr, fontDirStr string, fn func(*Tpl), copyFrom *PDF) Template {
	sizeStr := ""

	document := pdfNew(orientationStr, unitStr, sizeStr, fontDirStr, size)
	tpl := Tpl{*document}
	if copyFrom != nil {
		tpl.loadParamsFromPDF(copyFrom)
	}
	tpl.AddPage()
	fn(&tpl)

	bytes := make([][]byte, len(tpl.pages))

	for x := 1; x < len(bytes); x++ {
		bytes[x] = tpl.PDF.pages[x].Bytes()
	}

	templates := make([]Template, 0, len(tpl.templates))
	for _, key := range templateKeyList(tpl.templates, true) {
		templates = append(templates, tpl.templates[key])
	}
	images := tpl.images

	template := PDFTpl{corner, size, bytes, images, templates, tpl.page}
	return &template
}

// PDFTpl is a concrete implementation of the Template interface.
type PDFTpl struct {
	corner    PointType
	size      SizeType
	bytes     [][]byte
	images    map[string]*ImageInfoType
	templates []Template
	page      int
}

// ID returns the global template identifier
func (t *PDFTpl) ID() string {
	return fmt.Sprintf("%x", sha1.Sum(t.Bytes()))
}

// Size gives the bounding dimensions of this template
func (t *PDFTpl) Size() (corner PointType, size SizeType) {
	return t.corner, t.size
}

// Bytes returns the actual template data, not including resources
func (t *PDFTpl) Bytes() []byte {
	return t.bytes[t.page]
}

// FromPage creates a new template from a specific Page
func (t *PDFTpl) FromPage(page int) (Template, error) {
	if page == 0 {
		return nil, errors.New("Pages start at 1 No template will have a page 0")
	}

	if page > t.NumPages() {
		return nil, fmt.Errorf("The template does not have a page %d", page)
	}

	if t.page == page {
		return t, nil
	}

	t2 := *t
	t2.page = page
	return &t2, nil
}

// FromPages creates a template slice with all the pages within a template.
func (t *PDFTpl) FromPages() []Template {
	p := make([]Template, t.NumPages())
	for x := 1; x <= t.NumPages(); x++ {
		p[x-1], _ = t.FromPage(x)
	}

	return p
}

// Images returns a list of the images used in this template
func (t *PDFTpl) Images() map[string]*ImageInfoType {
	return t.images
}

// Templates returns a list of templates used in this template
func (t *PDFTpl) Templates() []Template {
	return t.templates
}

// NumPages returns the number of available pages within the template. Look at FromPage and FromPages on access to that content.
func (t *PDFTpl) NumPages() int {
	return len(t.bytes) - 1
}

// Serialize turns a template into a byte string for later deserialization
func (t *PDFTpl) Serialize() ([]byte, error) {
	b := new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	err := enc.Encode(t)

	return b.Bytes(), err
}

// childrenImages returns the next layer of children images, it doesn't dig into
// children of children. Applies template namespace to keys to ensure
// no collisions. See UseTemplateScaled
func (t *PDFTpl) childrenImages() map[string]*ImageInfoType {
	childrenImgs := make(map[string]*ImageInfoType)

	for x := range len(t.templates) {
		imgs := t.templates[x].Images()
		for key, val := range imgs {
			name := sprintf("t%s-%s", t.templates[x].ID(), key)
			childrenImgs[name] = val
		}
	}

	return childrenImgs
}

// childrensTemplates returns the next layer of children templates, it doesn't dig into
// children of children.
func (t *PDFTpl) childrensTemplates() []Template {
	childrenTmpls := make([]Template, 0)

	for x := range len(t.templates) {
		tmpls := t.templates[x].Templates()
		childrenTmpls = append(childrenTmpls, tmpls...)
	}

	return childrenTmpls
}

// GobEncode encodes the receiving template into a byte buffer. Use GobDecode
// to decode the byte buffer back to a template.
func (t *PDFTpl) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)

	childrensTemplates := t.childrensTemplates()
	firstClassTemplates := make([]Template, 0)

found_continue:
	for x := range len(t.templates) {
		for y := range childrensTemplates {
			if childrensTemplates[y].ID() == t.templates[x].ID() {
				continue found_continue
			}
		}

		firstClassTemplates = append(firstClassTemplates, t.templates[x])
	}
	err := encoder.Encode(firstClassTemplates)

	childrenImgs := t.childrenImages()
	firstClassImgs := make(map[string]*ImageInfoType)

	for key, img := range t.images {
		if _, ok := childrenImgs[key]; !ok {
			firstClassImgs[key] = img
		}
	}

	if err == nil {
		err = encoder.Encode(firstClassImgs)
	}
	if err == nil {
		err = encoder.Encode(t.corner)
	}
	if err == nil {
		err = encoder.Encode(t.size)
	}
	if err == nil {
		err = encoder.Encode(t.bytes)
	}
	if err == nil {
		err = encoder.Encode(t.page)
	}

	return w.Bytes(), err
}

// GobDecode decodes the specified byte buffer into the receiving template.
func (t *PDFTpl) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)

	firstClassTemplates := make([]*PDFTpl, 0)
	err := decoder.Decode(&firstClassTemplates)
	t.templates = make([]Template, len(firstClassTemplates))

	for x := 0; x < len(t.templates); x++ {
		t.templates[x] = Template(firstClassTemplates[x])
	}

	firstClassImages := t.childrenImages()

	t.templates = append(t.childrensTemplates(), t.templates...)

	t.images = make(map[string]*ImageInfoType)
	if err == nil {
		err = decoder.Decode(&t.images)
	}

	maps.Copy(t.images, firstClassImages)

	if err == nil {
		err = decoder.Decode(&t.corner)
	}
	if err == nil {
		err = decoder.Decode(&t.size)
	}
	if err == nil {
		err = decoder.Decode(&t.bytes)
	}
	if err == nil {
		err = decoder.Decode(&t.page)
	}

	return err
}

// Tpl is a PDF used for writing a template. It has most of the facilities of
// a PDF, but cannot add more pages. Tpl is used directly only during the
// limited time a template is writable.
type Tpl struct {
	PDF
}

func (t *Tpl) loadParamsFromPDF(f *PDF) {
	t.compress = false

	t.k = f.k
	t.x = f.x
	t.y = f.y
	t.lineWidth = f.lineWidth
	t.capStyle = f.capStyle
	t.joinStyle = f.joinStyle

	t.color.draw = f.color.draw
	t.color.fill = f.color.fill
	t.color.text = f.color.text

	t.fonts = f.fonts
	t.currentFont = f.currentFont
	t.fontFamily = f.fontFamily
	t.fontSize = f.fontSize
	t.fontSizePt = f.fontSizePt
	t.fontStyle = f.fontStyle
	t.ws = f.ws

	maps.Copy(t.images, f.images)
}
