package fat

// This file contains a function for limiting the range and remapping offsets
// when working with an underlying ReadSeeker. We use it to re-map offsets to 0
// when looking at a FAT partition in a larger disk image containing several
// partitions.

import (
	"fmt"
	"io"
)

// Wraps an underlying ReadSeeker, but limits reads to be within a single
// region. WARNING: This will modify the offset of the underlying ReadSeeker
// when used.
type LimitedReadSeeker struct {
	wrapped    io.ReadSeeker
	baseOffset int64
	size       int64
	// The number of bytes past the baseOffset the current seek is to.
	currentOffset int64
}

// Used to optimize wrapping LimitedReadSeeker instances: if it's detected,
// just use the original wrapped object and adjust the offsets.
func nestedReadSeekerOptimization(input *LimitedReadSeeker, baseOffset,
	limit int64) (io.ReadSeeker, error) {
	if (baseOffset + limit) > input.size {
		return nil, fmt.Errorf("Size of nested LimitedReadSeeker exceeds " +
			"the limit of the original instance")
	}
	// We already checked that the limit is larger than the offset.
	size := limit - baseOffset
	newBaseOffset := input.baseOffset + baseOffset
	_, e := input.wrapped.Seek(newBaseOffset, io.SeekStart)
	if e != nil {
		return nil, fmt.Errorf("Failed seeking to base offset in underlying "+
			"(wrapped) io.ReadSeeker: %w", e)
	}
	return &LimitedReadSeeker{
		wrapped:       input.wrapped,
		baseOffset:    newBaseOffset,
		size:          size,
		currentOffset: 0,
	}, nil
}

// Returns a new io.ReadSeeker using the input ReadSeeker, but with offset 0
// corresponding to the given baseOffset, and EOF at the given limit. WARNING:
// using the returned io.ReadSeeker will modify the offset in the original.
func LimitReadSeeker(input io.ReadSeeker, baseOffset,
	limit int64) (io.ReadSeeker, error) {
	if limit <= baseOffset {
		return nil, fmt.Errorf("The base offset must be below the limit")
	}
	tmp, isNested := input.(*LimitedReadSeeker)
	if isNested {
		return nestedReadSeekerOptimization(tmp, baseOffset, limit)
	}
	_, e := input.Seek(baseOffset, io.SeekStart)
	if e != nil {
		return nil, fmt.Errorf("Failed seeking to base offset in underlying "+
			"io.ReadSeeker: %w", e)
	}
	return &LimitedReadSeeker{
		wrapped:       input,
		currentOffset: 0,
		baseOffset:    baseOffset,
		size:          limit - baseOffset,
	}, nil
}

func (s *LimitedReadSeeker) Seek(offset int64, whence int) (int64, error) {
	// We'll just set the internal currentOffset here. We actually do the Seek
	// during the call to Read, in order to allow separate LimitedReadSeekers
	// to work with the same underlying source without messing up each other's
	// offsets (at least so long as calls to Read are not concurrent).
	newOffset := s.currentOffset
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset += offset
	case io.SeekEnd:
		newOffset = s.size + offset
	}
	if newOffset >= s.size {
		s.currentOffset = s.size
		return s.size, io.EOF
	}
	s.currentOffset = newOffset
	return newOffset, nil
}

// Note: this is *not* thread safe when using multiple wrapped ReadSeekers
// with the same underlying source!
func (s *LimitedReadSeeker) Read(dst []byte) (int, error) {
	readSize := len(dst)
	var resultErr error
	readEndOffset := s.currentOffset + int64(readSize)
	if readEndOffset > s.size {
		resultErr = io.EOF
		bytesOver := readEndOffset - s.size
		readSize = readSize - int(bytesOver)
	}
	_, e := s.wrapped.Seek(s.currentOffset+s.baseOffset, io.SeekStart)
	if e != nil {
		return 0, fmt.Errorf("Error seeking in underlying ReadSeeker: %w", e)
	}
	bytesRead, e := s.wrapped.Read(dst[0:readSize])
	s.currentOffset += int64(bytesRead)
	// If we didn't get an error from the read, return our own EOF error if it
	// was detected.
	if e == nil {
		return bytesRead, resultErr
	}
	// Return the underlying error returned by the wrapped read.
	return bytesRead, e
}
