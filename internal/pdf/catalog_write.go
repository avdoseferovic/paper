package pdf

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// deterministicDocTime replaces time.Now() for zero-valued document dates when
// deterministic output is requested, so repeated builds are byte-identical.
var deterministicDocTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// SetViewerPreferences configures catalog viewer hints (page layout, page
// mode, window flags, and the initial view).
func (f *PDF) SetViewerPreferences(prefs ViewerPreferences) {
	f.viewerPrefs = &prefs
}

// SetPageLabels defines the page label ranges shown by viewers instead of
// physical page numbers. PageIndex is zero-based.
func (f *PDF) SetPageLabels(labels ...PageLabelRange) {
	f.pageLabels = append([]PageLabelRange(nil), labels...)
}

// SetAttachments embeds the given files in the document (/EmbeddedFiles name
// tree plus /AF associated-files array).
func (f *PDF) SetAttachments(attachments ...FileAttachment) {
	clones := make([]FileAttachment, len(attachments))
	for i, attachment := range attachments {
		attachment.Data = append([]byte(nil), attachment.Data...)
		clones[i] = attachment
	}
	f.attachments = clones
}

// SetNamedDestinations defines named destinations resolvable from links and
// the /Dests name tree. Destinations with empty names or out-of-range page
// indexes are skipped at output time.
func (f *PDF) SetNamedDestinations(destinations ...NamedDestination) {
	f.namedDests = append([]NamedDestination(nil), destinations...)
}

// SetPageAnnotations adds page-level annotations (links, text notes, and
// markup annotations) to the generated pages. PageIndex is zero-based and
// rectangles are in PDF points with a bottom-left origin.
func (f *PDF) SetPageAnnotations(annotations ...PageAnnotation) {
	clones := make([]PageAnnotation, len(annotations))
	for i, annotation := range annotations {
		if annotation.DestPage != nil {
			destPage := *annotation.DestPage
			annotation.DestPage = &destPage
		}
		if annotation.Color != nil {
			color := *annotation.Color
			annotation.Color = &color
		}
		annotation.QuadPoints = append([][8]float64(nil), annotation.QuadPoints...)
		clones[i] = annotation
	}
	f.pageAnnotations = clones
}

// SetPageGeometries configures per-page geometry entries (/Rotate and the
// Crop/Bleed/Trim/Art boxes). PageIndex is zero-based; box values are PDF
// points. Rotations that are not multiples of 90 are skipped.
func (f *PDF) SetPageGeometries(geometries ...PageGeometry) {
	clones := make([]PageGeometry, len(geometries))
	for i, geometry := range geometries {
		geometry.CropBox = cloneBoxPtr(geometry.CropBox)
		geometry.BleedBox = cloneBoxPtr(geometry.BleedBox)
		geometry.TrimBox = cloneBoxPtr(geometry.TrimBox)
		geometry.ArtBox = cloneBoxPtr(geometry.ArtBox)
		clones[i] = geometry
	}
	f.pageGeometries = clones
}

func cloneBoxPtr(box *[4]float64) *[4]float64 {
	if box == nil {
		return nil
	}
	clone := *box
	return &clone
}

// SetFileID sets an explicit trailer /ID. It takes precedence over the
// derived deterministic ID and the encryption ID.
func (f *PDF) SetFileID(id []byte) {
	f.fileID = append([]byte(nil), id...)
}

// SetDeterministic makes repeated builds of the same document byte-identical:
// zero-valued creation/modification dates become a fixed timestamp, resource
// dictionaries are sorted, and a content-derived trailer /ID is emitted.
func (f *PDF) SetDeterministic(deterministic bool) {
	f.deterministic = deterministic
	if deterministic {
		f.catalogSort = true
	}
}

// docTime resolves a document timestamp honoring deterministic mode.
func (f *PDF) docTime(tm time.Time) time.Time {
	if tm.IsZero() && f.deterministic {
		return deterministicDocTime
	}
	return timeOrNow(tm)
}

// pageObjectNumber returns the object number of a zero-based page index.
// Pages are written first, two objects per page (dict + content), starting at
// object 3.
func pageObjectNumber(pageIndex int) int {
	return 3 + 2*pageIndex
}

