package parser

import (
	"reflect"
	"testing"
)

func TestSet_Add(t *testing.T) {
	s := NewSet[int]()
	s.Add(1)
	s.Add(2)
	s.Add(1) // 重复添加

	if !s.Contains(1) {
		t.Error("Set 应该包含 1")
	}
	if !s.Contains(2) {
		t.Error("Set 应该包含 2")
	}
	if len(s) != 2 {
		t.Errorf("Set 长度应该是 2，实际是 %d", len(s))
	}
}

func TestSet_Contains(t *testing.T) {
	s := NewSet(1, 2, 3)

	if !s.Contains(1) {
		t.Error("Set 应该包含 1")
	}
	if s.Contains(4) {
		t.Error("Set 不应该包含 4")
	}
}

func TestSet_ToSlice(t *testing.T) {
	s := NewSet(1, 2, 3)
	slice := s.ToSlice()

	if len(slice) != 3 {
		t.Errorf("切片长度应该是 3，实际是 %d", len(slice))
	}

	// 检查所有元素都在切片中
	for _, v := range []int{1, 2, 3} {
		found := false
		for _, sv := range slice {
			if sv == v {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("切片应该包含 %d", v)
		}
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]int{
		"c": 3,
		"a": 1,
		"b": 2,
	}

	keys := SortedKeys(m)
	want := []string{"a", "b", "c"}

	if !reflect.DeepEqual(keys, want) {
		t.Errorf("SortedKeys() = %v, want %v", keys, want)
	}
}

func TestFilter(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}
	result := Filter(slice, func(n int) bool {
		return n%2 == 0 // 只保留偶数
	})

	want := []int{2, 4}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("Filter() = %v, want %v", result, want)
	}
}

func TestMap(t *testing.T) {
	slice := []int{1, 2, 3}
	result := Map(slice, func(n int) string {
		return string(rune('a' + n - 1))
	})

	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("Map() = %v, want %v", result, want)
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name  string
		input []int
		want  []int
	}{
		{
			name:  "有重复元素",
			input: []int{1, 2, 2, 3, 1, 4},
			want:  []int{1, 2, 3, 4},
		},
		{
			name:  "无重复元素",
			input: []int{1, 2, 3},
			want:  []int{1, 2, 3},
		},
		{
			name:  "空切片",
			input: []int{},
			want:  []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Unique(tt.input)
			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("Unique() = %v, want %v", result, tt.want)
			}
		})
	}
}
