package docspec

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

var white Color
var black Color

func init() {
	white = Color{r: 255, g: 255, b: 255}
	black = Color{r: 0, g: 0, b: 0}
}

// PDF is an alias for a pointer to gopdf's FPDF type, created for easier typing
type PDF = *gofpdf.Fpdf

// PDFRenderer implements the DocumentRenderer interface for Adobe pdf documents.
type PDFRenderer struct {
	pdf PDF
}

// FontConfig represents the font data required to initialize the font with the
// underlying FPDF library.
type FontConfig struct {
	Name  string
	Style string
	File  string
}

func documentSizeToRendererString(s documentSize) string {
	switch s {
	case DocumentSizeLetter:
		return "Letter"
	}

	return ""
}

// NewPDFRenderer creates a new renderer that will render the document tree
// into a PDF.  The first font in the list of FontConfig objects will
func NewPDFRenderer(ds documentSize, fontsDir string, fonts ...FontConfig) (*PDFRenderer, error) {
	pdf := gofpdf.New("P", "mm", documentSizeToRendererString(ds), "")
	pdf.SetFontLocation(fontsDir)

	if len(fonts) == 0 {
		return nil, errors.New("must provide at least one font to render a PDF")
	}

	for _, font := range fonts {
		pdf.AddFont(font.Name, font.Style, font.File)
	}

	defaultFont := fonts[0]
	pdf.SetFont(defaultFont.Name, defaultFont.Style, 12)

	renderer := &PDFRenderer{pdf}

	// set some meaningful color defaults for the renderer
	renderer.setFillColor(white)
	renderer.setDrawColor(black)
	renderer.setTextColor(black)

	return renderer, nil
}

func (r *PDFRenderer) walkAndDrawChildren(node *LayoutNode) error {
	for _, child := range node.Children {
		err := r.drawDiv(child)
		if err != nil {
			return err
		}

		err = r.walkAndDrawChildren(child)
		if err != nil {
			return err
		}
	}

	return nil
}

// Render walks the document and renders the result to the underlying FPDF
// instance.
func (r *PDFRenderer) Render(document *Document) (result interface{}, err error) {
	pdf := r.pdf
	for _, page := range document.Children {
		pdf.AddPage()
		for _, node := range page.Children {
			r.drawDiv(node)
			err := r.walkAndDrawChildren(node)
			if err != nil {
				return nil, err
			}
		}
	}
	return r.pdf, nil
}

func (r *PDFRenderer) setFillColor(c Color) {
	r.pdf.SetFillColor(c.r, c.g, c.b)
}

func (r *PDFRenderer) setDrawColor(c Color) {
	r.pdf.SetDrawColor(c.r, c.g, c.b)
}

func (r *PDFRenderer) setTextColor(c Color) {
	r.pdf.SetTextColor(c.r, c.g, c.b)
}

func (r *PDFRenderer) drawDiv(node *LayoutNode) error {
	r.pdf.SetXY(node.X, node.Y)
	r.setDrawColor(node.BorderColor)

	if node.ShowFill {
		r.setFillColor(node.FillColor)
	} else {
		r.setFillColor(white)
	}

	width, err := node.Width.await()
	if err != nil {
		return err
	}

	height, err := node.Height.await()
	if err != nil {
		return err
	}

	r.pdf.CellFormat(
		width,                                    // width
		height,                                   // height
		"",                                       // text string (NA)
		borderStringFromBooleanQuad(node.Border), // border string
		0,                                        // cursor position after draw (NA)
		"",                                       // text alignment (NA)
		node.ShowFill,                            // show fill?
		0,                                        // link id (not supported)
		"",                                       // link str (alternate, not supported for layout nodes)
	)
	// because the fPDF library attempts to handle cursor position itself, we
	// need to reset X and Y here
	r.pdf.SetXY(node.X, node.Y)

	if node.VisualNode != nil {
		err := r.drawVisualNode(node.VisualNode, node)

		// reset X and Y again because drawing the visual node will have messed it up
		r.pdf.SetXY(node.X, node.Y)

		if err != nil {
			return err
		}
	}

	return nil
}

// drawVisualNode type switches the visual node to dispatch the node to the
// appropriate draw function.
func (r *PDFRenderer) drawVisualNode(visualNode interface{}, parentNode *LayoutNode) error {
	switch visualNode.(type) {
	case TextNode:
		r.drawTextNode(visualNode.(TextNode), parentNode)
		break
	case ImageNode:
		r.drawImageNode(visualNode.(ImageNode), parentNode)
	default:
		return fmt.Errorf("PDFRenderer: unhandled visual node type: %s", reflect.TypeOf(visualNode).String())
	}

	return nil
}

// returns an FPDF image type based on the file extension
func imageTypeFromFileName(src string) string {
	fileExtParts := strings.Split(src, ".")
	ext := fileExtParts[len(fileExtParts)-1]

	switch ext {
	case "png":
		return "PNG"
	case "jpg":
		return "JPEG"
	case "jpeg":
		return "JPEG"
	case "gif":
		return "GIF"
	default:
		fmt.Printf("[docspec]: Warning(imageTypeFromFileName): unknown image extension %s\n", ext)
		return ""
	}
}

