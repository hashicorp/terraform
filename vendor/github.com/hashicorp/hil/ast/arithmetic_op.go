package ast

// ArithmeticOp is the operation to use for the math.
type ArithmeticOp int

const (
	ArithmeticOpInvalid ArithmeticOp = 0

	ArithmeticOpAdd ArithmeticOp = iota
	ArithmeticOpSub
	ArithmeticOpMul
	ArithmeticOpDiv
	ArithmeticOpMod

	ArithmeticOpLogicalAnd
	ArithmeticOpLogicalOr

	ArithmeticOpEqual
	ArithmeticOpNotEqual
	ArithmeticOpLessThan
	ArithmeticOpLessThanOrEqual
	ArithmeticOpGreaterThan
	ArithmeticOpGreaterThanOrEqual
)
