package config

import (
	"github.com/avdoseferovic/paper/internal/htmllimits"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/consts/protection"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// WithProtection defines protection types to the PDF document.
//
// Protection uses the PDF standard security handler. RC4 is the compatibility
// default; use WithProtectionAlgorithm to select AES-128. Protection deters
// casual copying or printing, but is not confidentiality-grade encryption. For
// confidentiality, encrypt the file at rest or in transit.
func (b *CfgBuilder) WithProtection(protectionType protection.Type, userPassword, ownerPassword string) Builder {
	b.protection = &entity.Protection{
		Type:          protectionType,
		UserPassword:  userPassword,
		OwnerPassword: ownerPassword,
		Algorithm:     b.protectionAlgorithm,
	}

	return b
}

// WithProtectionAlgorithm selects the protection encryption algorithm.
//
// The default remains the legacy RC4 security handler for compatibility.
// AES128 selects AESV2, PDF standard security handler revision 4.
func (b *CfgBuilder) WithProtectionAlgorithm(algorithm protection.Encryption) Builder {
	b.protectionAlgorithm = algorithm
	if b.protection != nil {
		b.protection.Algorithm = algorithm
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
func (b *CfgBuilder) WithOrientation(pageOrientation consts.Orientation) Builder {
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

// WithHTMLLimits configures resource limits for AddHTML/FromHTML translation.
func (b *CfgBuilder) WithHTMLLimits(limits entity.HTMLLimits) Builder {
	b.htmlLimits = htmllimits.Normalize(limits)
	return b
}

// WithUnsafeNoHTMLLimits disables resource limits for AddHTML/FromHTML.
func (b *CfgBuilder) WithUnsafeNoHTMLLimits() Builder {
	b.htmlLimits = htmllimits.NoLimits()
	return b
}