// putPageGeometry writes the /Rotate and page-box entries for one page.
// pageNum is 1-based.
func (f *PDF) putPageGeometry(pageNum int) {
	for _, geometry := range f.pageGeometries {
		if geometry.PageIndex != pageNum-1 {
			continue
		}
		rotate := ((geometry.Rotate % 360) + 360) % 360
		if rotate != 0 && rotate%90 == 0 {
			f.outf("/Rotate %d", rotate)
		}
		f.putGeometryBox("CropBox", geometry.CropBox)
		f.putGeometryBox("BleedBox", geometry.BleedBox)
		f.putGeometryBox("TrimBox", geometry.TrimBox)
		f.putGeometryBox("ArtBox", geometry.ArtBox)
	}
}

func (f *PDF) putGeometryBox(name string, box *[4]float64) {
	if box == nil {
		return
	}
	f.outf("/%s [%.2f %.2f %.2f %.2f]", name, box[0], box[1], box[2], box[3])
}

// customAnnotationsForPage returns the serialized custom annotations for a
// 1-based page number.
func (f *PDF) customAnnotationsForPage(pageNum int) []string {
	var out []string
	for _, annotation := range f.pageAnnotations {
		if annotation.PageIndex != pageNum-1 {
			continue
		}
		out = append(out, f.serializeAnnotation(annotation))
	}
	return out
}

func (f *PDF) serializeAnnotation(annotation PageAnnotation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<</Type /Annot /Subtype /%s /Rect [%.2f %.2f %.2f %.2f]",
		pdfNameEscape(annotation.Subtype),
		annotation.Rect[0], annotation.Rect[1], annotation.Rect[2], annotation.Rect[3])
	switch annotation.Subtype {
	case "Link":
		b.WriteString(" /Border [0 0 0]")
		switch {
		case annotation.URI != "":
			fmt.Fprintf(&b, " /A <</S /URI /URI %s>>", f.textstring(annotation.URI))
		case annotation.DestName != "":
			fmt.Fprintf(&b, " /Dest %s", f.textstring(annotation.DestName))
		case annotation.DestPage != nil:
			fmt.Fprintf(&b, " /Dest [%d 0 R /Fit]", pageObjectNumber(*annotation.DestPage))
		}
	case "Text":
		if annotation.Contents != "" {
			fmt.Fprintf(&b, " /Contents %s", f.textstring(annotation.Contents))
		}
		icon := annotation.Name
		if icon == "" {
			icon = "Note"
		}
		fmt.Fprintf(&b, " /Name /%s", pdfNameEscape(icon))
		if annotation.Open {
			b.WriteString(" /Open true")
		}
	default:
		// Markup annotations (Highlight, Underline, Squiggly, StrikeOut, ...)
		// require /QuadPoints; derive them from the rectangle when absent.
		quads := annotation.QuadPoints
		if len(quads) == 0 {
			quads = [][8]float64{{
				annotation.Rect[0], annotation.Rect[3],
				annotation.Rect[2], annotation.Rect[3],
				annotation.Rect[0], annotation.Rect[1],
				annotation.Rect[2], annotation.Rect[1],
			}}
		}
		b.WriteString(" /QuadPoints [")
		for i, quad := range quads {
			for j, v := range quad {
				if i > 0 || j > 0 {
					b.WriteByte(' ')
				}
				fmt.Fprintf(&b, "%.2f", v)
			}
		}
		b.WriteString("]")
		if annotation.Contents != "" {
			fmt.Fprintf(&b, " /Contents %s", f.textstring(annotation.Contents))
		}
	}
	if annotation.Color != nil {
		fmt.Fprintf(&b, " /C [%.2f %.2f %.2f]", annotation.Color[0], annotation.Color[1], annotation.Color[2])
	}
	b.WriteString(">>")
	return b.String()
}

