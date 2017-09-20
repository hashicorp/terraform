package xpath

import (
	"reflect"
)

type iterator interface {
	Current() NodeNavigator
}

// An XPath query interface.
type query interface {
	// Select traversing iterator returns a query matched node NodeNavigator.
	Select(iterator) NodeNavigator

	// Evaluate evaluates query and returns values of the current query.
	Evaluate(iterator) interface{}

	Clone() query
}

// contextQuery is returns current node on the iterator object query.
type contextQuery struct {
	count int
	Root  bool // Moving to root-level node in the current context iterator.
}

func (c *contextQuery) Select(t iterator) (n NodeNavigator) {
	if c.count == 0 {
		c.count++
		n = t.Current().Copy()
		if c.Root {
			n.MoveToRoot()
		}
	}
	return n
}

func (c *contextQuery) Evaluate(iterator) interface{} {
	c.count = 0
	return c
}

func (c *contextQuery) Clone() query {
	return &contextQuery{count: 0, Root: c.Root}
}

// ancestorQuery is an XPath ancestor node query.(ancestor::*|ancestor-self::*)
type ancestorQuery struct {
	iterator func() NodeNavigator

	Self      bool
	Input     query
	Predicate func(NodeNavigator) bool
}

func (a *ancestorQuery) Select(t iterator) NodeNavigator {
	for {
		if a.iterator == nil {
			node := a.Input.Select(t)
			if node == nil {
				return nil
			}
			first := true
			a.iterator = func() NodeNavigator {
				if first && a.Self {
					first = false
					if a.Predicate(node) {
						return node
					}
				}
				for node.MoveToParent() {
					if !a.Predicate(node) {
						break
					}
					return node
				}
				return nil
			}
		}

		if node := a.iterator(); node != nil {
			return node
		}
		a.iterator = nil
	}
}

func (a *ancestorQuery) Evaluate(t iterator) interface{} {
	a.Input.Evaluate(t)
	a.iterator = nil
	return a
}

func (a *ancestorQuery) Test(n NodeNavigator) bool {
	return a.Predicate(n)
}

func (a *ancestorQuery) Clone() query {
	return &ancestorQuery{Self: a.Self, Input: a.Input.Clone(), Predicate: a.Predicate}
}

// attributeQuery is an XPath attribute node query.(@*)
type attributeQuery struct {
	iterator func() NodeNavigator

	Input     query
	Predicate func(NodeNavigator) bool
}

func (a *attributeQuery) Select(t iterator) NodeNavigator {
	for {
		if a.iterator == nil {
			node := a.Input.Select(t)
			if node == nil {
				return nil
			}
			node = node.Copy()
			a.iterator = func() NodeNavigator {
				for {
					onAttr := node.MoveToNextAttribute()
					if !onAttr {
						return nil
					}
					if a.Predicate(node) {
						return node
					}
				}
			}
		}

		if node := a.iterator(); node != nil {
			return node
		}
		a.iterator = nil
	}
}

func (a *attributeQuery) Evaluate(t iterator) interface{} {
	a.Input.Evaluate(t)
	a.iterator = nil
	return a
}

func (a *attributeQuery) Test(n NodeNavigator) bool {
	return a.Predicate(n)
}

func (a *attributeQuery) Clone() query {
	return &attributeQuery{Input: a.Input.Clone(), Predicate: a.Predicate}
}

// childQuery is an XPath child node query.(child::*)
type childQuery struct {
	posit    int
	iterator func() NodeNavigator

	Input     query
	Predicate func(NodeNavigator) bool
}

func (c *childQuery) Select(t iterator) NodeNavigator {
	for {
		if c.iterator == nil {
			c.posit = 0
			node := c.Input.Select(t)
			if node == nil {
				return nil
			}
			node = node.Copy()
			first := true
			c.iterator = func() NodeNavigator {
				for {
					if (first && !node.MoveToChild()) || (!first && !node.MoveToNext()) {
						return nil
					}
					first = false
					if c.Predicate(node) {
						return node
					}
				}
			}
		}

		if node := c.iterator(); node != nil {
			c.posit++
			return node
		}
		c.iterator = nil
	}
}

