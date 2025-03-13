# go-iso9660

[![Go Reference](https://pkg.go.dev/badge/github.com/davejbax/go-iso9660.svg)](https://pkg.go.dev/github.com/davejbax/go-iso9660)

🚧 **Work in progress** 🚧

Library for writing ISO9660 image files, with planned support for El Torito, Joliet, and Rock Ridge.

This library aims to be performant, efficient, and ergonomic. In line with these aims, it:

* Does not require you to specify the size of an ISO image in advance
* Does not create a 'staging area' for image files, or write to disk at all. You can simply provide an [`fs.ReadDirFS`](https://pkg.go.dev/io/fs#ReadDirFS), and files will be read as/when needed.
* Does not require you to write to a file; any `io.Writer` is supported
* Uses standard Go interfaces where possible