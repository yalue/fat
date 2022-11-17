// This defines a command-line utility for attempting to extract information
// from FAT32 filesystems, potentially contained within a disk image with
// several partitions.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/yalue/fat"
	"io"
	"os"
	"runtime"
)

func dumpChainContent(f *fat.FAT32Filesystem, outputDir string,
	chains []fat.FATChain) error {
	aviHeader1 := []byte("RIFF")
	aviHeader2 := []byte("AVI ")
	jpgHeader := []byte{0xff, 0xd8, 0xff}
	for i := range chains {
		c := &(chains[i])
		reader, e := f.GetChainReader(c)
		if e != nil {
			return fmt.Errorf("Error getting reader for chain %d: %w", i, e)
		}
		content, e := io.ReadAll(reader)
		if e != nil {
			return fmt.Errorf("Error reading chain %d content: %w", i, e)
		}
		contentSize := uint32(len(content))
		extension := "bin"
		if bytes.HasPrefix(content, aviHeader1) && bytes.HasPrefix(content[8:],
			aviHeader2) {
			extension = "avi"
			contentSize = binary.LittleEndian.Uint32(content[4:8])
		} else if bytes.HasPrefix(content, jpgHeader) {
			extension = "jpg"
		}
		filename := fmt.Sprintf("%s/data_%04d.%s", outputDir, i, extension)
		fmt.Printf("Saving chain %d/%d as %s (%d bytes).\n", i+1,
			len(chains), filename, contentSize)
		if extension == "jpg" {
			fmt.Printf("  ... Actually, skipping JPG files for now.\n")
			continue
		}
		f, e := os.Create(filename)
		if e != nil {
			return fmt.Errorf("Error opening %s: %w", filename, e)
		}
		_, e = f.Write(content[0:contentSize])
		f.Close()
		if e != nil {
			return fmt.Errorf("Error writing content to %s: %w", filename, e)
		}
		runtime.GC()
	}
	return nil
}

func run() int {
	var imagePath string
	var partitionIndex int
	var outputDir string
	flag.StringVar(&imagePath, "image", "", "The path to the disk image.")
	flag.IntVar(&partitionIndex, "partition_index", 0,
		"The index of the partition containing the FAT32 filesystem.")
	flag.StringVar(&outputDir, "output_directory", "",
		"Dump chain content into this directory, if specified.")
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

	// First, read the MBR on the image and find the FAT partition.
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

	// Read the FAT FS and print information.
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

	// Get chain info and save their content if requested.
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
	if outputDir != "" {
		e = dumpChainContent(fatFS, outputDir, chains)
		if e != nil {
			fmt.Printf("Error dumping chain content: %s\n", e)
			return 1
		}
		fmt.Println("Chain content dumped OK.")
	} else {
		fmt.Println("No output directory specified. Not dumping chain content")
	}

	return 0
}

func main() {
	os.Exit(run())
}
