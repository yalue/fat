// This defines a command-line utility for attempting to extract information
// from FAT32 filesystems, potentially contained within a disk image with
// several partitions.
package main

import (
	"flag"
	"fmt"
	"github.com/yalue/fat"
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
	return 0
}

func main() {
	os.Exit(run())
}
