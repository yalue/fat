// This defines a command-line utility for attempting to extract information
// from FAT32 filesystems, potentially contained within a disk image with
// several partitions.
package main

import (
	"flag"
	"fmt"
	"github.com/yalue/fat"
	"io"
	"os"
)

func run() int {
	var imagePath string
	var partitionIndex int
	flag.StringVar(&imagePath, "image", "", "The path to the disk image.")
	flag.IntVar(&partitionIndex, "partition_index", 0,
		"The index of the partition containing the FAT32 filesystem.")
	flag.Parse()
	if imagePath == "" {
		fmt.Println("Invalid arguments. Run with -help for more information.")
		return 1
	}
	imageFile, e := os.Open(imagePath)
	if e != nil {
		fmt.Printf("Failed opening %s: %s\n", imagePath, e)
		return 1
	}
	defer imageFile.Close()
	mbr, e := fat.ParseMBR(imageFile)
	if e != nil {
		fmt.Printf("Failed parsing MBR in %s: %s\n", imagePath, e)
		return 1
	}
	fmt.Printf("Loaded MBR in %s OK.\n", imagePath)
	for i := range mbr.Partitions[:] {
		partitionEntry := &(mbr.Partitions[i])
		fmt.Printf("  Partition %d: %s\n", i, partitionEntry)
	}
	fmt.Printf("Attempting to load from partition %d.\n", partitionIndex)
	partition, e := fat.GetPartition(imageFile, mbr, partitionIndex)
	if e != nil {
		fmt.Printf("Failed getting partition %d: %s\n", partitionIndex, e)
		return 1
	}
	fatFS, e := fat.NewFAT32Filesystem(partition)
	if e != nil {
		fmt.Printf("Error loading FAT32 filesystem: %s\n", e)
		return 1
	}
	fmt.Printf("Loaded FAT32 FS OK:\n%s\n", fatFS.FormatHumanReadable())
	fmt.Printf("First FAT entries:\n")
	for i := 0; i < 10; i++ {
		fmt.Printf("  %d: 0x%08x\n", i, fatFS.FAT[i])
	}
	chains, e := fatFS.GetAllChains()
	if e != nil {
		fmt.Printf("Error getting chains: %s\n", e)
		return 1
	}
	contiguousCount := 0
	for i := range chains {
		if chains[i].Contiguous {
			contiguousCount++
		}
	}
	fmt.Printf("Found %d chains in the FAT, %d were on contiguous clusters.\n",
		len(chains), contiguousCount)

	// FOR TESTING////////////////////////////////////////////////////////////////////////
	tmpDest := "F:/temp_dump.bin"
	fmt.Printf("Saving chain 392 to %s\n", tmpDest)
	f, e := os.Create(tmpDest)
	if e != nil {
		fmt.Printf("Error opening %s: %s\n", tmpDest, e)
		return 1
	}
	defer f.Close()
	chainReader, e := fatFS.GetChainReader(&(chains[393]))
	if e != nil {
		fmt.Printf("Error getting chain reader: %s\n", e)
		return 1
	}
	copied, e := io.Copy(f, chainReader)
	if e != nil {
		fmt.Printf("Error reading chain content: %s\n", e)
		return 1
	}
	fmt.Printf("Copied %d bytes of data to %s\n", copied, tmpDest)
	///////////////////////////////////////////////////////////////////// END TESTING ////
	return 0
}

func main() {
	os.Exit(run())
}
