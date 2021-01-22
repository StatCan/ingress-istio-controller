package controller

import (
	"testing"
)

func TestStringInArray(t *testing.T) {
	tests := []struct {
		str      string
		arr      []string
		expected bool
	}{
		{"a", []string{"a", "b", "c"}, true},
		{"d", []string{"a", "b", "c"}, false},
		{"alpha", []string{"alpha", "bravo", "charlie"}, true},
		{"alpha", []string{}, false},
		{"", []string{"one", "two", "three", "four", "five"}, false},
		{"", []string{"one", "two", "", "four", "five"}, true},
	}

	for _, test := range tests {
		result := stringInArray(test.str, test.arr)
		if result != test.expected {
			t.Errorf("got %t for %q in %q, expected %t", result, test.str, test.arr, test.expected)
		}
	}
}

func TestStringArrayEquals(t *testing.T) {
	tests := []struct {
		a        []string
		b        []string
		expected bool
	}{
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}, true},
		{[]string{"a", "b", "c"}, []string{"a", "c", "b"}, false},
		{[]string{}, []string{}, true},
		{[]string{"a"}, []string{}, false},
		{[]string{}, []string{"a"}, false},
	}

	for _, test := range tests {
		result := stringArrayEquals(test.a, test.b)
		if result != test.expected {
			t.Errorf("got %t for %q equals %q, expected %t", result, test.a, test.b, test.expected)
		}
	}
}
