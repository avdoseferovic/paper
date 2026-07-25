package paper

import (
	"errors"
	"fmt"

	"github.com/avdoseferovic/paper/internal/pdf"
	"github.com/avdoseferovic/paper/pkg/core"
)

// ErrNotAbsorbable reports that a provider pair cannot splice pages, because
// one of them is not backed by the concrete internal PDF writer.
var ErrNotAbsorbable = errors.New("provider: pages cannot be absorbed")

// PageAbsorber is implemented by providers that can take ownership of another
// provider's already-rendered pages. It lets pages be rendered concurrently
// into separate documents and then joined into one before serialization,
// avoiding a byte-level PDF merge.
type PageAbsorber interface {
	AbsorbPagesFrom(other core.Provider) error
}

// AbsorbPagesFrom appends other's rendered pages and resources to g's document,
// and carries over the render issues other collected.
func (g *provider) AbsorbPagesFrom(other core.Provider) error {
	src, ok := other.(*provider)
	if !ok {
		return fmt.Errorf("%w: source is not a paper provider", ErrNotAbsorbable)
	}

	dstPDF, dstOK := g.fpdf.(*pdf.PDF)
	srcPDF, srcOK := src.fpdf.(*pdf.PDF)
	if !dstOK || !srcOK {
		return fmt.Errorf("%w: not backed by the internal PDF writer", ErrNotAbsorbable)
	}

	if err := dstPDF.AbsorbPages(srcPDF); err != nil {
		return err
	}

	g.issues = append(g.issues, src.issues...)
	return nil
}
