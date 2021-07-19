package docspec

import "fmt"

// LayoutNode represents a rectangular container into which content can be
// rendered.  Layout nodes control all layout such as width, height, padding,
// and so on.  Value nodes such as a text nodes or image nodes are used to render
// content into the rect defined by the LayoutNode.
//
// Splitting over pages is determined by testing if the node's height would
// take it over the draw rect of the current page.  If it would, then the tree
// is traversed backwards until the parent node that is a direct child of the
// Page is found, and the origin of that node is placed at the origin of the
// page, and x,y coordination calculation continues down the page as before.
// If that parent node's height is greater than the height of the page (i.e.
// the node cannot fit on a single page), then an error is raised.  An error is
// also raised is a node's measured height is greater than the height of any page,
// meaning that the above test would result in infinite recursion.
//
// Layout calculation takes place in two phases: first the widths and heights
// of all the nodes are calculated, and then the x,y coordinates of each rect
// is calculated.  Thus the entire document tree is iterated over at least 3
// times: twice as described above, and then at least once again during rendering.
type LayoutNode struct {
	Children           []*LayoutNode
	VisualNode         interface{}
	Parent             *LayoutNode
	Page               *Page
	X                  Size
	Y                  Size
	Width              Future
	Height             Future
	Border             BooleanQuad
	BorderColor        Color
	ShowFill           bool
	FillColor          Color
	Padding            SizesQuad
	Margin             SizesQuad
	ChildAlignment     LayoutChildAlignment
	ChildFlowDirection childFlowDirection
}

// Clone returns a pointer to a copy of the original node tree, with all
// child nodes recursively cloned.
func (n *LayoutNode) Clone() *LayoutNode {
	root := *n
	root.Children = make([]*LayoutNode, 0)

	for _, child := range n.Children {
		root.addChild(child.Clone())
	}

	return &root
}

func (n *LayoutNode) addChild(node *LayoutNode) {
	n.Children = append(n.Children, node)
}

// returns the bounding rect of the node, i.e. the rectangle inside of which
// nothing but the node can render (defined as the node's rect itself + the
// node's margins)
func (n LayoutNode) getBoundingRect() (Rect, error) {
	width, err := n.Width.await()

	if err != nil {
		return Rect{}, err
	}

	height, err := n.Height.await()

	if err != nil {
		return Rect{}, err
	}

	width += n.Margin.left + n.Margin.right
	height += n.Margin.top + n.Margin.bottom

	return Rect{width, height}, nil
}

// getRenderRect gets the rect into which the node renders.  In other words,
// the rectangle that lies outside the draw rect of the node (because of
// padding) but inside the bounding rect (because of margin).
func (n LayoutNode) getRenderRect() (Rect, error) {
	width, err := n.Width.await()
	if err != nil {
		return Rect{}, err
	}

	height, err := n.Height.await()
	if err != nil {
		return Rect{}, err
	}

	return Rect{width, height}, nil
}

// getDrawRect gets the rect into which children of the node must render,
// defined as the render rect minus the inner padding of the node.
func (n LayoutNode) getDrawRect() (Rect, error) {
	width, err := n.Width.await()
	if err != nil {
		return Rect{}, nil
	}

	height, err := n.Height.await()
	if err != nil {
		return Rect{}, nil
	}

	width -= n.Padding.left + n.Padding.right
	height -= n.Padding.top + n.Padding.bottom

	return Rect{width, height}, nil
}

func (n LayoutNode) log() {
	width, _ := n.Width.await()
	height, _ := n.Height.await()

	fmt.Printf("layout node; x: %f, y: %f, width: %f, height: %f\n", n.X, n.Y, width, height)
}

// ---------------------------- Supporting types ----------------------------

type childAlignment int

type childFlowDirection int

const (
	// FlowVertical represents a list of children that should flow from top to
	// bottom within the parent.
	FlowVertical = iota
	// FlowHorizontal represents a list of children that should flow from left
	// to right within the parent.
	FlowHorizontal
)

// Note that we intentially leave out "justify-between" as this entire
// layout engine is for statically known layouts, so setting width
// percentages manually is better.

const (
	// Start is equal to justify-content: start in CSS
	Start childAlignment = iota
	// End is equal to justify-content: end in CSS
	End
	// Center is equal to justify-content: center in CSS
	Center
)

// LayoutChildAlignment configures where on the x and y axis inside a given
// node to position that node's children
type LayoutChildAlignment struct {
	Vertical   childAlignment
	Horizontal childAlignment
}

// Rect represents a rectangle in unknown space (no x and y values are
// specified, only size)
type Rect struct {
	width  Size
	height Size
}

// SizesQuad represents values that occur for each of the four sides of a
// rectangle, such as padding and margin
type SizesQuad struct {
	left   Size
	right  Size
	top    Size
	bottom Size
}

// NewSizeQuad creates a quadrilateral of sizes
func NewSizeQuad(top, right, bottom, left Size) SizesQuad {
	return SizesQuad{
		top:    top,
		right:  right,
		bottom: bottom,
		left:   left,
	}
}

// NewSingletonSizeQuad creates a quadrilateral of sizes where each side has
// the same value as every other side.
func NewSingletonSizeQuad(value Size) SizesQuad {
	return SizesQuad{
		left:   value,
		right:  value,
		top:    value,
		bottom: value,
	}
}

