package config_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/forms"
)

func TestBuilder_WithAcroFormClonesInput(t *testing.T) {
	t.Parallel()

	field := forms.NewTextField("name", [4]float64{1, 2, 3, 4}, 0).SetValue("Ada")
	form := forms.NewAcroForm().Add(field)

	cfg := config.NewBuilder().WithAcroForm(form).Build()
	field.Name = "mutated"
	field.Value = "Grace"

	require.NotNil(t, cfg.AcroForm)
	require.Len(t, cfg.AcroForm.Fields(), 1)
	assert.Equal(t, "name", cfg.AcroForm.Fields()[0].Name)
	assert.Equal(t, "Ada", cfg.AcroForm.Fields()[0].Value)
}

func TestNormalizeConfig_ClonesAcroForm(t *testing.T) {
	t.Parallel()

	field := forms.NewDropdown("country", [4]float64{1, 2, 3, 4}, 0, []string{"USA", "Canada"})
	input := &entity.Config{AcroForm: forms.NewAcroForm().Add(field)}

	normalized := config.NormalizeConfig(input)
	field.Options[0] = "mutated"

	require.NotNil(t, normalized.AcroForm)
	require.Len(t, normalized.AcroForm.Fields(), 1)
	assert.Equal(t, []string{"USA", "Canada"}, normalized.AcroForm.Fields()[0].Options)
}
