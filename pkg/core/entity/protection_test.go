package entity_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/core/entity"

	"github.com/avdoseferovic/paper/internal/assert"

	"github.com/avdoseferovic/paper/pkg/consts/protection"
)

func TestProtection_AppendMap(t *testing.T) {
	t.Parallel()
	// Arrange
	sut := fixtureProtection()
	m := make(map[string]any)

	// Act
	m = sut.AppendMap(m)

	// Assert
	assert.Equal(t, sut.Type, m["config_protection_type"])
	assert.Equal(t, sut.UserPassword, m["config_user_password"])
	assert.Equal(t, sut.OwnerPassword, m["config_owner_password"])
	assert.Equal(t, sut.Algorithm, m["config_protection_algorithm"])
}

func fixtureProtection() entity.Protection {
	return entity.Protection{
		Type:          protection.Print,
		OwnerPassword: "123456",
		UserPassword:  "654321",
		Algorithm:     protection.AES128,
	}
}
