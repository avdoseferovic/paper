package entity

type PageLayout string

const (
	LayoutSinglePage     PageLayout = "SinglePage"
	LayoutOneColumn      PageLayout = "OneColumn"
	LayoutTwoColumnLeft  PageLayout = "TwoColumnLeft"
	LayoutTwoColumnRight PageLayout = "TwoColumnRight"
	LayoutTwoPageLeft    PageLayout = "TwoPageLeft"
	LayoutTwoPageRight   PageLayout = "TwoPageRight"
)

type PageMode string

const (
	ModeUseNone        PageMode = "UseNone"
	ModeUseOutlines    PageMode = "UseOutlines"
	ModeUseThumbs      PageMode = "UseThumbs"
	ModeFullScreen     PageMode = "FullScreen"
	ModeUseOC          PageMode = "UseOC"
	ModeUseAttachments PageMode = "UseAttachments"
)

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