// putAttachmentObjects writes the embedded-file stream and filespec objects
// and records their references for the catalog name tree.
func (f *PDF) putAttachmentObjects() {
	f.attachmentRefs = f.attachmentRefs[:0]
	for _, attachment := range f.attachments {
		if strings.TrimSpace(attachment.FileName) == "" {
			continue
		}
		mime := attachment.MIMEType
		if mime == "" {
			mime = "application/octet-stream"
		}
		relationship := attachment.AFRelationship
		if relationship == "" {
			relationship = "Unspecified"
		}

		f.newobj()
		streamRef := f.n
		stream := f.encryptedStream(attachment.Data)
		if f.err != nil {
			return
		}
		modDate := f.docTime(attachment.CreationDate)
		f.outf("<< /Type /EmbeddedFile /Subtype /%s /Length %d /Params << /Size %d /ModDate %s >> >>",
			pdfNameEscape(mime), len(stream), len(attachment.Data), f.textstring(pdfDateString(modDate)))
		f.putstream(stream)
		f.out("endobj")

		f.newobj()
		f.out("<< /Type /Filespec")
		f.outf("/F %s", f.textstring(attachment.FileName))
		f.outf("/UF %s", f.textstring(attachment.FileName))
		if attachment.Description != "" {
			f.outf("/Desc %s", f.textstring(attachment.Description))
		}
		f.outf("/AFRelationship /%s", pdfNameEscape(relationship))
		f.outf("/EF << /F %d 0 R /UF %d 0 R >>", streamRef, streamRef)
		f.out(">>")
		f.out("endobj")

		f.attachmentRefs = append(f.attachmentRefs, attachmentFileSpecRef{name: attachment.FileName, ref: f.n})
	}
}

// namedDestinationValue serializes one destination array.
func namedDestinationValue(dest NamedDestination) string {
	pageRef := pageObjectNumber(dest.PageIndex)
	switch dest.FitType {
	case "XYZ":
		zoom := "null"
		if dest.Zoom != 0 {
			zoom = fmt.Sprintf("%.2f", dest.Zoom)
		}
		return fmt.Sprintf("[%d 0 R /XYZ %.2f %.2f %s]", pageRef, dest.Left, dest.Top, zoom)
	case "FitH":
		return fmt.Sprintf("[%d 0 R /FitH %.2f]", pageRef, dest.Top)
	case "FitV":
		return fmt.Sprintf("[%d 0 R /FitV %.2f]", pageRef, dest.Left)
	default:
		return fmt.Sprintf("[%d 0 R /Fit]", pageRef)
	}
}

// validNamedDestinations returns the emittable destinations sorted by name.
func (f *PDF) validNamedDestinations() []NamedDestination {
	valid := make([]NamedDestination, 0, len(f.namedDests))
	for _, dest := range f.namedDests {
		if dest.Name == "" || dest.PageIndex < 0 || dest.PageIndex >= f.page {
			continue
		}
		valid = append(valid, dest)
	}
	sort.Slice(valid, func(i, j int) bool { return valid[i].Name < valid[j].Name })
	return valid
}

// putCatalogNames emits the /Names dictionary (JavaScript, named
// destinations, embedded files) plus the /AF associated-files array.
func (f *PDF) putCatalogNames() {
	destinations := f.validNamedDestinations()
	hasNames := f.javascript != nil || len(destinations) > 0 || len(f.attachmentRefs) > 0
	if !hasNames {
		return
	}
	f.out("/Names <<")
	if f.javascript != nil {
		f.outf("/JavaScript %d 0 R", f.nJs)
	}
	if len(destinations) > 0 {
		var b strings.Builder
		b.WriteString("/Dests << /Names [")
		for i, dest := range destinations {
			if i > 0 {
				b.WriteByte(' ')
			}
			fmt.Fprintf(&b, "%s %s", f.textstring(dest.Name), namedDestinationValue(dest))
		}
		b.WriteString("] >>")
		f.out(b.String())
	}
	if len(f.attachmentRefs) > 0 {
		refs := append([]attachmentFileSpecRef(nil), f.attachmentRefs...)
		sort.Slice(refs, func(i, j int) bool { return refs[i].name < refs[j].name })
		var b strings.Builder
		b.WriteString("/EmbeddedFiles << /Names [")
		for i, ref := range refs {
			if i > 0 {
				b.WriteByte(' ')
			}
			fmt.Fprintf(&b, "%s %d 0 R", f.textstring(ref.name), ref.ref)
		}
		b.WriteString("] >>")
		f.out(b.String())
	}
	f.out(">>")
	if len(f.attachmentRefs) > 0 {
		var b strings.Builder
		b.WriteString("/AF [")
		for i, ref := range f.attachmentRefs {
			if i > 0 {
				b.WriteByte(' ')
			}
			fmt.Fprintf(&b, "%d 0 R", ref.ref)
		}
		b.WriteString("]")
		f.out(b.String())
	}
}

