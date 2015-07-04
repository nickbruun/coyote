package utils

import (
	"testing"
	"reflect"
)

func TestByteSliceBuffer(t *testing.T) {
	buf := NewByteSliceBuffer(10)

	// Test draining when adding at most the buffer size worth of elements.
	for i := 0; i < 10; i++ {
		expected := make([][]byte, 0, i+1)
		for j := 0; j <= i; j++ {
			s := []byte{byte(j)}
			expected = append(expected, s)
			buf.Add(s)
		}

		actual := buf.Drain()
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Expected drained slices to be %v after adding %d element(s), but got %v", expected, i+1, actual)
		}
	}

	// Test draining when the buffer is overflowing.
	for i := 10; i < 30; i++ {
		for j := 0; j <= i; j++ {
			buf.Add([]byte{byte(j)})
		}

		expected := make([][]byte, 0, i+1)
		for j := i-9; j <= i; j++ {
			expected = append(expected, []byte{byte(j)})
		}

		actual := buf.Drain()
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Expected drained slices to be %v after adding %d element(s), but got %v", expected, i+1, actual)
		}
	}
}
