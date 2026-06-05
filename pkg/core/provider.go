// Package core contains all core interfaces and basic implementations.
package core

// Provider is the abstraction of a document creator provider.
type Provider interface {
	GridProvider
	LineProvider
	TextProvider
	CodeProvider
	ImageProvider
	DocumentProvider
	DocumentConfigProvider
}
