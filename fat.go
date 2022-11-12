// This package provides tools for reading or recovering data from FAT32
// filesystem images.  It is unlikely to be a useful general-purpose library;
// it was written for some specific data-recovery projects. Note that virtually
// none of the structs in this are designed to be thread-safe; they rely on
// seeking within a file image, and the resulting offsets not being perturbed.
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

// Returns an io.ReadSeeker corresponding to the partition at the given index.
func GetPartition(image io.ReadSeeker, mbr *MBR, partitionIndex int) (
	io.ReadSeeker, error) {
	if (partitionIndex < 0) || (partitionIndex >= len(mbr.Partitions)) {
		return nil, fmt.Errorf("Invalid partition index: %d", partitionIndex)
	}
	tableEntry := &(mbr.Partitions[partitionIndex])
	startOffset := int64(tableEntry.LBAStartAddress) * SectorSize
	limit := startOffset + (int64(tableEntry.SectorCount) * SectorSize)
	return LimitReadSeeker(image, startOffset, limit)
}

// The BPB at the start of a partition.
type BIOSParameterBlock struct {
	// For bootable sectors, holds code for jumping over the rest of the BPB.
	JumpInstruction      [3]byte
	OEMID                [8]byte
	BytesPerSector       uint16
	SectorsPerCluster    uint8
	ReservedSectorCount  uint16
	FATCount             uint8
	RootDirEntryCount    uint16
	LogicalVolumeSectors uint16
	MediaDescriptorType  byte
	// Only used for FAT12/FAT16
	SectorsPerFAT   uint16
	SectorsPerTrack uint16
	MediaHeadCount  uint16
	// The number of sectors prior to this one
	HiddenSectorCount uint32
	// The number of sectors in this partition.
	LargeSectorCount uint32
}

// Returns a human-readable multi-line string containing information from this
// BPB.
func (bpb *BIOSParameterBlock) FormatHumanReadable() string {
	toReturn := "BPB information:\n"
	toReturn += fmt.Sprintf("  Bytes per sector: %d\n", bpb.BytesPerSector)
	toReturn += fmt.Sprintf("  Sectors per cluster: %d\n",
		bpb.SectorsPerCluster)
	toReturn += fmt.Sprintf("  Reserved sector count: %d\n",
		bpb.ReservedSectorCount)
	toReturn += fmt.Sprintf("  FAT count: %d\n", bpb.FATCount)
	toReturn += fmt.Sprintf("  Root dir entry count: %d\n",
		bpb.RootDirEntryCount)
	toReturn += fmt.Sprintf("  Logical volume sectors: %d\n",
		bpb.LogicalVolumeSectors)
	toReturn += fmt.Sprintf("  Media descriptor type: 0x%02x\n",
		bpb.MediaDescriptorType)
	toReturn += fmt.Sprintf("  Sectors per track: %d\n", bpb.SectorsPerTrack)
	toReturn += fmt.Sprintf("  Head count: %d\n", bpb.MediaHeadCount)
	toReturn += fmt.Sprintf("  Hidden sectors: %d\n", bpb.HiddenSectorCount)
	toReturn += fmt.Sprintf("  Large sector count: %d", bpb.LargeSectorCount)
	return toReturn
}

// The "extended boot record" for FAT32, should appear immediately after the
// BPB.
type FAT32EBR struct {
	SectorsPerFAT        uint32
	Flags                uint16
	FATVersion           uint16
	RootDirClusterNumber uint32
	FSInfoSector         uint16
	BackupBootSector     uint16
	Reserved             [12]byte
	DriveNumber          uint8
	WindowsNTFlags       uint8
	Signature            byte
	VolumeID             uint32
	// The volume label string, padded with spaces.
	VolumeLabel [11]byte
	// Should always be "FAT32   "
	SystemID [8]byte
	BootCode [420]byte
	// 0xaa55, if a bootable partition.
	BootSignature uint16
}

// Returns a multi-line string formatting the EBR information in a
// human-readable fashion.
func (ebr *FAT32EBR) FormatHumanReadable() string {
	toReturn := "FAT32 EBR information:\n"
	toReturn += fmt.Sprintf("  Sectors per FAT: %d\n", ebr.SectorsPerFAT)
	toReturn += fmt.Sprintf("  Flags: 0x%04x\n", ebr.Flags)
	toReturn += fmt.Sprintf("  FAT version: %d\n", ebr.FATVersion)
	toReturn += fmt.Sprintf("  Root dir cluster #: %d\n",
		ebr.RootDirClusterNumber)
	toReturn += fmt.Sprintf("  FS info sector: %d\n", ebr.FSInfoSector)
	toReturn += fmt.Sprintf("  Backup boot sector: %d\n", ebr.BackupBootSector)
	toReturn += fmt.Sprintf("  Drive number: %d\n", ebr.DriveNumber)
	toReturn += fmt.Sprintf("  Windows NT flags: 0x%02x\n", ebr.WindowsNTFlags)
	toReturn += fmt.Sprintf("  Signature: 0x%02x\n", ebr.Signature)
	toReturn += fmt.Sprintf("  Volume ID 0x%08x\n", ebr.VolumeID)
	toReturn += fmt.Sprintf("  Volume label: \"%s\"\n", ebr.VolumeLabel)
	toReturn += fmt.Sprintf("  System ID: \"%s\"\n", ebr.SystemID)
	toReturn += fmt.Sprintf("  Boot signature: 0x%04x", ebr.BootSignature)
	return toReturn
}

