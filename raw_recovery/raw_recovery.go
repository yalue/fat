// This package is for attemtping to scan an entire image at certain offsets,
// brute-force searching for whatever file types I decide. (Mostly images.)
// It's for cases where the image is completely missing FAT information but
// hasn't been formatted. Will probably work OK for non-FAT filesystems as
// well.
package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
)

// Saves an image to the output directory if the given reader is at the start
// of a recognized image file. Simply prints a message if outputDir is an empty
// string. Returns false, nil if not at an image.
func checkForImage(src io.ReadSeeker, outputDir string, tag int) (bool, error) {
	startOffset, e := src.Seek(0, io.SeekCurrent)
	if e != nil {
		return false, fmt.Errorf("Error getting current offset: %s", e)
	}
	_, format, e := image.Decode(src)
	if e != nil {
		return false, nil
	}
	if outputDir == "" {
		fmt.Printf("Found %s-format image %d, not saving.\n", format, tag)
		return true, nil
	}

	// Determine the number of bytes of the image file so we can save it.
	newOffset, e := src.Seek(0, io.SeekCurrent)
	if e != nil {
		return true, fmt.Errorf("Error determining size of image: %s", e)
	}
	sizeBytes := newOffset - startOffset
	_, e = src.Seek(startOffset, io.SeekStart)
	if e != nil {
		return true, fmt.Errorf("Error rewinding to start of image: %s", e)
	}

	var outputPath string
	if format == "jpeg" {
		// Strip the 'e' from jpeg file extensions because I prefer to.
		outputPath = fmt.Sprintf("%s/pic_%d.jpg", outputDir, tag)
	} else {
		outputPath = fmt.Sprintf("%s/pic_%d.%s", outputDir, tag, format)
	}
	f, e := os.Create(outputPath)
	if e != nil {
		return true, fmt.Errorf("Error creating output file %s: %s",
			outputPath, e)
	}
	defer f.Close()
	_, e = io.CopyN(f, src, sizeBytes)
	if e != nil {
		return true, fmt.Errorf("Error writing %s: %s", outputPath, e)
	}
	fmt.Printf("Image %s saved OK!\n", outputPath)
	return true, nil
}

// Checks for an avi-format file and saves it if found.
func checkForAvi(src io.ReadSeeker, outputDir string, tag int) (bool, error) {
	return false, fmt.Errorf("Not yet implemented")
}

// The top level function that checks whether each sector begins a new file.
func scanForFiles(src io.ReadSeeker, sectorSize int, outputDir string) error {
	endOffset, e := src.Seek(0, io.SeekEnd)
	if e != nil {
		return fmt.Errorf("Error determining size of disk image: %s", e)
	}
	numSectors := endOffset / int64(sectorSize)
	sectorsPerStatus := numSectors / 25
	sectorsThisStatus := int64(0)
	currentTag := 1
	for i := int64(0); i < numSectors; i++ {
		if sectorsThisStatus >= sectorsPerStatus {
			fmt.Printf("Now scanning sector %d/%d (%.02f%%).\n", i+1,
				numSectors, 100.0*(float32(i+1)/float32(numSectors)))
			sectorsThisStatus = 0
		}
		sectorsThisStatus++
		offset := i * int64(sectorSize)
		_, e = src.Seek(offset, io.SeekStart)
		if e != nil {
			return fmt.Errorf("Error seeking to offset %d: %s", offset, e)
		}

		// First, check for image files
		imageFound, e := checkForImage(src, outputDir, currentTag)
		if e != nil {
			return fmt.Errorf("Error checking for image at offset %d: %s",
				offset, e)
		}
		if imageFound {
			currentTag++
		}

		// Next, check for .mp4 videos
		_, e = src.Seek(offset, io.SeekStart)
		if e != nil {
			return fmt.Errorf("Error returning to offset %d: %s", offset, e)
		}
		mp4Found, e := TrySavingMp4(src, outputDir, currentTag)
		if e != nil {
			return fmt.Errorf("Error checking for mp4 at offset %d: %s",
				offset, e)
		}
		if mp4Found {
			currentTag++
		}

		// TODO: Rewind to current offset before checking for other file types
	}
	return nil
}

func run() int {
	var imagePath string
	var outputDir string
	var sectorSize int
	flag.StringVar(&imagePath, "image", "", "The path to the raw disk or "+
		"disk image to scan.")
	flag.StringVar(&outputDir, "output_directory", "",
		"Dump discovered content into this directory, if specified.")
	flag.IntVar(&sectorSize, "sector_size", 512,
		"The size of a \"sector\" in bytes. Sector boundaries will be "+
			"checked for file starts, so smaller sectors may do a finer-"+
			"grained search at the cost of longer execution time.")
	flag.Parse()
	if (imagePath == "") || (sectorSize < 1) {
		fmt.Println("Invalid arguments. Run with -help for more information.")
		return 1
	}
	imageFile, e := os.Open(imagePath)
	if e != nil {
		fmt.Printf("Failed opening %s: %s\n", imagePath, e)
		return 1
	}
	defer imageFile.Close()
	e = scanForFiles(imageFile, sectorSize, outputDir)
	if e != nil {
		fmt.Printf("Error scanning for files: %s\n", e)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}
