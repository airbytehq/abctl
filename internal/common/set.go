package common

type Set[T comparable] struct {
	vals map[T]struct{}
}

func (s *Set[T]) Add(v T) {
	if s.vals == nil {
		s.vals = map[T]struct{}{}
	}
	s.vals[v] = struct{}{}
}

func (s *Set[T]) Contains(v T) bool {
	if s.vals == nil {
		return false
	}
	_, ok := s.vals[v]
	return ok
}

func (s *Set[T]) Len() int {
	return len(s.vals)
}

func (s *Set[T]) Items() []T {
	out := make([]T, len(s.vals))
	for k := range s.vals {
		out = append(out, k)
	}
	return out
}
