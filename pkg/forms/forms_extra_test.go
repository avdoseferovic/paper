package forms_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/forms"
	"github.com/avdoseferovic/paper/pkg/reader"
)

func TestNewMultilineTextFieldSetsMultilineFlag(t *testing.T) {
	t.Parallel()

	field := forms.NewMultilineTextField("notes", [4]float64{72, 600, 300, 700}, 0)

	assert.Equal(t, entity.FieldText, field.Type)
	assert.True(t, field.Flags&entity.FlagMultiline != 0)
}

func TestNewPasswordFieldSetsPasswordFlag(t *testing.T) {
	t.Parallel()

	field := forms.NewPasswordField("secret", [4]float64{72, 600, 300, 620}, 0)

	assert.Equal(t, entity.FieldText, field.Type)
	assert.True(t, field.Flags&entity.FlagPassword != 0)
}

func TestFormFillerSaveTo(t *testing.T) {
	t.Parallel()

	form := forms.NewAcroForm().
		Add(forms.NewTextField("name", [4]float64{72, 700, 300, 720}, 0).SetValue("Ada"))
	cfg := config.NewBuilder().WithCompression(false).WithAcroForm(form).Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New("SaveTo test")))
	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)

	r, err := reader.Parse(pdf.GetBytes())
	require.NoError(t, err)
	filler := forms.NewFormFiller(r)
	require.NoError(t, filler.SetValue("name", "Grace"))

	path := filepath.Join(t.TempDir(), "filled.pdf")
	require.NoError(t, filler.SaveTo(path))

	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotEmpty(t, saved)
	assert.Contains(t, string(saved), "Grace")
}