func (c *childQuery) Evaluate(t iterator) interface{} {
	c.Input.Evaluate(t)
	c.iterator = nil
	return c
}

func (c *childQuery) Test(n NodeNavigator) bool {
	return c.Predicate(n)
}

func (c *childQuery) Clone() query {
	return &childQuery{Input: c.Input.Clone(), Predicate: c.Predicate}
}

// position returns a position of current NodeNavigator.
func (c *childQuery) position() int {
	return c.posit
}

// descendantQuery is an XPath descendant node query.(descendant::* | descendant-or-self::*)
type descendantQuery struct {
	iterator func() NodeNavigator
	posit    int

	Self      bool
	Input     query
	Predicate func(NodeNavigator) bool
}

func (d *descendantQuery) Select(t iterator) NodeNavigator {
	for {
		if d.iterator == nil {
			d.posit = 0
			node := d.Input.Select(t)
			if node == nil {
				return nil
			}
			node = node.Copy()
			level := 0
			first := true
			d.iterator = func() NodeNavigator {
				if first && d.Self {
					first = false
					if d.Predicate(node) {
						return node
					}
				}

				for {
					if node.MoveToChild() {
						level++
					} else {
						for {
							if level == 0 {
								return nil
							}
							if node.MoveToNext() {
								break
							}
							node.MoveToParent()
							level--
						}
					}
					if d.Predicate(node) {
						return node
					}
				}
			}
		}

		if node := d.iterator(); node != nil {
			d.posit++
			return node
		}
		d.iterator = nil
	}
}

func (d *descendantQuery) Evaluate(t iterator) interface{} {
	d.Input.Evaluate(t)
	d.iterator = nil
	return d
}

func (d *descendantQuery) Test(n NodeNavigator) bool {
	return d.Predicate(n)
}

// position returns a position of current NodeNavigator.
func (d *descendantQuery) position() int {
	return d.posit
}

func (d *descendantQuery) Clone() query {
	return &descendantQuery{Self: d.Self, Input: d.Input.Clone(), Predicate: d.Predicate}
}

// followingQuery is an XPath following node query.(following::*|following-sibling::*)
type followingQuery struct {
	iterator func() NodeNavigator

	Input     query
	Sibling   bool // The matching sibling node of current node.
	Predicate func(NodeNavigator) bool
}

func (f *followingQuery) Select(t iterator) NodeNavigator {
	for {
		if f.iterator == nil {
			node := f.Input.Select(t)
			if node == nil {
				return nil
			}
			node = node.Copy()
			if f.Sibling {
				f.iterator = func() NodeNavigator {
					for {
						if !node.MoveToNext() {
							return nil
						}
						if f.Predicate(node) {
							return node
						}
					}
				}
			} else {
				var q query // descendant query
				f.iterator = func() NodeNavigator {
					for {
						if q == nil {
							for !node.MoveToNext() {
								if !node.MoveToParent() {
									return nil
								}
							}
							q = &descendantQuery{
								Self:      true,
								Input:     &contextQuery{},
								Predicate: f.Predicate,
							}
							t.Current().MoveTo(node)
						}
						if node := q.Select(t); node != nil {
							return node
						}
						q = nil
					}
				}
			}
		}

		if node := f.iterator(); node != nil {
			return node
		}
		f.iterator = nil
	}
}

func (f *followingQuery) Evaluate(t iterator) interface{} {
	f.Input.Evaluate(t)
	return f
}

func (f *followingQuery) Test(n NodeNavigator) bool {
	return f.Predicate(n)
}

func (f *followingQuery) Clone() query {
	return &followingQuery{Input: f.Input.Clone(), Sibling: f.Sibling, Predicate: f.Predicate}
}

// precedingQuery is an XPath preceding node query.(preceding::*)
type precedingQuery struct {
	iterator  func() NodeNavigator
	Input     query
	Sibling   bool // The matching sibling node of current node.
	Predicate func(NodeNavigator) bool
}

