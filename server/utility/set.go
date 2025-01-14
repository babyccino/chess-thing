package utility

import (
	"fmt"
	"iter"
	"maps"
)

type Set[T comparable] struct {
	set map[T]struct{}
}

func NewSet[T comparable]() Set[T] {
	set := make(map[T]struct{})
	return Set[T]{set}
}

func (set *Set[T]) Add(key T) {
	set.set[key] = struct{}{}
}
func (set *Set[T]) Has(key T) bool {
	_, found := set.set[key]
	return found
}
func (set *Set[T]) Remove(key T) {
	delete(set.set, key)
}
func (set *Set[T]) Iter() iter.Seq[T] {
	return maps.Keys(set.set)
}
func (set *Set[T]) Len() int {
	return len(set.set)
}
func (set *Set[T]) String() string {
	return fmt.Sprintf("%+v", set.set)
}
