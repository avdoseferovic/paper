package fontrepository_test

import (
	"os"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/fontrepository"
	"golang.org/x/image/font/gofont/goregular"
)

func TestRepository_AddUTF8Font(t *testing.T) {
	t.Parallel()
	t.Run("when fontstyle family is empty, should not add value", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := fontrepository.New()

		// Act
		customFonts, err := sut.AddUTF8Font("", fontstyle.Bold, "file").Load()

		// Assert
		assert.Nil(t, err)
		assert.Empty(t, customFonts)
	})
	t.Run("when fontstyle style is invalid, should not add value", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := fontrepository.New()

		// Act
		customFonts, err := sut.AddUTF8Font("family", "invalid", "file").Load()

		// Assert
		assert.Nil(t, err)
		assert.Empty(t, customFonts)
	})
	t.Run("when fontstyle file is empty, should not add value", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := fontrepository.New()

		// Act
		customFonts, err := sut.AddUTF8Font("family", fontstyle.Bold, "").Load()

		// Assert
		assert.Nil(t, err)
		assert.Empty(t, customFonts)
	})
	t.Run("when fontstyle is valid, should add value", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := fontrepository.New()

		// Act
		customFonts, err := sut.AddUTF8Font("family", fontstyle.Bold, writeTestFont(t)).Load()

		// Assert
		assert.Nil(t, err)
		assert.Len(t, customFonts, 1)
		assert.Equal(t, "family", customFonts[0].GetFamily())
		assert.Equal(t, fontstyle.Bold, customFonts[0].GetStyle())
		assert.NotEmpty(t, customFonts[0].GetFile())
		assert.NotEmpty(t, customFonts[0].GetBytes())
	})
}

func TestRepository_AddUTF8FontFromBytes(t *testing.T) {
	t.Parallel()
	t.Run("when fontstyle family is empty, should not add value", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := fontrepository.New()

		// Act
		customFonts, err := sut.AddUTF8FontFromBytes("", fontstyle.Bold, []byte(``)).Load()

		// Assert
		assert.Nil(t, err)
		assert.Empty(t, customFonts)
	})
	t.Run("when fontstyle style is invalid, should not add value", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := fontrepository.New()

		// Act
		customFonts, err := sut.AddUTF8FontFromBytes("family", "invalid", []byte(``)).Load()

		// Assert
		assert.Nil(t, err)
		assert.Empty(t, customFonts)
	})
	t.Run("when fontstyle bytes is nil, should not add value", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := fontrepository.New()

		// Act
		customFonts, err := sut.AddUTF8FontFromBytes("family", fontstyle.Bold, nil).Load()

		// Assert
		assert.Nil(t, err)
		assert.Empty(t, customFonts)
	})
	t.Run("when fontstyle is valid, should add value", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := fontrepository.New()

		// Act
		customFonts, err := sut.AddUTF8FontFromBytes("family", fontstyle.Bold, goregular.TTF).Load()

		// Assert
		assert.Nil(t, err)
		assert.Len(t, customFonts, 1)
		assert.Equal(t, "family", customFonts[0].GetFamily())
		assert.Equal(t, fontstyle.Bold, customFonts[0].GetStyle())
		assert.Empty(t, customFonts[0].GetFile())
		assert.NotEmpty(t, customFonts[0].GetBytes())
	})
}

func writeTestFont(t *testing.T) string {
	t.Helper()

	path := t.TempDir() + "/goregular.ttf"
	require.NoError(t, os.WriteFile(path, goregular.TTF, 0o600))
	return path
}
