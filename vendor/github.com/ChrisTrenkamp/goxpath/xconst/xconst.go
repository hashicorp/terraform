package xconst

const (
	//AxisAncestor represents the "ancestor" axis
	AxisAncestor = "ancestor"
	//AxisAncestorOrSelf represents the "ancestor-or-self" axis
	AxisAncestorOrSelf = "ancestor-or-self"
	//AxisAttribute represents the "attribute" axis
	AxisAttribute = "attribute"
	//AxisChild represents the "child" axis
	AxisChild = "child"
	//AxisDescendent represents the "descendant" axis
	AxisDescendent = "descendant"
	//AxisDescendentOrSelf represents the "descendant-or-self" axis
	AxisDescendentOrSelf = "descendant-or-self"
	//AxisFollowing represents the "following" axis
	AxisFollowing = "following"
	//AxisFollowingSibling represents the "following-sibling" axis
	AxisFollowingSibling = "following-sibling"
	//AxisNamespace represents the "namespace" axis
	AxisNamespace = "namespace"
	//AxisParent represents the "parent" axis
	AxisParent = "parent"
	//AxisPreceding represents the "preceding" axis
	AxisPreceding = "preceding"
	//AxisPrecedingSibling represents the "preceding-sibling" axis
	AxisPrecedingSibling = "preceding-sibling"
	//AxisSelf represents the "self" axis
	AxisSelf = "self"
)

//AxisNames is all the possible Axis identifiers wrapped in an array for convenience
var AxisNames = []string{
	AxisAncestor,
	AxisAncestorOrSelf,
	AxisAttribute,
	AxisChild,
	AxisDescendent,
	AxisDescendentOrSelf,
	AxisFollowing,
	AxisFollowingSibling,
	AxisNamespace,
	AxisParent,
	AxisPreceding,
	AxisPrecedingSibling,
	AxisSelf,
}

const (
	//NodeTypeComment represents the "comment" node test
	NodeTypeComment = "comment"
	//NodeTypeText represents the "text" node test
	NodeTypeText = "text"
	//NodeTypeProcInst represents the "processing-instruction" node test
	NodeTypeProcInst = "processing-instruction"
	//NodeTypeNode represents the "node" node test
	NodeTypeNode = "node"
)

//NodeTypes is all the possible node tests wrapped in an array for convenience
var NodeTypes = []string{
	NodeTypeComment,
	NodeTypeText,
	NodeTypeProcInst,
	NodeTypeNode,
}
