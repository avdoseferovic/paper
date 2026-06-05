package paperpdf

import "sort"

type sortType struct {
	length int
	less   func(int, int) bool
	swap   func(int, int)
}

func (s *sortType) Len() int {
	return s.length
}

func (s *sortType) Less(i, j int) bool {
	return s.less(i, j)
}

func (s *sortType) Swap(i, j int) {
	s.swap(i, j)
}

func gensort(length int, less func(int, int) bool, swap func(int, int)) {
	sort.Sort(&sortType{length: length, less: less, swap: swap})
}