// putPageLabels emits the /PageLabels number tree.
func (f *PDF) putPageLabels() {
	if len(f.pageLabels) == 0 {
		return
	}
	labels := append([]PageLabelRange(nil), f.pageLabels...)
	sort.SliceStable(labels, func(i, j int) bool { return labels[i].PageIndex < labels[j].PageIndex })
	var b strings.Builder
	b.WriteString("/PageLabels << /Nums [")
	for i, label := range labels {
		if label.PageIndex < 0 {
			continue
		}
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%d <<", label.PageIndex)
		if label.Style != "" {
			fmt.Fprintf(&b, " /S /%s", pdfNameEscape(label.Style))
		}
		if label.Prefix != "" {
			fmt.Fprintf(&b, " /P %s", f.textstring(label.Prefix))
		}
		if label.Start > 0 {
			fmt.Fprintf(&b, " /St %d", label.Start)
		}
		b.WriteString(" >>")
	}
	b.WriteString("] >>")
	f.out(b.String())
}

// putViewerPreferences emits /PageLayout, /PageMode, the /ViewerPreferences
// dictionary, and the initial-view /OpenAction.
func (f *PDF) putViewerPreferences() {
	prefs := f.viewerPrefs
	if prefs == nil {
		return
	}
	if prefs.PageLayout != "" {
		f.outf("/PageLayout /%s", pdfNameEscape(prefs.PageLayout))
	}
	if prefs.PageMode != "" {
		f.outf("/PageMode /%s", pdfNameEscape(prefs.PageMode))
	}
	flags := []struct {
		name string
		set  bool
	}{
		{"HideToolbar", prefs.HideToolbar},
		{"HideMenubar", prefs.HideMenubar},
		{"HideWindowUI", prefs.HideWindowUI},
		{"FitWindow", prefs.FitWindow},
		{"CenterWindow", prefs.CenterWindow},
		{"DisplayDocTitle", prefs.DisplayDocTitle},
	}
	hasFlag := false
	for _, flag := range flags {
		if flag.set {
			hasFlag = true
			break
		}
	}
	if hasFlag {
		f.out("/ViewerPreferences <<")
		for _, flag := range flags {
			if flag.set {
				f.outf("/%s true", flag.name)
			}
		}
		f.out(">>")
	}
	f.putOpenAction(prefs)
}

func (f *PDF) putOpenAction(prefs *ViewerPreferences) {
	if prefs.OpenPage == 0 && prefs.OpenZoom == "" {
		return
	}
	if prefs.OpenPage < 0 || prefs.OpenPage >= f.page {
		return
	}
	pageRef := pageObjectNumber(prefs.OpenPage)
	switch prefs.OpenZoom {
	case "":
		f.outf("/OpenAction [%d 0 R /Fit]", pageRef)
	case "Fit":
		f.outf("/OpenAction [%d 0 R /Fit]", pageRef)
	case "FitH":
		f.outf("/OpenAction [%d 0 R /FitH null]", pageRef)
	case "FitV":
		f.outf("/OpenAction [%d 0 R /FitV null]", pageRef)
	case "FitB":
		f.outf("/OpenAction [%d 0 R /FitB]", pageRef)
	default:
		percent := strings.TrimSuffix(strings.TrimSpace(prefs.OpenZoom), "%")
		value, err := strconv.ParseFloat(percent, 64)
		if err != nil || value <= 0 {
			f.outf("/OpenAction [%d 0 R /Fit]", pageRef)
			return
		}
		//nolint:dupword // XYZ destinations take two literal null operands.
		f.outf("/OpenAction [%d 0 R /XYZ null null %.2f]", pageRef, value/100)
	}
}

// trailerFileID returns the /ID value to write: the explicit file ID, a
// content-derived deterministic ID, or nil.
func (f *PDF) trailerFileID() []byte {
	if len(f.fileID) > 0 {
		return f.fileID
	}
	if f.deterministic {
		sum := sha256.Sum256(f.buffer.Bytes())
		return sum[:16]
	}
	return nil
}
