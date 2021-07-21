package docspec

import (
	"errors"
	"fmt"
)

// ------------------- future data type --------------------------

// identitfyFutureDefinition is a definition for a future that simply returns
// the value cast to a size.
func identityFutureDefinition(node *LayoutNode, params interface{}) (Size, error) {
	return params.(Size), nil
}

type futureState = int
type futureDefinition = func(node *LayoutNode, params interface{}) (Size, error)

const (
	futureIncomplete futureState = iota
	futureComplete
)

// Future is a datatype modeling a value defined by a recursive function that
// cannot be evaluated until the entire layout tree is constructed.  For example,
// the height of a node is often defined by the height of its children, which
// do not yet exist at the time of the node's construction.  While in this way
// it is conceptually similar to a "Future" as used in many programming
// language's asyncIO implementation, it has nothing to do with concurrency.
type Future struct {
	value      Size
	state      futureState
	definition futureDefinition
	params     interface{}
	node       *LayoutNode
}

func (f Future) isUnitialized() bool {
	// only the constructor can set the definition, so we know that if the
	// future has no definition, then it is not initialized and is just a
	// "blank"
	return f.definition == nil
}

// newIncompleteFuture creates a future from a given functional definition with
// whatever params will be required at evaluation time to construct the
// future's value
func newIncompleteFuture(def futureDefinition, params interface{}) Future {
	return Future{
		value:      emptySize,
		state:      futureIncomplete,
		definition: def,
		params:     params,
	}
}

// newCompleteFuture provides an already completed future, when you want
// to initialize a value that would normally be a future, but where you already
// know the value
func newCompleteFuture(value Size) Future {
	val := Future{
		value:      value,
		state:      futureComplete,
		definition: identityFutureDefinition,
		params:     value,
	}

	return val
}

// await gets the value from a future, and returns the value, or an error in
// case the value cannot be resolved
// TODO: it would be really nice to detect infinitely recursive situations,
// such as when a parent is defined as a function of its children, but the
// children are defined as a function of their parent, without just having Go's
// runtime stack overflow.  The resolver ought to detect that this is an
// invalid state of affairs before attempting to resolve the value.
func (f *Future) await() (Size, error) {
	if f.node == nil {
		return emptySize, invariantViolation("future's node is nil.  This indicates an error in a constructor for LayoutNode, as a LayoutNode constructor should always assign the node to each future in the node")
	}

	if f.state == futureComplete {
		return f.value, nil
	}

	return f.definition(f.node, f.params)
}

// isKnown tells us whether the value of the future is already known.  This is
// useful to detect circularity.  For example, if a child element's width
// depends on its parent's width, but the parent's width is not yet known, then
// we can throw an error, because we know that calling await() on the parent's
// width would result in a stack overflow because document trees are always
// walked from the top down.
func (f *Future) isKnown() bool {
	return f.state == futureComplete
}

// ------------------- common future definitions --------------------------

func heightAsChildren(node *LayoutNode, params interface{}) (Size, error) {

	if len(node.Children) == 1 && node.Children[0].VisualNode != nil {
		childNode := node.Children[0]
		// we are dealing with a leaf node that has a visual node with an
		// inherent size, which we should use if possible
		switch childNode.VisualNode.(type) {
		case TextNode:
			return childNode.rendererContext.GetInherentTextRect(childNode.VisualNode.(TextNode)).height + node.Padding.top + node.Padding.bottom, nil
		default:
			return emptySize, errors.New("requested height as children, but reached a leaf node with no inherent height")
		}
	}

	if node.ChildFlowDirection == FlowHorizontal {
		// get the height of the tallest child
		tallest := 0.0
		for _, child := range node.Children {
			cH, err := child.getBoundingHeight()
			if err != nil {
				return emptySize, err
			}
			height := cH
			if height > tallest {
				tallest = height
			}
		}
		return tallest + node.Padding.top + node.Padding.bottom, nil
	}
	// get the sum of the heights of the children
	result := emptySize
	for _, child := range node.Children {
		cH, err := child.getBoundingHeight()
		if err != nil {
			return emptySize, err
		}
		height := cH
		result += height

	}
	return result + node.Padding.top + node.Padding.bottom, nil
}

func widthAsChildren(node *LayoutNode, params interface{}) (Size, error) {
	// in the case that we're dealing with a node that wraps a visual node, we
	// can handle the case in a special way, because if said node has it's
	// width defined as widthAsChildren (which we do, since we're in that
	// function definition), then we need to extract the width and height from
	// the visual node underlying the first child.  If we don't, then the two
	// wrapper divs around visual nodes will cause an infinite loop.
	if len(node.Children) == 1 && node.Children[0].VisualNode != nil {
		childNode := node.Children[0]
		// we are dealing with a leaf node that has a visual node with an
		// inherent size, which we should use if possible
		switch childNode.VisualNode.(type) {
		case TextNode:
			return childNode.rendererContext.GetInherentTextRect(childNode.VisualNode.(TextNode)).width + node.Padding.left + node.Padding.right, nil
		default:
			return emptySize, errors.New("requested width as children, but reached a leaf node with no inherent height")
		}
	}

	if node.ChildFlowDirection == FlowVertical {
		// get the width of the widest child
		widest := 0.0
		for _, child := range node.Children {
			cW, err := child.getBoundingWidth()
			if err != nil {
				return emptySize, err
			}
			width := cW
			if width > widest {
				widest = width
			}
		}

		return widest + node.Padding.left + node.Padding.right, nil
	}
	// sums the widths of the children
	result := emptySize
	for _, child := range node.Children {
		cW, err := child.getBoundingWidth()
		if err != nil {
			return emptySize, err
		}
		width := cW
		if err != nil {
			return emptySize, err
		}
		result += width

	}
	return result + node.Padding.left + node.Padding.right, nil
}

