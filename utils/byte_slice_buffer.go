package utils

// Byte slice buffer.
//
// Ring buffer with a maximum size.
type ByteSliceBuffer struct {
	size   int
	slices [][]byte
	offset int
}

// Add a slice to the buffer.
func (b *ByteSliceBuffer) Add(slice []byte) {
	if len(b.slices) < b.size {
		b.slices = append(b.slices, slice)
	} else {
		b.slices[b.offset] = slice
		b.offset++
		if b.offset == b.size {
			b.offset = 0
		}
	}
}

// Test if the buffer is full.
func (b *ByteSliceBuffer) Full() bool {
	return len(b.slices) >= b.size
}

// Test if the buffer is empty.
func (b *ByteSliceBuffer) Empty() bool {
	return len(b.slices) == 0
}

// Drain the buffer.
//
// Returns the contents of the byte slice buffer and resets it to zero.
func (b *ByteSliceBuffer) Drain() [][]byte {
	slices := b.slices
	offset := b.offset
	b.slices = nil
	b.offset = 0

	if offset > 0 {
		return append(slices[offset:], slices[:offset]...)
	} else {
		return slices
	}
}

// New byte slice buffer.
func NewByteSliceBuffer(size int) *ByteSliceBuffer {
	return &ByteSliceBuffer{
		size: size,
	}
}
