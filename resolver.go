package docspec

import (
	"fmt"
)

/* Functions for walking the document tree and resolving x, y, width & height values */

type resolverCursor struct {
	x Size
	y Size
}

// recursively sets the positions for child trees inside a node.  Note that in
// this ideal little world we can just walk the tree and increment a cursor
// without worrying about page breaks, since we only allow page breaks over top
// level elements.  This function assumes that the cursor is currently
// positioned at the top left corner of the parent node's (the node argument's)
// render rect
func recursiveSetPositions(node *LayoutNode, cursor *resolverCursor) error {

	// if the node has no children, this function has nothing to do
	if len(node.Children) == 0 {
		return nil
	}

	startingX := cursor.x
	startingY := cursor.y

	firstChildNode := node.Children[0]
	parentDrawRect, err := node.getDrawRect()

	if err != nil {
		return err
	}

	// for every possible permutation of flowDirection and childAlignment, find
	// the starting X and Y position of the first child.  Once this is found,
	// we can simply iterate along the other children, and use width and height
	// to increment the cursor.
	childBoundingRect, err := firstChildNode.getBoundingRect()
	if err != nil {
		return err
	}

	// measure the full column/row width and height for below calculations
	fullChildrenWidth := 0.0
	fullChildrenHeight := 0.0

	for _, child := range node.Children {
		cbr, err := child.getBoundingRect()

		if err != nil {
			return err
		}

		fullChildrenHeight += cbr.height
		fullChildrenWidth += cbr.width
	}

	// move the cursor to the "start" position of the parent's draw rect -- the
	// top left corner.  All calculations below will use this point as "0,0" of
	// the parent's draw rect.
	cursor.x += node.Padding.left
	cursor.y += node.Padding.top

	if node.ChildFlowDirection == FlowVertical {
		// the column of children will only be one child across
		switch node.ChildAlignment.Horizontal {
		case Start:
			// we're starting at the top left corner and going down, so we
			// don't have any measurement to do, besides taking into account
			// the child's margin.  We don't have to take this into account in
			// the other cases because we use the child's bounding rect, which
			// already takes into account margin.
			cursor.x += firstChildNode.Margin.left
			break
		case End:
			// we're starting at the right edge and going down, so to get
			// the x we need to move the column of children as far over the
			// parent's draw rect will allow
			cursor.x += (parentDrawRect.width - childBoundingRect.width)
			break
		case Center:
			// we're starting at the top, but the middle of the child node has
			// to be equidistant from both sides of the parent draw rect
			childWidth := childBoundingRect.width
			midParentWidth := parentDrawRect.width / 2
			distanceToMove := (midParentWidth - (childWidth / 2))
			cursor.x += distanceToMove
			break
		}

		switch node.ChildAlignment.Vertical {
		case Start:
			// see "Start" documentation for horizontal above
			cursor.y += firstChildNode.Margin.top
			break
		case End:
			// we're starting at the bottom edge, so to get the y we need to
			// move the column of children as far down as we can.  This
			// requires measuing the height of the column.
			diff := parentDrawRect.height - fullChildrenHeight
			cursor.y += diff
			break
		case Center:
			// the middle of the column of children must be equidistant from
			// the top and bottom of the parent's draw rect
			diff := (parentDrawRect.height - fullChildrenHeight) / 2
			cursor.y += diff
			break
		}
	} else if node.ChildFlowDirection == FlowHorizontal {
		// see comments above for thoughts.  Basically we're just inverting
		// everything.
		switch node.ChildAlignment.Horizontal {
		case Start:
			cursor.x += firstChildNode.Margin.left
			break
		case End:
			diff := parentDrawRect.width - fullChildrenWidth
			cursor.x += diff
			break
		case Center:
			diff := (parentDrawRect.width - fullChildrenWidth) / 2
			cursor.x += diff
			break
		}

		switch node.ChildAlignment.Vertical {
		case Start:
			cursor.y += firstChildNode.Margin.top
			break
		case End:
			childHeight := childBoundingRect.height
			diff := parentDrawRect.height - childHeight
			cursor.y += diff
			break
		case Center:
			diff := parentDrawRect.height - childBoundingRect.height
			distanceToMove := diff / 2
			cursor.y += distanceToMove
			break
		}
	} else {
		return fmt.Errorf("unhandled childFlowDirection '%+v' in resolver", node.ChildFlowDirection)
	}

	for idx, child := range node.Children {
		child.X = cursor.x
		child.Y = cursor.y

		err := recursiveSetPositions(child, cursor)
		if err != nil {
			return err
		}

		// now that we have set the x and y positions for the child and all of
		// its children, we need to move the cursor into the correct position
		// for the next sibling.

		childBoundingRect, err = child.getBoundingRect()

		// difference in width between this node and the next node in the list
		var diffW Size
		// difference in height between this node and the next node in the list
		var diffH Size

		if idx != len(node.Children)-1 {
			// there is a nextnode
			nextNode := node.Children[idx+1]
			nextBoundingRect, err := nextNode.getBoundingRect()
			if err != nil {
				return err
			}

			diffW = childBoundingRect.width - nextBoundingRect.width
			diffH = childBoundingRect.height - nextBoundingRect.height

			switch node.ChildFlowDirection {
			case FlowHorizontal:
				cursor.x += childBoundingRect.width + nextNode.Margin.left
				// depending on the child alignment along the oppositive axis,
				// we may need to adjust the opposite axis to account for
				// differently sized sibling nodes
				switch node.ChildAlignment.Vertical {
				case Start:
					break
				case End:
					cursor.y += diffH
					break
				case Center:
					cursor.y += diffH / 2
					break
				}
				break
			case FlowVertical:
				cursor.y += childBoundingRect.height + nextNode.Margin.top
				switch node.ChildAlignment.Horizontal {
				case Start:
					break
				case End:
					cursor.x += diffW
					break
				case Center:
					cursor.x += diffW / 2
					break
				}
				break
			}
		}
	}

	// Finally, reset the cursor to the position that it was at prior to the
	// function being called -- when we pass the cursor to children, we don't
	// want those children mutating the cursor's state when we move on to the
	// next sibling.  Think of the cursor as moving in two directions --
	// downward into the onion (where we want it to be mutated), and
	// side-to-side along a list of siblings (where we don't want it to be
	// mutated).  Resetting at the end of the function gives us the behavior we
	// want.

	cursor.x = startingX
	cursor.y = startingY

	return nil
}

