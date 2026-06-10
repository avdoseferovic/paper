// Package config implements custom configuration builder.
package config

import (
	"time"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/extension"

	"github.com/avdoseferovic/paper/pkg/consts/protection"
	"github.com/avdoseferovic/paper/pkg/core/entity"

	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/consts/pagesize"
	"github.com/avdoseferovic/paper/pkg/props"
)

// Builder is the abstraction responsible for global customizations on the document.
type Builder interface {
	dimensionsBuilder
	generationBuilder
	fontBuilder
	documentBuilder
	metadataBuilder
	Build() *entity.Config
}

type dimensionsBuilder interface {
	WithPageSize(size pagesize.Type) Builder
	WithDimensions(width float64, height float64) Builder
	WithLeftMargin(left float64) Builder
	WithTopMargin(top float64) Builder
	WithRightMargin(right float64) Builder
	WithBottomMargin(bottom float64) Builder
}

type generationBuilder interface {
	WithConcurrentMode(chunkWorkers int) Builder
	WithSequentialMode() Builder
	WithSequentialLowMemoryMode(chunkWorkers int) Builder
	WithDebug(on bool) Builder
	WithMaxGridSize(maxGridSize int) Builder
}

type fontBuilder interface {
	WithDefaultFont(font *props.Font) Builder
	WithPageNumber(pageNumber ...props.PageNumber) Builder
}

type documentBuilder interface {
	WithProtection(protectionType protection.Type, userPassword, ownerPassword string) Builder
	WithProtectionAlgorithm(algorithm protection.Encryption) Builder
	WithCompression(compression bool) Builder
	WithOrientation(orientation consts.Orientation) Builder
	WithCustomFonts(customFonts []entity.CustomFont) Builder
	WithBackgroundImage(bytes []byte, extensionType extension.Type) Builder
	WithDisableAutoPageBreak(disabled bool) Builder
	WithHTMLLimits(limits entity.HTMLLimits) Builder
	WithUnsafeNoHTMLLimits() Builder
	WithOutlineFromHeadings(enabled bool) Builder
}

type metadataBuilder interface {
	WithAuthor(author string, isUTF8 bool) Builder
	WithCreator(creator string, isUTF8 bool) Builder
	WithSubject(subject string, isUTF8 bool) Builder
	WithTitle(title string, isUTF8 bool) Builder
	WithCreationDate(time time.Time) Builder
	WithKeywords(keywordsStr string, isUTF8 bool) Builder
}

type CfgBuilder struct {
	providerType         consts.ProviderType
	dimensions           *entity.Dimensions
	margins              *entity.Margins
	chunkWorkers         int
	debug                bool
	maxGridSize          int
	defaultFont          *props.Font
	customFonts          []entity.CustomFont
	pageNumber           *props.PageNumber
	protection           *entity.Protection
	protectionAlgorithm  protection.Encryption
	compression          bool
	pageSize             *pagesize.Type
	orientation          consts.Orientation
	metadata             *entity.Metadata
	backgroundImage      *entity.Image
	disableAutoPageBreak bool
	outlineFromHeadings  bool
	generationMode       consts.GenerationMode
	htmlLimits           entity.HTMLLimits
}

// NewBuilder is responsible to create an instance of Builder.
func NewBuilder() Builder {
	return NewCfgBuilder()
}

// NewCfgBuilder creates a concrete configuration builder.
// NewBuilder keeps returning the public Builder interface, while this function
// exposes the concrete builder for callers that need fluent methods.
func NewCfgBuilder() *CfgBuilder {
	defaultFontColor := props.Black()
	return &CfgBuilder{
		providerType: consts.ProviderPaper,
		margins: &entity.Margins{
			Left:   pagesize.DefaultLeftMargin,
			Right:  pagesize.DefaultRightMargin,
			Top:    pagesize.DefaultTopMargin,
			Bottom: pagesize.DefaultBottomMargin,
		},
		maxGridSize: pagesize.DefaultMaxGridSum,
		defaultFont: &props.Font{
			Size:   pagesize.DefaultFontSize,
			Family: consts.FontFamilyArial,
			Style:  fontstyle.Normal,
			Color:  &defaultFontColor,
		},
		generationMode: consts.GenerationSequential,
		chunkWorkers:   1,
	}
}