func heightFill(node *LayoutNode, params interface{}) (Size, error) {
	result := emptySize

	parentHeight := emptySize
	var siblings []*LayoutNode
	var flowDirection childFlowDirection

	// first attempt to use the parent node
	if node.Parent != nil {
		h, err := node.Parent.getDrawHeight()
		if err != nil {
			return emptySize, err
		}
		parentHeight = h
		siblings = node.Parent.Children
		flowDirection = node.Parent.ChildFlowDirection
	} else if node.Page != nil {
		parentDrawRect := node.Page.getDrawRect()
		parentHeight = parentDrawRect.height
		siblings = node.Page.Children
		flowDirection = FlowVertical
	}

	if parentHeight == emptySize {
		return result, fmt.Errorf("orphan node(%s): parent has height of 0", node.ID)
	}

	if flowDirection == FlowHorizontal {
		return heightPercentage(node, 100.0)
	}

	siblingHeights := emptySize
	for _, siblingNode := range siblings {
		if siblingNode != node {
			hR, e := siblingNode.getBoundingHeight()
			if e != nil {
				return emptySize, e
			}
			siblingHeights += hR
		}
	}

	result = parentHeight - siblingHeights
	result -= node.Margin.top + node.Margin.bottom

	// return the difference between the parent height and all the sibling heights
	return result, nil
}

func widthFill(node *LayoutNode, params interface{}) (Size, error) {
	result := emptySize

	parentWidth := emptySize
	var siblings []*LayoutNode
	var flowDirection childFlowDirection

	// first attempt to use the parent node
	if node.Parent != nil {
		w, err := node.Parent.getDrawWidth()
		if err != nil {
			return emptySize, err
		}
		parentWidth = w
		siblings = node.Parent.Children
		flowDirection = node.Parent.ChildFlowDirection
	} else if node.Page != nil {
		parentDrawRect := node.Page.getDrawRect()
		parentWidth = parentDrawRect.width
		siblings = node.Page.Children
		flowDirection = FlowVertical
	}

	if parentWidth == emptySize {
		return result, fmt.Errorf("orphan node(%s): parent has width of 0", node.ID)
	}

	if flowDirection == FlowVertical {
		return widthPercentage(node, 100.0)
	}

	siblingWidths := emptySize
	for _, siblingNode := range siblings {
		if siblingNode != node {
			wR, e := siblingNode.getBoundingWidth()
			if e != nil {
				return emptySize, e
			}
			siblingWidths += wR
		}
	}

	result = parentWidth - siblingWidths
	result -= node.Margin.left + node.Margin.right

	// return the difference between the parent Width and all the sibling Widths
	return result, nil
}

func widthPercentage(node *LayoutNode, params interface{}) (Size, error) {
	var parentDrawWidth Size
	result := emptySize

	if node.Parent != nil {
		r, err := node.Parent.getDrawWidth()
		if err != nil {
			return result, err
		}

		parentDrawWidth = r
	} else if node.Page != nil {
		parentDrawWidth = node.Page.getDrawRect().width
	}

	if parentDrawWidth == 0 {
		return result, fmt.Errorf("orphan node(%s): parent has width of 0", node.ID)
	}

	percentage := params.(Size)
	result = parentDrawWidth * (percentage / 100)

	return result, nil
}

func heightPercentage(node *LayoutNode, params interface{}) (Size, error) {
	var parentDrawHeight Size
	result := emptySize

	if node.Parent != nil {
		r, err := node.Parent.getDrawHeight()
		if err != nil {
			return result, err
		}
		parentDrawHeight = r
	} else if node.Page != nil {
		parentDrawHeight = node.Page.getDrawRect().height
	}

	if parentDrawHeight == 0 {
		return result, fmt.Errorf("orphan node(%s): parent has height of 0", node.ID)
	}

	percentage := params.(Size)
	result = parentDrawHeight * (percentage / 100)

	return result, nil
}

/* ------------------------------ Exported Constructors ---------------------------- */

// HeightPercentage creates a future that will resolve to a value that is a
// percentage of the parent draw rect's height.
func HeightPercentage(percentage Size) Future {
	return newIncompleteFuture(heightPercentage, percentage)
}

// WidthPercentage creates a future that will resolve to a value that is a
// percentage of the parent draw rect's width.
func WidthPercentage(percentage Size) Future {
	return newIncompleteFuture(widthPercentage, percentage)
}

// HeightAsChildren creates a future that will resolve to a value that is the
// height of the full document tree below the current node.
func HeightAsChildren() Future {
	return newIncompleteFuture(heightAsChildren, nil)
}

// WidthAsChildren creates a future that will resolve to a value that is the
// width of the full document tree below the current node.
func WidthAsChildren() Future {
	return newIncompleteFuture(widthAsChildren, nil)
}

// HeightFill creates a future that will resolve to whatever space is left
// available within the draw rect of the parent after all sibling trees have
// been resolved.
func HeightFill() Future {
	return newIncompleteFuture(heightFill, nil)
}

// WidthFill creates a future that will resolve to whatever space is left
// available within the draw rect of the parent after all sibling trees have
// been resolved.
func WidthFill() Future {
	return newIncompleteFuture(widthFill, nil)
}

// StaticSize returns a future that is already resolved to a static value.
func StaticSize(size Size) Future {
	return newCompleteFuture(size)
}
