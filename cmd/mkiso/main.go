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
	joliet := flag.Bool("joliet", false, "Whether to use Joliet extension")

	flag.Parse()

	if len(*dir) == 0 {
		flag.Usage()
	}

	builder := iso9660.New(os.DirFS(*dir).(fs.ReadDirFS)).
		WithMetadata(iso9660.Metadata{ApplicationIdentifier: "foobar"})

	if *joliet {
		builder = builder.WithJoliet()
	}

	image, err := builder.Build()
	if err != nil {
		log.Fatal(err)
	}

	outputFile, err := os.OpenFile(*output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	defer outputFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	if _, err := image.WriteTo(outputFile); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("successfully wrote file %s\n", *output)
}
