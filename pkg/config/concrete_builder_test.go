package config_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type customFontFixture struct {
	family string
}

func (c customFontFixture) GetFamily() string {
	return c.family
}

func (c customFontFixture) GetStyle() fontstyle.Type {
	return fontstyle.Normal
}

func (c customFontFixture) GetFile() string {
	return ""
}

func (c customFontFixture) GetBytes() []byte {
	return nil
}

func TestNewCfgBuilderReturnsConcreteBuilderWithoutChangingNewBuilder(t *testing.T) {
	t.Parallel()

	concrete := config.NewCfgBuilder()
	assert.IsType(t, &config.CfgBuilder{}, concrete)

	var asInterface config.Builder = concrete
	assert.NotNil(t, asInterface)

	fromExistingConstructor := config.NewBuilder()
	assert.Implements(t, (*config.Builder)(nil), fromExistingConstructor)
}

func TestBuildCopiesCallerOwnedFontColorAndBackgroundBytes(t *testing.T) {
	t.Parallel()

	color := &props.Color{Red: 1, Green: 2, Blue: 3}
	font := &props.Font{Family: consts.FontFamilyHelvetica, Size: 9, Color: color}
	background := []byte{1, 2, 3}
	firstPageBackground := []byte{4, 5, 6}
	firstPageForeground := []byte{7, 8, 9}

	cfg := config.NewCfgBuilder().
		WithDefaultFont(font).
		WithBackgroundImage(background, extension.Png).
		WithFirstPageBackgroundImage(firstPageBackground, extension.Png).
		WithFirstPageForegroundImage(firstPageForeground, extension.Png).
		Build()

	font.Family = consts.FontFamilyCourier
	color.Red = 99
	background[0] = 9
	firstPageBackground[0] = 9
	firstPageForeground[0] = 9

	assert.Equal(t, consts.FontFamilyHelvetica, cfg.DefaultFont.Family)
	assert.Equal(t, 1, cfg.DefaultFont.Color.Red)
	assert.Equal(t, []byte{1, 2, 3}, cfg.BackgroundImage.Bytes)
	assert.Equal(t, []byte{4, 5, 6}, cfg.FirstPageBackgroundImage.Bytes)
	assert.Equal(t, []byte{7, 8, 9}, cfg.FirstPageForegroundImage.Bytes)
}

func TestNewCfgBuilderDefaultFontDoesNotShareMutableColorGlobal(t *testing.T) {
	original := props.BlackColor
	defer func() {
		props.BlackColor = original
	}()

	builder := config.NewCfgBuilder()
	props.BlackColor.Red = 99

	cfg := builder.Build()

	assert.Equal(t, 0, cfg.DefaultFont.Color.Red)
}

func TestBuilderCopiesCallerOwnedInputsBeforeBuild(t *testing.T) {
	t.Parallel()

	color := &props.Color{Red: 1, Green: 2, Blue: 3}
	font := &props.Font{Family: consts.FontFamilyHelvetica, Size: 9, Color: color}
	background := []byte{1, 2, 3}
	firstPageBackground := []byte{4, 5, 6}
	firstPageForeground := []byte{7, 8, 9}
	customFonts := []entity.CustomFont{customFontFixture{family: "original"}}

	builder := config.NewCfgBuilder().
		WithDefaultFont(font).
		WithBackgroundImage(background, extension.Png).
		WithFirstPageBackgroundImage(firstPageBackground, extension.Png).
		WithFirstPageForegroundImage(firstPageForeground, extension.Png).
		WithCustomFonts(customFonts)

	font.Family = consts.FontFamilyCourier
	color.Red = 99
	background[0] = 9
	firstPageBackground[0] = 9
	firstPageForeground[0] = 9
	customFonts[0] = customFontFixture{family: "changed"}

	cfg := builder.Build()

	assert.Equal(t, consts.FontFamilyHelvetica, cfg.DefaultFont.Family)
	assert.Equal(t, 1, cfg.DefaultFont.Color.Red)
	assert.Equal(t, []byte{1, 2, 3}, cfg.BackgroundImage.Bytes)
	assert.Equal(t, []byte{4, 5, 6}, cfg.FirstPageBackgroundImage.Bytes)
	assert.Equal(t, []byte{7, 8, 9}, cfg.FirstPageForegroundImage.Bytes)
	assert.Equal(t, "original", cfg.CustomFonts[0].GetFamily())
}
