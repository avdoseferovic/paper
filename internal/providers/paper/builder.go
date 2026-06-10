package paper

import (
	"github.com/avdoseferovic/paper/internal/cache"
	"github.com/avdoseferovic/paper/internal/code"
	"github.com/avdoseferovic/paper/internal/math"
	pdf "github.com/avdoseferovic/paper/internal/pdf"
	"github.com/avdoseferovic/paper/internal/providers/paper/cellwriter"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// Dependencies is the dependencies provider for gofpdf.
type Dependencies struct {
	PDF        any
	Font       core.Font
	Text       core.Text
	Code       core.Code
	Image      core.Image
	Line       core.Line
	Checkbox   core.Checkbox
	Cache      cache.Cache
	CellWriter cellwriter.CellWriter
	Cfg        *entity.Config
}

// Builder is the dependencies builder for gofpdf.
type Builder interface {
	Build(cfg *entity.Config, cache cache.Cache) *Dependencies
}

type builder struct{}

// NewBuilder create a new Builder
func NewBuilder() Builder {
	return &builder{}
}

// Build create a new Dependencies.
func (b *builder) Build(cfg *entity.Config, cache cache.Cache) *Dependencies {
	fpdf := pdf.NewCustom(&pdf.InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		FontDirStr:     "",
		Size: pdf.SizeType{
			Wd: cfg.Dimensions.Width,
			Ht: cfg.Dimensions.Height,
		},
	})

	for _, font := range cfg.CustomFonts {
		fpdf.AddUTF8FontFromBytes(font.GetFamily(), string(font.GetStyle()), font.GetBytes())
	}

	if cfg.DisableAutoPageBreak {
		fpdf.SetAutoPageBreak(false, 0)
	} else {
		fpdf.SetAutoPageBreak(true, cfg.Margins.Bottom)
	}

	fpdf.SetMargins(cfg.Margins.Left, cfg.Margins.Top, cfg.Margins.Right)
	fpdf.AddPage()

	font := NewFont(fpdf, cfg.DefaultFont.Size, cfg.DefaultFont.Family, cfg.DefaultFont.Style)
	math := math.New()
	code := code.New()
	text := NewText(fpdf, math, font)
	image := NewImage(fpdf, math)
	line := NewLine(fpdf)
	checkbox := NewCheckbox(fpdf, font)
	gradientRenderer := NewGradientRenderer(fpdf)
	cellWriter := cellwriter.NewBuilder().
		Build(fpdf, gradientRenderer)

	return &Dependencies{
		PDF:        fpdf,
		Font:       font,
		Text:       text,
		Code:       code,
		Image:      image,
		Line:       line,
		Checkbox:   checkbox,
		CellWriter: cellWriter,
		Cfg:        cfg,
		Cache:      cache,
	}
}
