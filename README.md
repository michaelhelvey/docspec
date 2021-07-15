# Docspec

[![MIT
licensed](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/michaelhelvey/docspec/master/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/michaelhelvey/docspec.svg)](https://pkg.go.dev/github.com/michaelhelvey/docspec)

A work-in-progress PDF renderer, used to provide a declarative API similar to
CSS for laying out simple PDF documents.

[This port](https://github.com/jung-kurt/gofpdf) of FPDF for Go provides all
the basic primitives.  I want to stress that all the interesting PDF-specific
stuff happens in that library thanks to its authors.

Nevertheless, I wanted to create a layout engine where I could specify layouts
more declaratively (similar to CSS), rather than calculating all the rects and x
and y values myself for each page.  The layout engine represents the document as
a tree, and walks the tree to calculate widths and heights of each node on the
tree in terms of defined layout contraints.

Used internally at my job but open sourced here on my profile for more general
use.  For the record, I have no intention as of yet of expanding the project
beyond what I require for work, but forks or contributions are welcome if
desired.

## Installation

- `go get github.com/michaelhelvey/docspec`

## Documentation && Usage

See [example/main.go](./example/main.go) for a usage example.

## TODO:

- [ ] Write a good unit test suite.  Shouldn't be too difficult as all the heavy
  lifting happens in the creation of the document tree, which is much easier to
  assert things about than a PDF document.

- [ ] Improve the automated API documentation enough to be able to create a good
  godoc site if desired.

## Authors

- Michael Helvey <michael.helvey1@gmail.com>

## License

MIT
