package docspec

import (
	"fmt"
	"io"
)

/*
DocSpec is a DSL for rendering rectangle-based layouts.  While it currently
targets PDF rendering, any backend (think, SDL, image formats, HTML) could
theoretically be constructed that walks the document tree and follows the spec
laid out by the current PDF renderer.

The fundamental unit of layout is the document, which forms a list of pages,
along with some universal settings such as page margin, height, and width.  All
such settings are set globally at the document level rather than by page,
because page breaks and positioning are abstracted from userspace.  For
directly drawing onto a page, or manually managing page breaks, lower level
access to the renderer than DocSpec currently provides would be required.
*/

// Size is the fundmental unit of measurement used for widths, heights,
// margins, and other measured areas.
type Size = float64

// emptySize is the empty value for a 'size' type.  Useful if you want to
// assign the empty value explicitly rather than letting Go do it for you.
const emptySize Size = 0.0

// DocumentRenderer defines the interface which must be implemented by a type
// in order to render a document tree into a document.  It's defined here in
// the docspec package so that users can write their own renderer beyond the
// ones already defined by docspec.
type DocumentRenderer interface {
	// Render takes a document and renders it into some result value suitable
	// for the type of renderer.  For example, for an FPDF renderer, the result
	// would be an FPDF struct that can be saved to a file.  For a JPEG
	// renderer, the result would be JPEG byte data.  And so on.
	Render(root *Document) (result interface{}, err error)

	// SaveToFile takes in the render result and saves it via a io.Writer
	Save(renderResult interface{}, writer io.Writer) error

	// SplitText calculates from font attributes present in TextNode the
	// optimal number and size of lines required to render the text node into
	// the given rect.  This method is present in the renderer so that the main
	// layout engine can communicate with the renderer to prepare text node's
	// with different text overflow behaviors.
	SplitText(targetRect Rect, textNode TextNode) []string

	// GetTextWidth gets the width that a piece of text would have if it was
	// not split.  This is so that a text node can have an inherent width and
	// height can be used in a Height/WidthAsChildren definition.
	GetInherentTextRect(textNode TextNode) Rect
}

// -----------------------  Simple Types --------------------------

// DocumentBuilder specifies the public interface for building documents
type DocumentBuilder struct {
	renderer DocumentRenderer
	document *Document
	nodes    []*LayoutNode
}

type documentSize = int

const (
	// DocumentSizeLetter represents a page with the size U.S. Letter
	DocumentSizeLetter documentSize = iota
)

func getDocumentSize(s documentSize) (Size, Size) {
	switch s {
	case DocumentSizeLetter:
		return 216.0, 279.0
	}

	return 0.0, 0.0
}

// NewDocumentBuilder creates a new document with the specified margin and size, and a renderer
func NewDocumentBuilder(renderer DocumentRenderer, size documentSize, margin Size) *DocumentBuilder {
	w, h := getDocumentSize(size)
	return &DocumentBuilder{
		renderer: renderer,
		document: newDocument(w, h, margin),
	}
}

// RenderToWriter outputs the document tree to a specified writer.  For
// example, to render the document tree as a PDF, you would create a
// DocumentBuilder with a pdf renderer as the renderer, and call RenderToWriter
// where the writer is the file that you want to save the PDF to.
func (d *DocumentBuilder) RenderToWriter(writer io.Writer) error {
	result, err := d.renderer.Render(d.document)
	if err != nil {
		return err
	}
	err = d.renderer.Save(result, writer)
	return err
}

// CreateDocumentTree traverses the given list of nodes and calculates the
// coordinates for all rects in the tree.
func (d *DocumentBuilder) CreateDocumentTree(nodeList []*LayoutNode) error {
	d.nodes = nodeList
	// before we resolve the node rect positions, recursively walk the tree and
	// set the renderer context
	for _, node := range nodeList {
		setDocumentRendererContext(node, d.renderer)
	}

	err := resolveNodeRectPositions(d)
	return err
}

func setDocumentRendererContext(node *LayoutNode, renderer DocumentRenderer) {
	node.rendererContext = renderer
	for _, child := range node.Children {
		setDocumentRendererContext(child, renderer)
	}
}

// PrettyPrintDocumentTree prints the AST of the document with sizing
// information.  This method will only produce useful output after
// `CreateDocumentTree` has been called.
func (d *DocumentBuilder) PrettyPrintDocumentTree() {
	for _, page := range d.document.Children {
		fmt.Printf("Page (w: %f, h: %f) {\n", page.Width, page.Height)

		for _, node := range page.Children {
			prettyPrintNodeTree(node, 1)
		}

		fmt.Printf("}\n")
	}
}

func prettyPrintNodeTree(root *LayoutNode, depth int) {
	result := ""
	footer := ""
	for n := 0; n < depth; n++ {
		footer += "\t"
		result += "\t"
	}

	result += "LayoutNode (x: %f, y: %f, w: %f, h: %f, margin: %v, padding: %v) {\n"

	width, err := root.Width.await()
	if err != nil {
		panic(err)
	}

	height, err := root.Height.await()

	if err != nil {
		panic(err)
	}

	fmt.Printf(result, root.X, root.Y, width, height, root.Margin, root.Padding)
	for _, child := range root.Children {
		prettyPrintNodeTree(child, depth+1)
	}

	if root.VisualNode != nil {
		visualLeader := footer + "\t"
		fmt.Printf(visualLeader+"VisualNode (%+v)\n", root.VisualNode)
	}

	footer += "}"
	fmt.Println(footer)
}

// Document is the fundamental unit of layout. It encapsulates global
// settings and a list of pages.  The list of pages is manipulated by the
// layout engine to create page breaks, and is thus not exposed to userspace.
type Document struct {
	Children []*Page
}

func newDocument(width, height, margin Size) *Document {
	page := Page{
		Children: make([]*LayoutNode, 0),
		Width:    width,
		Height:   height,
		Margin:   NewSingletonSizeQuad(margin),
	}

	document := Document{
		Children: []*Page{&page},
	}

	return &document
}

// Page is a special layout node that is always statically sized at the size
// specified in the document settings.
type Page struct {
	Children []*LayoutNode
	Width    Size
	Height   Size
	Margin   SizesQuad
}

// getDrawRect returns the area inside the page into which we can render child nodes
func (p *Page) getDrawRect() Rect {
	width := p.Width
	height := p.Height

	width -= p.Margin.left + p.Margin.right
	height -= p.Margin.top + p.Margin.bottom

	return Rect{width, height}
}

func (p *Page) addNode(node *LayoutNode) {
	p.Children = append(p.Children, node)
	node.Page = p
}

func (d *Document) addPage() *Page {
	previousPage := d.Children[len(d.Children)-1]
	page := &Page{
		Children: make([]*LayoutNode, 0),
		Width:    previousPage.Width,
		Height:   previousPage.Height,
		Margin:   previousPage.Margin,
	}
	d.Children = append(d.Children, page)
	return page
}
