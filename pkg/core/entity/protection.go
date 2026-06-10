package entity

import (
	"github.com/avdoseferovic/paper/pkg/consts/protection"
)

// Protection is the representation of a pdf protection.
type Protection struct {
	Type          protection.Type
	UserPassword  string
	OwnerPassword string
	Algorithm     protection.Encryption
}

// AppendMap adds the Protection fields to the map.
func (p *Protection) AppendMap(m map[string]any) map[string]any {
	if p.Type != 0 {
		m["config_protection_type"] = p.Type
	}

	if p.UserPassword != "" {
		m["config_user_password"] = p.UserPassword
	}

	if p.OwnerPassword != "" {
		m["config_owner_password"] = p.OwnerPassword
	}

	if p.Algorithm != protection.RC4128 {
		m["config_protection_algorithm"] = p.Algorithm
	}

	return m
}
