// Package merror holds the default text style used to draw error messages onto a
// generated page.
package merror

import (
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/props"
)

// DefaultErrorText is the default error text properties.
var DefaultErrorText = &props.Text{
	Family: consts.FontFamilyArial,
	Style:  fontstyle.Bold,
	Size:   10,
	Color: &props.Color{
		Red:   255,
		Green: 0,
		Blue:  0,
	},
}
