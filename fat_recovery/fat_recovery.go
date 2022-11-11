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
	flag.StringVar(&imagePath, "image", "", "The path to the disk image.")
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
		fmt.Printf("  Partition %d: %s\n", i+1, partitionEntry)
	}
	return 0
}

func main() {
	os.Exit(run())
}