// BooleanQuad represents on/off state for each side of a rectangle
type BooleanQuad struct {
	left   bool
	right  bool
	top    bool
	bottom bool
}

// NewBooleanQuad creates a quadrilateral of boolean values
func NewBooleanQuad(top, right, bottom, left bool) BooleanQuad {
	return BooleanQuad{
		top:    top,
		right:  right,
		bottom: bottom,
		left:   left,
	}
}

// NewSingletonBooleanQuad creates a quadrilateral of boolean values where each
// value is equal to every other value.
func NewSingletonBooleanQuad(value bool) BooleanQuad {
	return BooleanQuad{
		left:   value,
		right:  value,
		top:    value,
		bottom: value,
	}
}

// Color represents an RGB color
type Color struct {
	r int
	g int
	b int
}

// NewColor constructs an RGB color from its arguments
func NewColor(red, green, blue int) Color {
	return Color{red, green, blue}
}

// ---------------------- Node Constructors ---------------------

// NoChildren is the NoOp callback for creating node children.
func NoChildren(parent *LayoutNode) {}

// CreateNodeList is syntactic suguar for creating a list of layout nodes
func CreateNodeList(nodes ...*LayoutNode) []*LayoutNode {
	return nodes
}

// LayoutNodeProps is the subset of the properties of the LayoutNode struct
// which can be passed into node constructors.
type LayoutNodeProps struct {
	Border             BooleanQuad
	BorderColor        Color
	ShowFill           bool
	FillColor          Color
	Padding            SizesQuad
	Margin             SizesQuad
	Width              Future
	Height             Future
	ChildAlignment     LayoutChildAlignment
	ChildFlowDirection childFlowDirection
}

// mergeProps merges LayoutNodeProps (which is a subset of LayoutNode) into the
// node.  In C I could just rely on alignment and cast stuff but I'm not sure
// how to do that here declaratively without doing a lot of expensive runtime
// introspection.  I'm not a huge fan of this method, because it requires
// manually keeping fields up to date, but using introspection might be even
// uglier, I'm not sure.  TODO: create a more declarative way of merging props.
func (n *LayoutNode) mergeProps(props LayoutNodeProps) {
	n.Border = props.Border
	n.BorderColor = props.BorderColor
	n.ShowFill = props.ShowFill
	n.FillColor = props.FillColor
	n.Padding = props.Padding
	n.Margin = props.Margin
	n.ChildAlignment = props.ChildAlignment
	n.ChildFlowDirection = props.ChildFlowDirection
	n.Width = props.Width
	n.Height = props.Height
}

// Div inserts a plain layout node into the document tree
func Div(parent *LayoutNode, options LayoutNodeProps, cb func(*LayoutNode)) *LayoutNode {
	// create the div and add it to to the parent's children...
	newNode := &LayoutNode{
		Parent:   parent,
		Page:     nil, // if the parent is a page, that will be determined later by the resolver
		Children: make([]*LayoutNode, 0),
	}
	newNode.mergeProps(options)

	newNode.Width.node = newNode
	newNode.Height.node = newNode

	cb(newNode)

	if parent != nil {
		parent.Children = append(parent.Children, newNode)
	}

	return newNode
}

// Text inserts a text component in the document tree
func Text(parent *LayoutNode, options LayoutNodeProps, textProps TextNode) *LayoutNode {
	// a "text node" is really three nodes -- a layout node for layout, a
	// wrapper layout node for padding, and inside a visual node for the actual
	// text.

	wrapperNode := &LayoutNode{
		Parent:     nil,
		Page:       nil,
		VisualNode: textProps,
		Width:      newIncompleteFuture(widthFill, nil),
		Height:     newIncompleteFuture(heightFill, nil),
	}

	wrapperNode.Width.node = wrapperNode
	wrapperNode.Height.node = wrapperNode

	layoutNode := &LayoutNode{
		Parent:   parent,
		Page:     nil,
		Children: []*LayoutNode{wrapperNode},
	}
	wrapperNode.Parent = layoutNode

	layoutNode.mergeProps(options)

	layoutNode.Width.node = layoutNode
	layoutNode.Height.node = layoutNode

	if parent != nil {
		parent.Children = append(parent.Children, layoutNode)
	}

	return layoutNode
}

// Image inserts an image component into the document tree.  It has no callback
// because a visual node by definition must be a leaf of the document tree.
func Image(parent *LayoutNode, options LayoutNodeProps, imageProps ImageNode) *LayoutNode {
	// an "image node" is really three nodes -- a layout node for layout, a
	// node for apdding, and inside a visual node for the actual image.
	wrapperNode := &LayoutNode{
		Parent:     nil,
		Page:       nil,
		VisualNode: imageProps,
		Width:      newIncompleteFuture(widthFill, nil),
		Height:     newIncompleteFuture(heightFill, nil),
	}

	wrapperNode.Width.node = wrapperNode
	wrapperNode.Height.node = wrapperNode

	layoutNode := &LayoutNode{
		Parent:   parent,
		Page:     nil,
		Children: []*LayoutNode{wrapperNode},
	}

	wrapperNode.Parent = layoutNode
	layoutNode.mergeProps(options)

	layoutNode.Width.node = layoutNode
	layoutNode.Height.node = layoutNode

	if parent != nil {
		parent.Children = append(parent.Children, layoutNode)
	}

	return layoutNode
}