func (p *precedingQuery) Select(t iterator) NodeNavigator {
	for {
		if p.iterator == nil {
			node := p.Input.Select(t)
			if node == nil {
				return nil
			}
			node = node.Copy()
			if p.Sibling {
				p.iterator = func() NodeNavigator {
					for {
						for !node.MoveToPrevious() {
							return nil
						}
						if p.Predicate(node) {
							return node
						}
					}
				}
			} else {
				var q query
				p.iterator = func() NodeNavigator {
					for {
						if q == nil {
							for !node.MoveToPrevious() {
								if !node.MoveToParent() {
									return nil
								}
							}
							q = &descendantQuery{
								Self:      true,
								Input:     &contextQuery{},
								Predicate: p.Predicate,
							}
							t.Current().MoveTo(node)
						}
						if node := q.Select(t); node != nil {
							return node
						}
						q = nil
					}
				}
			}
		}
		if node := p.iterator(); node != nil {
			return node
		}
		p.iterator = nil
	}
}

func (p *precedingQuery) Evaluate(t iterator) interface{} {
	p.Input.Evaluate(t)
	return p
}

func (p *precedingQuery) Test(n NodeNavigator) bool {
	return p.Predicate(n)
}

func (p *precedingQuery) Clone() query {
	return &precedingQuery{Input: p.Input.Clone(), Sibling: p.Sibling, Predicate: p.Predicate}
}

// parentQuery is an XPath parent node query.(parent::*)
type parentQuery struct {
	Input     query
	Predicate func(NodeNavigator) bool
}

func (p *parentQuery) Select(t iterator) NodeNavigator {
	for {
		node := p.Input.Select(t)
		if node == nil {
			return nil
		}
		node = node.Copy()
		if node.MoveToParent() && p.Predicate(node) {
			return node
		}
	}
}

func (p *parentQuery) Evaluate(t iterator) interface{} {
	p.Input.Evaluate(t)
	return p
}

func (p *parentQuery) Clone() query {
	return &parentQuery{Input: p.Input.Clone(), Predicate: p.Predicate}
}

func (p *parentQuery) Test(n NodeNavigator) bool {
	return p.Predicate(n)
}

// selfQuery is an Self node query.(self::*)
type selfQuery struct {
	Input     query
	Predicate func(NodeNavigator) bool
}

func (s *selfQuery) Select(t iterator) NodeNavigator {
	for {
		node := s.Input.Select(t)
		if node == nil {
			return nil
		}

		if s.Predicate(node) {
			return node
		}
	}
}

func (s *selfQuery) Evaluate(t iterator) interface{} {
	s.Input.Evaluate(t)
	return s
}

func (s *selfQuery) Test(n NodeNavigator) bool {
	return s.Predicate(n)
}

func (s *selfQuery) Clone() query {
	return &selfQuery{Input: s.Input.Clone(), Predicate: s.Predicate}
}

// filterQuery is an XPath query for predicate filter.
type filterQuery struct {
	Input     query
	Predicate query
}

func (f *filterQuery) do(t iterator) bool {
	val := reflect.ValueOf(f.Predicate.Evaluate(t))
	switch val.Kind() {
	case reflect.Bool:
		return val.Bool()
	case reflect.String:
		return len(val.String()) > 0
	case reflect.Float64:
		pt := float64(getNodePosition(f.Input))
		return int(val.Float()) == int(pt)
	default:
		if q, ok := f.Predicate.(query); ok {
			return q.Select(t) != nil
		}
	}
	return false
}

func (f *filterQuery) Select(t iterator) NodeNavigator {
	for {
		node := f.Input.Select(t)
		if node == nil {
			return node
		}
		node = node.Copy()
		//fmt.Println(node.LocalName())

		t.Current().MoveTo(node)
		if f.do(t) {
			return node
		}
	}
}

func (f *filterQuery) Evaluate(t iterator) interface{} {
	f.Input.Evaluate(t)
	return f
}

func (f *filterQuery) Clone() query {
	return &filterQuery{Input: f.Input.Clone(), Predicate: f.Predicate.Clone()}
}

// functionQuery is an XPath function that call a function to returns
// value of current NodeNavigator node.
type functionQuery struct {
	Input query                             // Node Set
	Func  func(query, iterator) interface{} // The xpath function.
}