func (r *PDFRenderer) drawImageNode(i ImageNode, parentNode *LayoutNode) error {
	reader, err := i.getBytesReader()
	if err != nil {
		return err
	}

	r.pdf.RegisterImageOptionsReader(
		i.Src,
		gofpdf.ImageOptions{ImageType: imageTypeFromFileName(i.Src)},
		reader,
	)

	parentRect, err := parentNode.getDrawRect()
	if err != nil {
		return err
	}

	drawRect, err := i.getDrawRect(parentRect)
	if err != nil {
		return err
	}

	x := parentNode.X
	y := parentNode.Y

	switch i.Alignment {
	case ImageCenter:
		x += (parentRect.width - drawRect.width) / 2
		y += (parentRect.height - drawRect.height) / 2
		break
	case ImageStart:
		// nothing to do here, since start = top left of parent
		break
	case ImageEnd:
		x += parentRect.width - drawRect.width
		y += parentRect.height - drawRect.height
		break
	}

	r.pdf.Image(i.Src, x, y, drawRect.width, drawRect.height, false, "", 0, "")

	return nil
}

func (r *PDFRenderer) drawTextNode(t TextNode, parentNode *LayoutNode) error {
	r.setAttributesForTextNode(t)

	targetDrawRect, err := parentNode.getDrawRect()
	if err != nil {
		return err
	}

	lines := r.SplitText(targetDrawRect, t)

	// TODO(#3): we need a way to split request that the lines start at a
	// particular point within the text box.  For example, given that the text
	// is in a "box", we should be able to tell the parent that the children
	// should have a particular alignment.  Basically, the wrapping text node
	// is always the full size of the parent draw rect, when in in reality it
	// should always be width fill and height = children, after we support
	// heightAsChildren on text nodes.

	if t.OverflowBehavior == overflowTruncate {
		r.printTextLine(lines[0], t)
	} else {
		for _, line := range lines {
			r.printTextLine(line, t)
		}
	}

	return nil
}

func (r *PDFRenderer) printTextLine(line string, t TextNode) {
	startX := r.pdf.GetX()
	startY := r.pdf.GetY()
	r.pdf.CellFormat(
		r.pdf.GetStringWidth(line), // width
		t.getLineHeightMM(),        // height
		line,                       // text string (NA)
		"",                         // border string
		0,                          // cursor position after draw (NA)
		"",                         // text alignment (NA)
		false,                      // show fill?
		0,                          // link id (not supported)
		t.Link,                     // link str
	)
	r.pdf.SetX(startX)
	r.pdf.SetXY(startX, startY+t.getLineHeightMM())
}

// Save outputs the created pdf to a given io.Writer
func (r *PDFRenderer) Save(renderResult interface{}, writer io.Writer) error {
	// in this world, renderResult is unused because the fpdf.PDF struct
	// contains all the needed data to save the file
	if r.pdf.Error() != nil {
		return r.pdf.Error()
	}

	r.pdf.Output(writer)
	return nil
}

// SplitText calculates the number and size of lines required to render the
// text node into the given rect.
func (r *PDFRenderer) SplitText(targetRect Rect, textNode TextNode) []string {
	r.setAttributesForTextNode(textNode)

	maxWidth := targetRect.width
	maxHeight := targetRect.height
	lineHeight := textNode.getLineHeightMM()
	currLineHeights := 0.0

	textToGo := strings.TrimSpace(textNode.Text)

	currWidth := r.pdf.GetStringWidth(textToGo)

	if currWidth < maxWidth {
		return []string{textNode.Text}
	}

	// basically use binary search to find the right width for each line...
	results := make([]string, 0)

	wordDelimiter := byte(' ')

	for {
		currWidth := r.pdf.GetStringWidth(textToGo)

		multiplier := maxWidth / currWidth

		if multiplier > 1 {
			if currLineHeights+lineHeight > maxHeight {
				return results
			}
			results = append(results, textToGo)
			return results
		}

		idealCharCount := int(float64(len(textToGo)) * multiplier)

		// get the string up to idealCharCount, then start chopping until we
		// get to delimter
		splitStr := textToGo[:idealCharCount]
		strLen := len(splitStr)
		// if the last char is not a word delimiter, truncate the string by a character
		for (strLen-1 != 0) && splitStr[strLen-1] != wordDelimiter {
			splitStr = splitStr[:strLen-1]
			strLen--
		}

		results = append(results, splitStr)
		currLineHeights += lineHeight

		if currLineHeights+lineHeight > maxHeight {
			return results
		}

		textToGo = strings.Split(textToGo, splitStr)[1]
	}
}

func (s fontStyle) toString() string {
	switch s {
	case FontRegular:
		return ""
	case FontItalic:
		return "I"
	case FontBold:
		return "B"
	case FontUnderscore:
		return "U"
	case FontStrikeOut:
		return "S"
	default:
		return ""
	}
}

func borderStringFromBooleanQuad(quad BooleanQuad) string {
	result := ""

	if quad.bottom {
		result += "B"
	}

	if quad.top {
		result += "T"
	}

	if quad.left {
		result += "L"
	}

	if quad.right {
		result += "R"
	}

	return result
}

func (r *PDFRenderer) setAttributesForTextNode(textNode TextNode) {
	r.pdf.SetFont(textNode.FontFamily, textNode.FontStyle.toString(), textNode.FontSize)
	r.setTextColor(textNode.Color)
}
