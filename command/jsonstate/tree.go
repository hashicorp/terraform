package jsonstate

import (
	"fmt"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"log"
)

type node struct {
	addr     addrs.ModuleInstance
	children map[string]*node
	parent   *node
	module   *states.Module
}

func newNode(addr addrs.ModuleInstance, parent *node, module *states.Module) *node {
	return &node{
		addr:     addr,
		children: make(map[string]*node),
		parent:   parent,
		module:   module,
	}
}

func buildModulesTree(s *states.State) (*node, error) {
	root, exists := s.Modules[addrs.RootModuleInstance.String()]
	if !exists {
		return nil, fmt.Errorf("root module does not exist")
	}
	tree := newNode(addrs.RootModuleInstance, nil, root)
	for _, module := range s.Modules {
		err := tree.add(s, module.Addr)
		if err != nil {
			return nil, fmt.Errorf("failed to build modules tree: %w", err)
		}
	}

	err := tree.prune()
	if err != nil {
		return nil, fmt.Errorf("failed to prune modules' tree: %w", err)
	}

	return tree, nil
}

func (n *node) add(s *states.State, addr addrs.ModuleInstance) error {
	parent := n
	for i := 0; i < len(addr); i++ {
		curAddr := addr[:i]
		if parent.addr.String() != curAddr.String() {
			log.Printf(
				"[ERROR] Failed to build path to node: addr=%v, curAddr=%v, parentAddr=%v",
				addr.String(),
				curAddr.String(),
				parent.addr.String(),
			)
			return fmt.Errorf("failed to build path")
		}

		nextAddr := curAddr.Child(addr[i].Name, addr[i].InstanceKey)
		next, exists := parent.children[nextAddr.String()]
		if !exists {
			module, moduleExists := s.Modules[nextAddr.String()]
			if !moduleExists {
				module = nil
			}
			next = newNode(nextAddr, parent, module)
			parent.children[nextAddr.String()] = next
		}

		parent = next
	}

	return nil
}

func (n *node) prune() error {
	for _, child := range n.children {
		err := child.prune()
		if err != nil {
			log.Printf("[ERROR] Failed to prune a child: child=%v, parent=%v", child, n)
			return err
		}
	}

	if n.module != nil {
		return nil
	}

	if n.parent == nil {
		log.Printf("[ERROR] Root module node has no real module: %v", n)
		return fmt.Errorf("failed to build module structure")
	}

	for _, child := range n.children {
		n.parent.children[child.addr.String()] = child
		child.parent = n.parent
		delete(n.parent.children, n.addr.String())
	}

	return nil
}
