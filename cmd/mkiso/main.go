package main

import (
	"flag"
	"fmt"
	"github.com/davejbax/go-iso9660"
	"io/fs"
	"log"
	"os"
)

func main() {
	dir := flag.String("dir", "", "Directory to use as source for ISO file")
	output := flag.String("output", "mkiso.iso", "Output file name/path")

	flag.Parse()

	if len(*dir) == 0 {
		flag.Usage()
	}

	img, err := iso9660.NewImage(os.DirFS(*dir).(fs.ReadDirFS))
	if err != nil {
		log.Fatal(err)
	}

	outputFile, err := os.OpenFile(*output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	defer outputFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	if _, err := img.WriteTo(outputFile); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("successfully wrote file %s\n", *output)
}
