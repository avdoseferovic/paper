package cellwriter

import (
	"github.com/avdoseferovic/paper/internal/providers/paper/gofpdfwrapper"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type stylerTemplate struct {
	next CellWriter
	fpdf gofpdfwrapper.PDF
	name string
}

func (s *stylerTemplate) SetNext(next CellWriter) {
	s.next = next
}

func (s *stylerTemplate) GetName() string {
	return s.name
}

func (s *stylerTemplate) GetNext() CellWriter {
	return s.next
}

func (s *stylerTemplate) GoToNext(width, height float64, config *entity.Config, prop *props.Cell) {
	if s.next == nil {
		return
	}

	s.next.Apply(width, height, config, prop)
}
