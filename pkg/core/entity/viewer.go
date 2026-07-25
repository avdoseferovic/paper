package entity

// PageLayout is how a reader should arrange pages on screen.
type PageLayout string

// The page arrangements a reader can be asked to use.
const (
	LayoutSinglePage     PageLayout = "SinglePage"
	LayoutOneColumn      PageLayout = "OneColumn"
	LayoutTwoColumnLeft  PageLayout = "TwoColumnLeft"
	LayoutTwoColumnRight PageLayout = "TwoColumnRight"
	LayoutTwoPageLeft    PageLayout = "TwoPageLeft"
	LayoutTwoPageRight   PageLayout = "TwoPageRight"
)

// PageMode is what a reader should show alongside the page when the document
// opens, such as the bookmark or thumbnail panel.
type PageMode string

// The side panels (or full-screen mode) a reader can open the document with.
const (
	ModeUseNone        PageMode = "UseNone"
	ModeUseOutlines    PageMode = "UseOutlines"
	ModeUseThumbs      PageMode = "UseThumbs"
	ModeFullScreen     PageMode = "FullScreen"
	ModeUseOC          PageMode = "UseOC"
	ModeUseAttachments PageMode = "UseAttachments"
)

// ViewerPreferences asks the reader how to present the document when it opens.
// Every field is optional; a zero field leaves that choice to the reader.
type ViewerPreferences struct {
	PageLayout      PageLayout
	PageMode        PageMode
	HideToolbar     bool
	HideMenubar     bool
	HideWindowUI    bool
	FitWindow       bool
	CenterWindow    bool
	DisplayDocTitle bool
	OpenPage        int
	OpenZoom        string
}

// AppendMap adds the preferences that are actually set into m and returns it.
// It is used to report the document's configuration.
func (v *ViewerPreferences) AppendMap(m map[string]any) map[string]any {
	if v == nil {
		return m
	}
	if v.PageLayout != "" {
		m["config_viewer_page_layout"] = v.PageLayout
	}
	if v.PageMode != "" {
		m["config_viewer_page_mode"] = v.PageMode
	}
	if v.HideToolbar {
		m["config_viewer_hide_toolbar"] = v.HideToolbar
	}
	if v.HideMenubar {
		m["config_viewer_hide_menubar"] = v.HideMenubar
	}
	if v.HideWindowUI {
		m["config_viewer_hide_window_ui"] = v.HideWindowUI
	}
	if v.FitWindow {
		m["config_viewer_fit_window"] = v.FitWindow
	}
	if v.CenterWindow {
		m["config_viewer_center_window"] = v.CenterWindow
	}
	if v.DisplayDocTitle {
		m["config_viewer_display_doc_title"] = v.DisplayDocTitle
	}
	if v.OpenPage > 0 || v.OpenZoom != "" {
		m["config_viewer_open_page"] = v.OpenPage
	}
	if v.OpenZoom != "" {
		m["config_viewer_open_zoom"] = v.OpenZoom
	}
	return m
}
