// Package pkg is a compatibility anchor for Paper's public v2 packages.
//
// New code should import the concrete packages it needs directly, such as
// github.com/avdoseferovic/paper/v2/pkg/components/text or
// github.com/avdoseferovic/paper/v2/pkg/config. This package intentionally
// does not re-export the full public API because doing so would create a large
// unstable aggregate namespace.
package pkg
