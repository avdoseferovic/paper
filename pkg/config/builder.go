// Package config implements custom configuration builder.
// nolint:interfacebloat // there is no need to reduce this interface
package config

import (
	"time"

	"github.com/avdoseferovic/paper/pkg/consts/generation"

	"github.com/avdoseferovic/paper/pkg/consts/extension"

	"github.com/avdoseferovic/paper/pkg/consts/orientation"
	"github.com/avdoseferovic/paper/pkg/consts/protection"
	"github.com/avdoseferovic/paper/pkg/core/entity"

	"github.com/avdoseferovic/paper/pkg/consts/fontfamily"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/consts/pagesize"
	"github.com/avdoseferovic/paper/pkg/consts/provider"
	"github.com/avdoseferovic/paper/pkg/props"
)

// Builder is the abstraction responsible for global customizations on the document.
type Builder interface {
	WithPageSize(size pagesize.Type) Builder
	WithDimensions(width float64, height float64) Builder
	WithLeftMargin(left float64) Builder
	WithTopMargin(top float64) Builder
	WithRightMargin(right float64) Builder
	WithBottomMargin(bottom float64) Builder
	WithConcurrentMode(chunkWorkers int) Builder
	WithSequentialMode() Builder
	WithSequentialLowMemoryMode(chunkWorkers int) Builder
	WithDebug(on bool) Builder
	WithMaxGridSize(maxGridSize int) Builder
	WithDefaultFont(font *props.Font) Builder
	WithPageNumber(pageNumber ...props.PageNumber) Builder
	WithProtection(protectionType protection.Type, userPassword, ownerPassword string) Builder
	WithCompression(compression bool) Builder
	WithOrientation(orientation orientation.Type) Builder
	WithAuthor(author string, isUTF8 bool) Builder
	WithCreator(creator string, isUTF8 bool) Builder
	WithSubject(subject string, isUTF8 bool) Builder
	WithTitle(title string, isUTF8 bool) Builder
	WithCreationDate(time time.Time) Builder
	WithCustomFonts(customFonts []entity.CustomFont) Builder
	WithBackgroundImage(bytes []byte, extensionType extension.Type) Builder
	WithDisableAutoPageBreak(disabled bool) Builder
	WithKeywords(keywordsStr string, isUTF8 bool) Builder
	Build() *entity.Config
}

type CfgBuilder struct {
	providerType         provider.Type
	dimensions           *entity.Dimensions
	margins              *entity.Margins
	chunkWorkers         int
	debug                bool
	maxGridSize          int
	defaultFont          *props.Font
	customFonts          []entity.CustomFont
	pageNumber           *props.PageNumber
	protection           *entity.Protection
	compression          bool
	pageSize             *pagesize.Type
	orientation          orientation.Type
	metadata             *entity.Metadata
	backgroundImage      *entity.Image
	disableAutoPageBreak bool
	generationMode       generation.Mode
}

// NewBuilder is responsible to create an instance of Builder.
func NewBuilder() Builder {
	return NewCfgBuilder()
}

// NewCfgBuilder creates a concrete configuration builder.
// NewBuilder is kept for v2 compatibility with its existing Builder return type.
func NewCfgBuilder() *CfgBuilder {
	defaultFontColor := props.Black()
	return &CfgBuilder{
		providerType: provider.Paper,
		margins: &entity.Margins{
			Left:   pagesize.DefaultLeftMargin,
			Right:  pagesize.DefaultRightMargin,
			Top:    pagesize.DefaultTopMargin,
			Bottom: pagesize.DefaultBottomMargin,
		},
		maxGridSize: pagesize.DefaultMaxGridSum,
		defaultFont: &props.Font{
			Size:   pagesize.DefaultFontSize,
			Family: fontfamily.Arial,
			Style:  fontstyle.Normal,
			Color:  &defaultFontColor,
		},
		generationMode: generation.Sequential,
		chunkWorkers:   1,
	}
}
