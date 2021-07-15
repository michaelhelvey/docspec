package docspec

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
	result := emptySize
	for _, child := range node.Children {
		height, err := child.Height.await()
		if err != nil {
			return emptySize, err
		}
		result += height

	}
	return result, nil
}

func widthAsChildren(node *LayoutNode, params interface{}) (Size, error) {
	result := emptySize
	for _, child := range node.Children {
		width, err := child.Width.await()
		if err != nil {
			return emptySize, err
		}
		result += width

	}
	return result, nil
}

func heightFill(node *LayoutNode, params interface{}) (Size, error) {
	result := emptySize

	parentHeight := emptySize
	var siblings []*LayoutNode
	var flowDirection childFlowDirection

	// first attempt to use the parent node
	if node.Parent != nil {
		parentDrawRect, err := node.Parent.getDrawRect()
		if err != nil {
			return emptySize, err
		}

		parentHeight = parentDrawRect.height
		siblings = node.Parent.Children
		flowDirection = node.Parent.ChildFlowDirection
	} else if node.Page != nil {
		parentDrawRect := node.Page.getDrawRect()
		parentHeight = parentDrawRect.height
		siblings = node.Page.Children
		flowDirection = FlowVertical
	}

	if parentHeight == emptySize {
		return result, invariantViolation("orphan node: parent is nil or has height of 0")
	}

	if flowDirection == FlowHorizontal {
		return heightPercentage(node, 100.0)
	}

	siblingHeights := emptySize
	for _, siblingNode := range siblings {
		if siblingNode != node {
			hR, e := siblingNode.getBoundingRect()
			if e != nil {
				return emptySize, e
			}
			siblingHeights += hR.height
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
		parentDrawRect, err := node.Parent.getDrawRect()
		if err != nil {
			return emptySize, err
		}
		parentWidth = parentDrawRect.width
		siblings = node.Parent.Children
		flowDirection = node.Parent.ChildFlowDirection
	} else if node.Page != nil {
		parentDrawRect := node.Page.getDrawRect()
		parentWidth = parentDrawRect.width
		siblings = node.Page.Children
		flowDirection = FlowVertical
	}

	if parentWidth == emptySize {
		return result, invariantViolation("orphan node: parent is nil or has width of 0")
	}

	if flowDirection == FlowVertical {
		return widthPercentage(node, 100.0)
	}

	siblingWidths := emptySize
	for _, siblingNode := range siblings {
		if siblingNode != node {
			wR, e := siblingNode.getBoundingRect()
			if e != nil {
				return emptySize, e
			}
			siblingWidths += wR.width
		}
	}

	result = parentWidth - siblingWidths
	result -= node.Margin.left + node.Margin.right

	// return the difference between the parent Width and all the sibling Widths
	return result, nil
}

func widthPercentage(node *LayoutNode, params interface{}) (Size, error) {
	var parentDrawRect Rect
	result := emptySize

	if node.Parent != nil {
		r, err := node.Parent.getDrawRect()
		if err != nil {
			return result, err
		}

		parentDrawRect = r
	} else if node.Page != nil {
		parentDrawRect = node.Page.getDrawRect()
	}

	if parentDrawRect.width == 0 {
		return result, invariantViolation("orphan node: parent is nil or has width of 0")
	}

	percentage := params.(Size)
	result = parentDrawRect.width * (percentage / 100)

	return result, nil
}

func heightPercentage(node *LayoutNode, params interface{}) (Size, error) {
	var parentDrawRect Rect
	result := emptySize

	if node.Parent != nil {
		r, err := node.Parent.getDrawRect()
		if err != nil {
			return result, err
		}
		parentDrawRect = r
	} else if node.Page != nil {
		parentDrawRect = node.Page.getDrawRect()
	}

	if parentDrawRect.height == 0 {
		return result, invariantViolation("orphan node: parent is nil or has height of 0")
	}

	percentage := params.(Size)
	result = parentDrawRect.height * (percentage / 100)

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
