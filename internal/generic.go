package internal

import (
	"cmp"
	"slices"
)

// Set 泛型集合类型，用于去重.
type Set[T comparable] map[T]struct{}

// NewSet 创建一个新的集合.
func NewSet[T comparable](items ...T) Set[T] {
	s := make(Set[T], len(items))
	for _, item := range items {
		s.Add(item)
	}
	return s
}

// Add 向集合添加元素.
func (s Set[T]) Add(item T) {
	s[item] = struct{}{}
}

// Contains 检查集合是否包含元素.
func (s Set[T]) Contains(item T) bool {
	_, ok := s[item]
	return ok
}

// ToSlice 将集合转换为切片.
func (s Set[T]) ToSlice() []T {
	result := make([]T, 0, len(s))
	for item := range s {
		result = append(result, item)
	}
	return result
}

// SortedKeys 返回 map 的排序键列表(泛型版本).
func SortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// Filter 过滤切片元素(泛型版本).
func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map 映射切片元素(泛型版本).
func Map[T, U any](slice []T, mapper func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = mapper(item)
	}
	return result
}

// Unique 去重切片(泛型版本).
func Unique[T comparable](slice []T) []T {
	seen := make(Set[T], len(slice))
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if !seen.Contains(item) {
			seen.Add(item)
			result = append(result, item)
		}
	}
	return result
}
