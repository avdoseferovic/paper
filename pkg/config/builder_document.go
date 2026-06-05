package config

import (
	"github.com/avdoseferovic/paper/v2/pkg/consts/extension"
	"github.com/avdoseferovic/paper/v2/pkg/consts/orientation"
	"github.com/avdoseferovic/paper/v2/pkg/consts/protection"
	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
)

// WithProtection defines protection types to the PDF document.
func (b *CfgBuilder) WithProtection(protectionType protection.Type, userPassword, ownerPassword string) Builder {
	b.protection = &entity.Protection{
		Type:          protectionType,
		UserPassword:  userPassword,
		OwnerPassword: ownerPassword,
	}

	return b
}

// WithCompression defines compression.
func (b *CfgBuilder) WithCompression(compression bool) Builder {
	b.compression = compression
	return b
}

// WithOrientation defines the page orientation. The default orientation is vertical,
// if horizontal is defined width and height will be flipped.
func (b *CfgBuilder) WithOrientation(pageOrientation orientation.Type) Builder {
	b.orientation = pageOrientation
	return b
}

// WithCustomFonts add custom fonts.
func (b *CfgBuilder) WithCustomFonts(customFonts []entity.CustomFont) Builder {
	b.customFonts = append([]entity.CustomFont(nil), customFonts...)
	return b
}

// WithBackgroundImage defines the background image that will be applied in every page.
func (b *CfgBuilder) WithBackgroundImage(bytes []byte, ext extension.Type) Builder {
	b.backgroundImage = &entity.Image{
		Bytes:     append([]byte(nil), bytes...),
		Extension: ext,
	}

	return b
}

// WithDisableAutoPageBreak defines the option to disable automatic page breaks.
func (b *CfgBuilder) WithDisableAutoPageBreak(disabled bool) Builder {
	b.disableAutoPageBreak = disabled
	return b
}
