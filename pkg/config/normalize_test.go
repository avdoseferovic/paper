package config_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// fakeCustomFont is a minimal entity.CustomFont stub for slice-clone tests.
type fakeCustomFont struct {
	family string
}

func (f fakeCustomFont) GetFamily() string        { return f.family }
func (f fakeCustomFont) GetStyle() fontstyle.Type { return fontstyle.Normal }
func (f fakeCustomFont) GetFile() string          { return "" }
func (f fakeCustomFont) GetBytes() []byte         { return nil }

func TestNormalizeConfig_NilReturnsBuilderDefaults(t *testing.T) {
	t.Parallel()

	// Act
	normalized := config.NormalizeConfig(nil)

	// Assert
	require.NotNil(t, normalized)
	assert.Equal(t, config.NewBuilder().Build(), normalized,
		"nil input must produce the builder default config")
	require.NotNil(t, normalized.Dimensions)
	require.NotNil(t, normalized.Margins)
	require.NotNil(t, normalized.DefaultFont)
	assert.NotEmpty(t, normalized.ProviderType)
	assert.NotEmpty(t, normalized.GenerationMode)
	assert.GreaterOrEqual(t, normalized.ChunkWorkers, 1)
	assert.Greater(t, normalized.MaxGridSize, 0)
}

func TestNormalizeConfig_FillsMissingRequiredFieldsWithDefaults(t *testing.T) {
	t.Parallel()

	// Arrange: a config with invalid/zero required fields.
	defaults := config.NewBuilder().Build()
	input := &entity.Config{ChunkWorkers: 0, MaxGridSize: -5}

	// Act
	normalized := config.NormalizeConfig(input)

	// Assert: the invalid required fields fall back to builder defaults.
	assert.Equal(t, defaults.ProviderType, normalized.ProviderType)
	assert.Equal(t, defaults.GenerationMode, normalized.GenerationMode)
	assert.Equal(t, defaults.ChunkWorkers, normalized.ChunkWorkers)
	assert.Equal(t, defaults.MaxGridSize, normalized.MaxGridSize)
	require.NotNil(t, normalized.Dimensions)
	assert.Equal(t, defaults.Dimensions, normalized.Dimensions)
	require.NotNil(t, normalized.Margins)
	assert.Equal(t, defaults.Margins, normalized.Margins)
	require.NotNil(t, normalized.DefaultFont)
}

func TestNormalizeConfig_PreservesValidCallerValues(t *testing.T) {
	t.Parallel()

	// Arrange
	input := &entity.Config{
		Dimensions:   &entity.Dimensions{Width: 100, Height: 200},
		Margins:      &entity.Margins{Left: 1, Right: 2, Top: 3, Bottom: 4},
		ChunkWorkers: 7,
		MaxGridSize:  16,
	}

	// Act
	normalized := config.NormalizeConfig(input)

	// Assert
	assert.Equal(t, 100.0, normalized.Dimensions.Width)
	assert.Equal(t, 200.0, normalized.Dimensions.Height)
	assert.Equal(t, &entity.Margins{Left: 1, Right: 2, Top: 3, Bottom: 4}, normalized.Margins)
	assert.Equal(t, 7, normalized.ChunkWorkers)
	assert.Equal(t, 16, normalized.MaxGridSize)
}

func TestNormalizeConfig_ReturnsIndependentCloneOfInput(t *testing.T) {
	t.Parallel()

	// Arrange
	input := &entity.Config{
		Dimensions:  &entity.Dimensions{Width: 100, Height: 200},
		Margins:     &entity.Margins{Left: 1, Right: 2, Top: 3, Bottom: 4},
		DefaultFont: &props.Font{Size: 12, Family: "Arial", Color: &props.Color{Red: 1, Green: 2, Blue: 3}},
		CustomFonts: []entity.CustomFont{fakeCustomFont{family: "X"}},
	}
	normalized := config.NormalizeConfig(input)

	// Act: mutate the caller-owned input after normalization.
	input.Dimensions.Width = 999
	input.Margins.Left = 99
	input.DefaultFont.Size = 99
	input.CustomFonts = append(input.CustomFonts, fakeCustomFont{family: "Y"})

	// Assert: the returned config is unaffected by input mutation.
	assert.Equal(t, 100.0, normalized.Dimensions.Width,
		"normalized Dimensions must be an independent clone")
	assert.Equal(t, 1.0, normalized.Margins.Left,
		"normalized Margins must be an independent clone")
	assert.Equal(t, 12.0, normalized.DefaultFont.Size,
		"normalized DefaultFont must be an independent clone")
	require.Len(t, normalized.CustomFonts, 1,
		"normalized CustomFonts slice must be an independent clone")
	assert.Equal(t, "X", normalized.CustomFonts[0].GetFamily())
}

func TestNormalizeConfig_MutatingOutputDoesNotAffectInput(t *testing.T) {
	t.Parallel()

	// Arrange
	input := &entity.Config{
		Dimensions: &entity.Dimensions{Width: 100, Height: 200},
		Margins:    &entity.Margins{Left: 1, Right: 2, Top: 3, Bottom: 4},
	}
	normalized := config.NormalizeConfig(input)

	// Act: mutate the returned config.
	normalized.Dimensions.Width = 777
	normalized.Margins.Left = 77

	// Assert: the caller-owned input is unaffected.
	assert.Equal(t, 100.0, input.Dimensions.Width)
	assert.Equal(t, 1.0, input.Margins.Left)
}
