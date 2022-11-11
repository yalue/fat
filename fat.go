// This package provides tools for reading or recovering data from FAT32
// filesystem images.  It is unlikely to be a useful general-purpose library;
// it was written for some specific data-recovery projects.
package fat

import (
	"encoding/binary"
	"fmt"
	"io"
)

// The standard size of a drive sector, in bytes.
const SectorSize = 512

// Holds a single partition table entry in the MBR
type PartitionTableEntry struct {
	Attributes      byte
	CHSStartAddress [3]byte
	PartitionType   byte
	CHSEndAddress   [3]byte
	LBAStartAddress uint32
	SectorCount     uint32
}

func (n *PartitionTableEntry) String() string {
	var active string
	if (n.Attributes & 0x80) != 0 {
		active = "Active"
	} else {
		active = "Inactive"
	}
	sizeMB := (float32(n.SectorCount) * SectorSize) / (1024.0 * 1024.0)
	return fmt.Sprintf("%s partition starting at sector %d: %f MB", active,
		n.LBAStartAddress, sizeMB)
}

// Used for parsing the master boot record layout.
type MBR struct {
	BootstrapCode [440]byte
	DiskID        [4]byte
	Reserved      [2]byte
	Partitions    [4]PartitionTableEntry
	Signature     [2]byte
}

// Attempts to parse an MBR at offset 0 in the given image. Returns an error if
// it can't be read, or if it has an invalid signature.
func ParseMBR(image io.ReadSeeker) (*MBR, error) {
	var toReturn MBR
	_, e := image.Seek(0, io.SeekStart)
	if e != nil {
		return nil, fmt.Errorf("Failed seeking to start of image: %w", e)
	}
	e = binary.Read(image, binary.LittleEndian, &toReturn)
	if e != nil {
		return nil, fmt.Errorf("Failed parsing MBR: %w", e)
	}
	if (toReturn.Signature[0] != 0x55) || (toReturn.Signature[1] != 0xaa) {
		return nil, fmt.Errorf("Image missing 0x55, 0xAA signature")
	}
	return &toReturn, nil
}