// resolveNodeRectPositions iterates over the nodes in the document builder's
// node list with a cursor and resolve the X and Y positions
func resolveNodeRectPositions(d *DocumentBuilder) error {
	document := d.document

	currentPage := document.Children[0]
	cursor := resolverCursor{
		x: currentPage.Margin.left,
		y: currentPage.Margin.top,
	}

	for idx, node := range d.nodes {
		fmt.Printf("new node: x: %f, h: %f\n", cursor.x, cursor.y)
		node.X = cursor.x + node.Margin.left
		node.Y = cursor.y + node.Margin.top
		currentPage.addNode(node)

		nextCursor := cursor
		// prepare nextCursor position so that it's at the top left corner of
		// the parent node's RENDER rect, not the bounding rect
		nextCursor.x += node.Margin.left
		nextCursor.y += node.Margin.top
		err := recursiveSetPositions(node, &nextCursor)

		if err != nil {
			return err
		}

		nodeBoundingRect, err := node.getBoundingRect()
		nodeHeight := nodeBoundingRect.height

		if err != nil {
			return err
		}

		if nodeHeight > currentPage.Height {
			return fmt.Errorf("top level element had calculated height of %f, which is larger than page height of %f", nodeHeight, currentPage.Height)
		}

		// note that we are purposely not changing the X cursor at the top level.
		// The top level gets special treatment -- see recursiveSetPositions for
		// the "real" implementaiton
		cursor.y += nodeHeight

		if idx != len(d.nodes)-1 {
			// if we are not on the last node, get the height of the NEXT node,
			// and figure out if we need to split pages.
			nextHeight, err := d.nodes[idx+1].Height.await()
			if err != nil {
				return err
			}

			if cursor.y+nextHeight > (currentPage.getDrawRect().height - currentPage.Margin.bottom) {
				currentPage = d.document.addPage()
				cursor.x = currentPage.Margin.left
				cursor.y = currentPage.Margin.top
			}
		}

	}

	return nil
}
