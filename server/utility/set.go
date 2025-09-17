package utility

import (
	"fmt"
	"iter"
	"maps"
	"strings"
)

type Set[T comparable] map[T]struct{}

func NewSet[T comparable]() Set[T] {
	return make(Set[T])
}

func (set Set[T]) Add(key T) {
	set[key] = struct{}{}
}

func (set Set[T]) Has(key T) bool {
	_, found := set[key]
	return found
}

func (set Set[T]) Remove(key T) {
	delete(set, key)
}

func (set Set[T]) DiffArr(other *Set[T]) []T {
	ret := make([]T, 0)
	for el := range set.Keys() {
		if !other.Has(el) {
			ret = append(ret, el)
		}
	}
	return ret
}

func (set Set[T]) Keys() iter.Seq[T] {
	return maps.Keys(set)
}

func (set Set[T]) Len() int {
	return len(set)
}

func (set Set[T]) String() string {
	builder := strings.Builder{}
	builder.WriteString("set{ ")
	for el := range set {
		builder.WriteString(fmt.Sprintf("%v, ", el))
	}
	builder.WriteString(" }")
	return builder.String()
}
