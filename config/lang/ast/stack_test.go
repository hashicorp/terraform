package ast

import (
	"reflect"
	"testing"
)

func TestStack(t *testing.T) {
	var s Stack
	if s.Len() != 0 {
		t.Fatalf("bad: %d", s.Len())
	}

	n := &LiteralNode{Value: 42}
	s.Push(n)

	if s.Len() != 1 {
		t.Fatalf("bad: %d", s.Len())
	}

	actual := s.Pop()
	if !reflect.DeepEqual(actual, n) {
		t.Fatalf("bad: %#v", actual)
	}

	if s.Len() != 0 {
		t.Fatalf("bad: %d", s.Len())
	}
}

func TestStack_reset(t *testing.T) {
	var s Stack

	n := &LiteralNode{Value: 42}
	s.Push(n)

	if s.Len() != 1 {
		t.Fatalf("bad: %d", s.Len())
	}

	s.Reset()

	if s.Len() != 0 {
		t.Fatalf("bad: %d", s.Len())
	}
}