func (f *functionQuery) Select(t iterator) NodeNavigator {
	return nil
}

// Evaluate call a specified function that will returns the
// following value type: number,string,boolean.
func (f *functionQuery) Evaluate(t iterator) interface{} {
	return f.Func(f.Input, t)
}

func (f *functionQuery) Clone() query {
	return &functionQuery{Input: f.Input.Clone(), Func: f.Func}
}

// constantQuery is an XPath constant operand.
type constantQuery struct {
	Val interface{}
}

func (c *constantQuery) Select(t iterator) NodeNavigator {
	return nil
}

func (c *constantQuery) Evaluate(t iterator) interface{} {
	return c.Val
}

func (c *constantQuery) Clone() query {
	return c
}

// logicalQuery is an XPath logical expression.
type logicalQuery struct {
	Left, Right query

	Do func(iterator, interface{}, interface{}) interface{}
}

func (l *logicalQuery) Select(t iterator) NodeNavigator {
	// When a XPath expr is logical expression.
	node := t.Current().Copy()
	val := l.Evaluate(t)
	switch val.(type) {
	case bool:
		if val.(bool) == true {
			return node
		}
	}
	return nil
}

func (l *logicalQuery) Evaluate(t iterator) interface{} {
	m := l.Left.Evaluate(t)
	n := l.Right.Evaluate(t)
	return l.Do(t, m, n)
}

func (l *logicalQuery) Clone() query {
	return &logicalQuery{Left: l.Left.Clone(), Right: l.Right.Clone(), Do: l.Do}
}

// numericQuery is an XPath numeric operator expression.
type numericQuery struct {
	Left, Right query

	Do func(interface{}, interface{}) interface{}
}

func (n *numericQuery) Select(t iterator) NodeNavigator {
	return nil
}

func (n *numericQuery) Evaluate(t iterator) interface{} {
	m := n.Left.Evaluate(t)
	k := n.Right.Evaluate(t)
	return n.Do(m, k)
}

func (n *numericQuery) Clone() query {
	return &numericQuery{Left: n.Left.Clone(), Right: n.Right.Clone(), Do: n.Do}
}

type booleanQuery struct {
	IsOr        bool
	Left, Right query
	iterator    func() NodeNavigator
}

func (b *booleanQuery) Select(t iterator) NodeNavigator {
	if b.iterator == nil {
		var list []NodeNavigator
		i := 0
		root := t.Current().Copy()
		if b.IsOr {
			for {
				node := b.Left.Select(t)
				if node == nil {
					break
				}
				node = node.Copy()
				list = append(list, node)
			}
			t.Current().MoveTo(root)
			for {
				node := b.Right.Select(t)
				if node == nil {
					break
				}
				node = node.Copy()
				list = append(list, node)
			}
		} else {
			var m []NodeNavigator
			var n []NodeNavigator
			for {
				node := b.Left.Select(t)
				if node == nil {
					break
				}
				node = node.Copy()
				list = append(m, node)
			}
			t.Current().MoveTo(root)
			for {
				node := b.Right.Select(t)
				if node == nil {
					break
				}
				node = node.Copy()
				list = append(n, node)
			}
			for _, k := range m {
				for _, j := range n {
					if k == j {
						list = append(list, k)
					}
				}
			}
		}

		b.iterator = func() NodeNavigator {
			if i >= len(list) {
				return nil
			}
			node := list[i]
			i++
			return node
		}
	}
	return b.iterator()
}

func (b *booleanQuery) Evaluate(t iterator) interface{} {
	m := b.Left.Evaluate(t)
	if m.(bool) == b.IsOr {
		return m
	}
	return b.Right.Evaluate(t)
}

func (b *booleanQuery) Clone() query {
	return &booleanQuery{IsOr: b.IsOr, Left: b.Left.Clone(), Right: b.Right.Clone()}
}

func getNodePosition(q query) int {
	type Position interface {
		position() int
	}
	if count, ok := q.(Position); ok {
		return count.position()
	}
	return 1
}