// Used to obtain information about a FAT32 partition in a unified manner.
type FAT32Header struct {
	BPB BIOSParameterBlock
	EBR FAT32EBR
}

// Returns a multi-line string containing information about the FAT32 header in
// a human-readable manner.
func (h *FAT32Header) FormatHumanReadable() string {
	toReturn := (&(h.BPB)).FormatHumanReadable() + "\n"
	toReturn += (&(h.EBR)).FormatHumanReadable()
	return toReturn
}

// Parses a FAT32 header, expected at the beginning of the given disk image.
func ParseFAT32Header(image io.ReadSeeker) (*FAT32Header, error) {
	var toReturn FAT32Header
	_, e := image.Seek(0, io.SeekStart)
	if e != nil {
		return nil, fmt.Errorf("Error seeking start of FAT32 image: %w", e)
	}
	e = binary.Read(image, binary.LittleEndian, &toReturn)
	if e != nil {
		return nil, fmt.Errorf("Error parsing FAT32 header: %w", e)
	}
	if toReturn.BPB.BytesPerSector != SectorSize {
		return nil, fmt.Errorf("Unsupported bytes per sector: %d, need %d",
			toReturn.BPB.BytesPerSector, SectorSize)
	}
	// NOTE: It may be good to do more signature checking here?
	return &toReturn, nil
}

// The FSInfo structure used by FAT32 to do things like speed up free space
// computations and find empty sectors.
type FSInfo struct {
	Signature1                uint32
	Reserved1                 [480]byte
	Signature2                uint32
	LastKnownFreeCluster      uint32
	FirstAvailableClusterHint uint32
	Reserved2                 [12]byte
	Signature3                uint32
}

// Checks that the signatures in the FSInfo struct match, and returns a non-nil
// error if one of them is wrong.
func (n *FSInfo) Validate() error {
	expected := uint32(0x41615252)
	if n.Signature1 != expected {
		return fmt.Errorf("Bad first signature. Expected 0x%08x, got 0x%08x",
			expected, n.Signature1)
	}
	expected = 0x61417272
	if n.Signature2 != expected {
		return fmt.Errorf("Bad second signature. Expected 0x%08x, got 0x%08x",
			expected, n.Signature2)
	}
	expected = 0xaa550000
	if n.Signature3 != expected {
		return fmt.Errorf("Bad third signature. Expected 0x%08x, got 0x%08x",
			expected, n.Signature3)
	}
	return nil
}

// Returns a multi-line string containing information from the FSInfo
// structure.
func (n *FSInfo) FormatHumanReadable() string {
	toReturn := "FAT32 FSInfo structure:\n"
	toReturn += fmt.Sprintf("  Last known free cluster: 0x%08x (%d)\n",
		n.LastKnownFreeCluster, n.LastKnownFreeCluster)
	toReturn += fmt.Sprintf("  First available cluster hint: 0x%08x (%d)",
		n.FirstAvailableClusterHint, n.FirstAvailableClusterHint)
	return toReturn
}

// Wraps all of the stuff we need to track regarding the FAT32 FS.
type FAT32Filesystem struct {
	// The actual content of the full image, starting with the BPB. Must
	// outlive the FAT32Filesystem object.
	Content io.ReadSeeker
	// The parsed BPB and EBR
	Header *FAT32Header
	// The parsed FSInfo block
	Info *FSInfo
}

// Prints a human-readable string of all metadata associated with this FAT32
// filesystem.
func (s *FAT32Filesystem) FormatHumanReadable() string {
	toReturn := s.Header.FormatHumanReadable() + "\n" +
		s.Info.FormatHumanReadable()
	return toReturn
}

// To be called after setting s.Content and s.Header. Finds and parses the
// FSInfo block, populating s.Info.
func (s *FAT32Filesystem) parseFSInfo() error {
	byteOffset := int64(s.Header.EBR.FSInfoSector) * SectorSize
	_, e := s.Content.Seek(byteOffset, io.SeekStart)
	if e != nil {
		return fmt.Errorf("Failed seeking to FSInfo offset: %w", e)
	}
	var info FSInfo
	e = binary.Read(s.Content, binary.LittleEndian, &info)
	if e != nil {
		return fmt.Errorf("Failed reading FSInfo struct: %w", e)
	}
	e = (&info).Validate()
	if e != nil {
		return fmt.Errorf("Invalid FSInfo struct: %w", e)
	}
	s.Info = &info
	return nil
}

// Loads our FAT32Filesystem struct, parsing header contents as necessary.
// The content ReadSeeker must outlive the usage of the returned
// FAT32Filesystem object.  For example, if it's backed by a file, the file
// should not be closed until the FAT32Filesystem isn't needed anymore.
func NewFAT32Filesystem(content io.ReadSeeker) (*FAT32Filesystem, error) {
	header, e := ParseFAT32Header(content)
	if e != nil {
		return nil, fmt.Errorf("Error reading FAT32 header: %w", e)
	}
	toReturn := &FAT32Filesystem{
		Content: content,
		Header:  header,
		Info:    nil,
	}
	e = toReturn.parseFSInfo()
	if e != nil {
		return nil, fmt.Errorf("Error reading FSInfo block: %w", e)
	}
	return toReturn, nil
}
