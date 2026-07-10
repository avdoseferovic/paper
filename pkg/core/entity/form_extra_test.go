package entity_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

func TestEntityNewMultilineTextField(t *testing.T) {
	t.Parallel()

	field := entity.NewMultilineTextField("notes", [4]float64{1, 2, 3, 4}, 0)

	assert.Equal(t, entity.FieldText, field.Type)
	assert.True(t, field.Flags&entity.FlagMultiline != 0)
}

func TestEntityNewPasswordField(t *testing.T) {
	t.Parallel()

	field := entity.NewPasswordField("secret", [4]float64{1, 2, 3, 4}, 0)

	assert.Equal(t, entity.FieldText, field.Type)
	assert.True(t, field.Flags&entity.FlagPassword != 0)
}

func TestConfigToMapIncludesAcroFormDetails(t *testing.T) {
	t.Parallel()

	form := entity.NewAcroForm().Add(entity.NewTextField("name", [4]float64{1, 2, 3, 4}, 0))
	cfg := &entity.Config{AcroForm: form}

	m := cfg.ToMap()
	assert.Equal(t, 1, m["config_acroform_fields"])
	assert.Equal(t, "name", m["config_acroform_field_0_name"])
}
