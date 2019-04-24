package parser

import "github.com/ChrisTrenkamp/goxpath/lexer"

//NodeType enumerations
const (
	Empty lexer.XItemType = ""
)

//Node builds an AST tree for operating on XPath expressions
type Node struct {
	Val    lexer.XItem
	Left   *Node
	Right  *Node
	Parent *Node
	next   *Node
}

var beginPathType = map[lexer.XItemType]bool{
	lexer.XItemAbsLocPath:     true,
	lexer.XItemAbbrAbsLocPath: true,
	lexer.XItemAbbrRelLocPath: true,
	lexer.XItemRelLocPath:     true,
	lexer.XItemFunction:       true,
}

func (n *Node) add(i lexer.XItem) {
	if n.Val.Typ == Empty {
		n.Val = i
	} else if n.Left == nil {
		n.Left = &Node{Val: n.Val, Parent: n}
		n.Val = i
	} else if beginPathType[n.Val.Typ] {
		next := &Node{Val: n.Val, Left: n.Left, Parent: n}
		n.Left = next
		n.Val = i
	} else if n.Right == nil {
		n.Right = &Node{Val: i, Parent: n}
	} else {
		next := &Node{Val: n.Val, Left: n.Left, Right: n.Right, Parent: n}
		n.Left, n.Right = next, nil
		n.Val = i
	}
	n.next = n
}

func (n *Node) push(i lexer.XItem) {
	if n.Left == nil {
		n.Left = &Node{Val: i, Parent: n}
		n.next = n.Left
	} else if n.Right == nil {
		n.Right = &Node{Val: i, Parent: n}
		n.next = n.Right
	} else {
		next := &Node{Val: i, Left: n.Right, Parent: n}
		n.Right = next
		n.next = n.Right
	}
}

func (n *Node) pushNotEmpty(i lexer.XItem) {
	if n.Val.Typ == Empty {
		n.add(i)
	} else {
		n.push(i)
	}
}

/*
func (n *Node) prettyPrint(depth, width int) {
	nodes := []*Node{}
	n.getLine(depth, &nodes)
	fmt.Printf("%*s", (width-depth)*2, "")
	toggle := true
	if len(nodes) > 1 {
		for _, i := range nodes {
			if i != nil {
				if toggle {
					fmt.Print("/   ")
				} else {
					fmt.Print("\\   ")
				}
			}
			toggle = !toggle
		}
		fmt.Println()
		fmt.Printf("%*s", (width-depth)*2, "")
	}
	for _, i := range nodes {
		if i != nil {
			fmt.Print(i.Val.Val, "   ")
		}
	}
	fmt.Println()
}

func (n *Node) getLine(depth int, ret *[]*Node) {
	if depth <= 0 && n != nil {
		*ret = append(*ret, n)
		return
	}
	if n.Left != nil {
		n.Left.getLine(depth-1, ret)
	} else if depth-1 <= 0 {
		*ret = append(*ret, nil)
	}
	if n.Right != nil {
		n.Right.getLine(depth-1, ret)
	} else if depth-1 <= 0 {
		*ret = append(*ret, nil)
	}
}
*/
