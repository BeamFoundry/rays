package logger

/*
BSD 3-Clause License

Copyright 2022 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
   contributors may be used to endorse or promote products derived from
   this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import (
	"sync"
)

// Buffer is a byte buffer.
//
// This implementation is adapted from the unexported type buffer
// in go/src/fmt/print.go.
type Buffer []byte

// Having an initial size gives a dramatic speedup.
var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return (*Buffer)(&b)
	},
}

// NewBuffer retrieves a Buffer from the pool. Callers should call Free when finished.
func NewBuffer() *Buffer {
	return bufPool.Get().(*Buffer)
}

// Free returns the Buffer to the pool if its capacity does not exceed the maximum buffer size.
func (b *Buffer) Free() {
	// To reduce peak allocation, return only smaller buffers to the pool.
	const maxBufferSize = 16 << 10
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		bufPool.Put(b)
	}
}

// Reset resets the buffer length to zero.
func (b *Buffer) Reset() {
	b.SetLen(0)
}

// Write appends the byte slice p to the buffer.
func (b *Buffer) Write(p []byte) (int, error) {
	*b = append(*b, p...)
	return len(p), nil
}

// WriteString appends the string s to the buffer.
func (b *Buffer) WriteString(s string) (int, error) {
	*b = append(*b, s...)
	return len(s), nil
}

// WriteByte appends a single byte c to the buffer.
func (b *Buffer) WriteByte(c byte) error {
	*b = append(*b, c)
	return nil
}

// String returns the contents of the unread portion of the buffer as a string.
func (b *Buffer) String() string {
	return string(*b)
}

// Len returns the number of bytes in the buffer.
func (b *Buffer) Len() int {
	return len(*b)
}

// SetLen truncates the buffer length to n bytes.
func (b *Buffer) SetLen(n int) {
	*b = (*b)[:n]
}
