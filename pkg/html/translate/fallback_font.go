package translate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

const (
	fallbackFontFamily   = "__paper_html_fallback"
	maxFallbackFontBytes = 50 << 20
)

var (
	errFallbackFontPathEmpty    = errors.New("html: fallback font path is empty")
	errRemoteFallbackFontRefuse = errors.New("html: remote fallback font refused; configure WithURLPolicy or WithHTTPClient")
	errFallbackFontTooLarge     = errors.New("html: fallback font exceeds configured limit")
)

func (tr *translator) loadFallbackFontIfNeeded(ctx context.Context, body *dom.Node, resolver StylesheetResolver) {
	if tr == nil || strings.TrimSpace(tr.fallbackFontPath) == "" || body == nil {
		return
	}
	if !nodeContainsNonWinAnsiText(body) {
		return
	}
	data, err := tr.loadFallbackFont(ctx, resolver, tr.fallbackFontPath)
	if err != nil {
		tr.unsupported("fallback-font.skipped", tr.fallbackFontPath)
		tr.reportAssetError("FallbackFontPath", tr.fallbackFontPath, err)
		return
	}
	for _, style := range fallbackFontStyles() {
		tr.loadedFonts = append(tr.loadedFonts, loadedFont{
			family: fallbackFontFamily,
			style:  style,
			bytes:  data,
		})
	}
	tr.fallbackFontReady = true
}

func (tr *translator) loadFallbackFont(ctx context.Context, resolver StylesheetResolver, ref string) ([]byte, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, errFallbackFontPathEmpty
	}
	if strings.HasPrefix(ref, "data:") {
		data, _, err := decodeDataURIWithLimits(ref, tr.limits)
		if err != nil {
			return nil, fmt.Errorf("html: loading fallback font %q: %w", ref, err)
		}
		return data, nil
	}
	if isHTTPURL(ref) {
		if tr.remoteAssets {
			return tr.fetchRemoteURL(ctx, ref, maxFallbackFontBytes)
		}
		return nil, errRemoteFallbackFontRefuse
	}

	var resolverErr error
	if !filepath.IsAbs(ref) && resolver != nil {
		data, err := safeLoadStylesheetErr(resolver, ref)
		if err == nil {
			return data, nil
		}
		resolverErr = err
	}
	data, err := readProgrammaticFallbackFont(ref)
	if err != nil {
		if resolverErr != nil {
			return nil, fmt.Errorf("%w; OS fallback: %w", resolverErr, err)
		}
		return nil, err
	}
	return data, nil
}

func readProgrammaticFallbackFont(ref string) ([]byte, error) {
	f, err := os.Open(ref)
	if err != nil {
		return nil, fmt.Errorf("html: opening fallback font %q: %w", ref, err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxFallbackFontBytes+1))
	if err != nil {
		return nil, fmt.Errorf("html: reading fallback font %q: %w", ref, err)
	}
	if int64(len(data)) > maxFallbackFontBytes {
		return nil, fmt.Errorf("%w: %q exceeds %d bytes", errFallbackFontTooLarge, ref, maxFallbackFontBytes)
	}
	return data, nil
}

func fallbackFontStyles() []fontstyle.Type {
	return []fontstyle.Type{
		fontstyle.Normal,
		fontstyle.Bold,
		fontstyle.Italic,
		fontstyle.BoldItalic,
		fontstyle.Semibold,
		fontstyle.SemiboldItalic,
	}
}

func nodeContainsNonWinAnsiText(n *dom.Node) bool {
	if n == nil {
		return false
	}
	switch n.Tag() {
	case "script", "style":
		return false
	case "":
		return textContainsNonWinAnsi(n.TextContent())
	}
	return slices.ContainsFunc(n.Children(), nodeContainsNonWinAnsiText)
}

func textContainsNonWinAnsi(text string) bool {
	return strings.ContainsFunc(text, func(r rune) bool {
		return !canEncodeWinAnsiRune(r)
	})
}

func canEncodeWinAnsiRune(r rune) bool {
	if r >= 0 && r <= 0x7F || r >= 0xA0 && r <= 0xFF {
		return true
	}
	switch r {
	case 0x0152, // Œ
		0x0153, // œ
		0x0160, // Š
		0x0161, // š
		0x0178, // Ÿ
		0x017D, // Ž
		0x017E, // ž
		0x0192, // ƒ
		0x02C6, // ˆ
		0x02DC, // ˜
		0x2013, // –
		0x2014, // —
		0x2018, // ‘
		0x2019, // ’
		0x201A, // ‚
		0x201C, // “
		0x201D, // ”
		0x201E, // „
		0x2020, // †
		0x2021, // ‡
		0x2022, // •
		0x2026, // …
		0x2030, // ‰
		0x2039, // ‹
		0x203A, // ›
		0x20AC, // €
		0x2122: // ™
		return true
	default:
		return false
	}
}

func fallbackRunsFromRun(run props.RichRun, fallbackReady bool) []props.RichRun {
	if !fallbackReady || run.Text == "" || run.Image != nil || run.ForceBreak {
		return []props.RichRun{run}
	}
	if !fallbackCandidateFamily(run.Family) || !textContainsNonWinAnsi(run.Text) {
		return []props.RichRun{run}
	}

	out := make([]props.RichRun, 0, 3)
	var b strings.Builder
	currentFallback := false
	haveSegment := false
	flush := func() {
		if !haveSegment {
			return
		}
		segment := run
		segment.Text = b.String()
		if currentFallback {
			segment.Family = fallbackFontFamily
		}
		out = append(out, segment)
		b.Reset()
		haveSegment = false
	}
	for _, r := range run.Text {
		useFallback := !canEncodeWinAnsiRune(r)
		if haveSegment && useFallback != currentFallback {
			flush()
		}
		currentFallback = useFallback
		haveSegment = true
		b.WriteRune(r)
	}
	flush()
	normalizeSplitRunMargins(out)
	return out
}

func normalizeSplitRunMargins(runs []props.RichRun) {
	if len(runs) <= 1 {
		return
	}
	last := len(runs) - 1
	for i := range runs {
		if i > 0 {
			runs[i].InlineMarginLeft = 0
		}
		if i < last {
			runs[i].InlineMarginRight = 0
		}
	}
}

func fallbackCandidateFamily(family string) bool {
	switch strings.ToLower(strings.TrimSpace(family)) {
	case "",
		consts.FontFamilyArial,
		consts.FontFamilyHelvetica,
		consts.FontFamilyCourier,
		consts.FontFamilySymbol,
		consts.FontFamilyZapBats,
		"sans",
		"sans-serif",
		"serif",
		"monospace",
		"courier new",
		"times",
		"times new roman":
		return true
	default:
		return false
	}
}
