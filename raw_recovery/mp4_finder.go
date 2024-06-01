package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// An mp4 file is a sequence of "Boxes" that follow this format.
type Mp4BoxHeader struct {
	Size uint32
	Type [4]byte
}

func isPrintableASCII(v []byte) bool {
	for _, c := range v {
		if (c < ' ') || (c > '~') {
			return false
		}
	}
	return true
}

// Returns a positive value and a non-nil error if src is at the start of an
// mp4 file. The given size is the size of the MP4 file.
func Mp4FileSize(src io.ReadSeeker) (int64, error) {
	currentBox := Mp4BoxHeader{}
	currentFileSize := int64(0)
	firstBox := true
	var e error
	ftypTag := []byte{'f', 't', 'y', 'p'}
	boxType := currentBox.Type[:]

	// Consume mp4 boxes until we're past the last one.
	for {
		e = binary.Read(src, binary.BigEndian, &currentBox)
		if e != nil {
			if (e == io.EOF) || (e == io.ErrUnexpectedEOF) {
				return currentFileSize, nil
			}
			// There should be no reason for this to happen other than EOF.
			return 0, fmt.Errorf("Likely internal error parsing box header: %s", e)
		}
		// Every mp4 box must be at least 8 bytes: a size and a name.
		if currentBox.Size < 8 {
			return currentFileSize, nil
		}
		// The type name must be ASCII.
		if !isPrintableASCII(boxType) {
			return currentFileSize, nil
		}
		// .mp4 files must start with a 'ftyp' box.
		if firstBox {
			if !bytes.Equal(boxType, ftypTag) {
				return 0, nil
			}
			firstBox = false
		} else {
			if bytes.Equal(boxType, ftypTag) {
				// We found another 'ftyp' box, which should indicate a
				// separate video file starting.
				return currentFileSize, nil
			}
			// We're at some random type of box.
		}
		// Accumulate the bytes from this box and seek past it. We've already
		// read 8 bytes of the header.
		currentFileSize += int64(currentBox.Size)
		_, e = src.Seek(int64(currentBox.Size-8), io.SeekCurrent)
		if e != nil {
			return currentFileSize, fmt.Errorf("Error seeking past box: %s", e)
		}
	}
	return 0, fmt.Errorf("Internal error: unreachable")
}

// A utility function used to save a .mp4 file to disk once one has been
// identified. Requires src to be at the start of the .mp4 file (so rewind it
// after Mp4FileSize returns!)
func SaveMp4Data(src io.Reader, size int64, outputDir string, tag int) error {
	outputPath := fmt.Sprintf("%s/video_%d.mp4", outputDir, tag)
	f, e := os.Create(outputPath)
	if e != nil {
		return fmt.Errorf("Error creating %s: %s", outputPath, e)
	}
	defer f.Close()
	_, e = io.CopyN(f, src, size)
	if e != nil {
		return fmt.Errorf("Error writing data to %s: %s", outputPath, e)
	}
	fmt.Printf("Saved video %s OK!\n", outputPath)
	return nil
}

// Attempts to save a .mp4 file if src is at the start of one. Returns
// false, nil if not at the start of a .mp4 file. Returns true if a file was
// found. May modify the offset in src arbitrarily.
func TrySavingMp4(src io.ReadSeeker, outputDir string, tag int) (bool, error) {
	startOffset, e := src.Seek(0, io.SeekCurrent)
	if e != nil {
		return false, fmt.Errorf("Error determining current offset: %s", e)
	}
	size, e := Mp4FileSize(src)
	if e != nil {
		return false, fmt.Errorf("Error checking for mp4 file: %s", e)
	}
	if size == 0 {
		return false, nil
	}
	_, e = src.Seek(startOffset, io.SeekStart)
	if e != nil {
		return true, fmt.Errorf("Error returning to start of mp4 file: %s", e)
	}
	if outputDir == "" {
		fmt.Printf("Found mp4 file %d, not saving.\n", tag)
		return true, nil
	}
	e = SaveMp4Data(src, size, outputDir, tag)
	if e != nil {
		return true, fmt.Errorf("Error saving .mp4 file: %s", e)
	}
	return true, nil
}
