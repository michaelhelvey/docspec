package docspec

import (
	"bytes"
	"image"
	"io"
	"os"

	// registers the gif format
	_ "image/gif"
	// registers the jpeg format
	_ "image/jpeg"
	// registers the png format
	_ "image/png"
)

/*
Visual nodes are nodes which have no children, and render the actual content
inside layout nodes.  Examples of content would include text or images.
*/

/* ----------------------------- Text Node ---------------------------- */

type textAlignment = int

const (
	textLeft textAlignment = iota
	textCenter
	textRight
)

type overflowBehavior = int

const (
	// wrap the text into as many lines as possible, truncating the last line
	overflowWrap overflowBehavior = iota
	// do not wrap or throw an error, simply cut off the text
	overflowTruncate
)

type fontStyle int

const (
	// FontRegular describes a regular font in a TextNode
	FontRegular fontStyle = iota
	// FontItalic describes an instalic font in a TextNode
	FontItalic
	// FontBold describes a bold font in a TextNode
	FontBold
	// FontUnderscore describes a underscored font in a TextNode
	FontUnderscore
	// FontStrikeOut describes a strikethrough font in a TextNode
	FontStrikeOut
)

// TextNode represents a paragraph of text, possibly containing a link
type TextNode struct {
	Text       string
	FontFamily string
	FontStyle  fontStyle
	// size of the font in points (1pt == 0.3528mm)
	FontSize Size
	Color    Color
	// multiplier used to determine line height based on font size
	LineHeight       Size
	Alignment        textAlignment
	Link             string
	OverflowBehavior overflowBehavior
}

// getLineHeightMM returns the line height in mm for a given text node
func (n *TextNode) getLineHeightMM() float64 {
	conversionFactor := 0.3528
	return (n.FontSize * conversionFactor) * n.LineHeight
}

/* -----------------------------  Image Node ---------------------------- */

type imageFit = int
type imageAlignment = int

// Note: we're specifying both aspect ratio and alignment in the same enum
// here.  We may want to split these out.
const (
	// fill draw rect of parent with image, do not preserve aspect ratio
	ImageStretch imageFit = iota
	// preserve aspect ratio of original image
	ImagePreserve
)

const (
	// ImageCenter places the image in the center of the parent
	ImageCenter imageAlignment = iota
	// ImageStart preserve aspect ratio, and aligns longest edge of image along
	// the start of the parent draw rect.  if the image is taller than it is
	// wide, this will be the left edge of the draw rect.  If the image is
	// wider than is is tall, this will be the bottom of the parent draw rect.
	ImageStart
	// ImageEnd preserves aspect ratio and is the inverse of `ImageStart`
	ImageEnd
)

// ImageNode represents an image drawn from a local file
type ImageNode struct {
	// full path to a local file from which to read the image.  Currently only
	// gif, jpeg, and png file types are supported.
	Src           string
	RatioBehavior imageFit
	Alignment     imageAlignment
	// if an operation requires reading the data from the file, cache it here
	// for future use
	data []byte
}

// getBytesReader creates an io.Reader interface that will read out the bytes
// of the image file
func (n *ImageNode) getBytesReader() (io.Reader, error) {
	if n.data == nil {
		// we have no data, so we need to load it
		d, err := os.ReadFile(n.Src)
		if err != nil {
			return nil, err
		}
		n.data = d
	}

	return bytes.NewReader(n.data), nil
}

// getDrawRect calculates the width and height of the image for the required
// fit.
func (n *ImageNode) getDrawRect(parentRect Rect) (Rect, error) {
	switch n.RatioBehavior {
	case ImageStretch:
		{
			// to stretch the image, simply render it into the rect of its
			// container
			return parentRect, nil
		}
	case ImagePreserve:
		{
			dataReader, err := n.getBytesReader()
			if err != nil {
				return Rect{}, err
			}

			m, _, err := image.Decode(dataReader)

			if err != nil {
				return Rect{}, err
			}

			bounds := m.Bounds()

			w := bounds.Dx()
			h := bounds.Dy()

			typeWidth := 1
			typeHeight := 2

			var longestSide int
			var longestSideType int

			if w > h {
				longestSide = w
				longestSideType = typeWidth
			} else {
				longestSide = h
				longestSideType = typeHeight
			}

			var scaleFactor float64

			// compare sides of image & parent draw rect to get scale factor
			if longestSideType == typeHeight {
				scaleFactor = parentRect.height / float64(longestSide)
			} else {
				scaleFactor = parentRect.width / float64(longestSide)
			}

			return Rect{
				width:  float64(w) * scaleFactor,
				height: float64(h) * scaleFactor,
			}, nil

		}
	}
	return Rect{}, nil
}
