package exprstress

import (
	"math/rand"

	"github.com/zclconf/go-cty/cty"
)

type exprArithmetic struct {
	LHS      Expression
	RHS      Expression
	Operator string
}

var arithmeticOperators = []string{
	"+", "-", "*", "/", "%",
}

var moduloOperands = []Expression{
	&exprLiteral{
		Value: cty.Zero,
	},
	&exprFixed{
		`local.unknown.number`,
		Expected{
			Type: cty.Number,
			Mode: UnknownValue,
		},
	},
	&exprFixed{
		`local.sensitive.zero`,
		Expected{
			Type:          cty.Number,
			Mode:          SpecifiedValue,
			Sensitive:     true,
			SpecialNumber: NumberZero,
		},
	},
}

func generateArithmeticOperator(depth int) expressionGenerator {
	generateOperand := generateArithmeticOperand(depth + 1)
	return func(rand *rand.Rand) Expression {
		n := rand.Intn(len(arithmeticOperators))
		op := arithmeticOperators[n]

		if op == "%" {
			// Our model for the mathematical behavior of the modulo operator
			// is not fully correct, so for this one we intentionally
			// generate only a small set of variations that our model can
			// handle, with our testing focused mainly on the non-mathematical
			// behaviors of this operator that Terraform relies on, such as
			// passing through both unknown and sensitive values.
			lhs := moduloOperands[rand.Intn(len(moduloOperands))]
			rhs := moduloOperands[rand.Intn(len(moduloOperands))]
			return &exprArithmetic{
				LHS:      lhs,
				RHS:      rhs,
				Operator: op,
			}
		}

		lhs := generateOperand(rand)
		rhs := generateOperand(rand)

		if op == "+" || op == "-" {
			// We can't sum or subtract pairs of opposing infinities, but
			// thankfully infinities are only a small part of the possible
			// space of values and so we'll just brute-force generate new
			// lhs values until we get a non-infinite one, assuming that
			// it shouldn't take many attempts.
			lhsExpected := lhs.ExpectedResult()
			rhsExpected := rhs.ExpectedResult()
			if lhsExpected.SpecialNumber == NumberInfinity && rhsExpected.SpecialNumber == NumberInfinity {
				for {
					lhs = generateOperand(rand)
					lhsExpected = lhs.ExpectedResult()
					if lhsExpected.SpecialNumber != NumberInfinity {
						break
					}
				}
			}
		}

		return &exprArithmetic{
			LHS:      lhs,
			RHS:      rhs,
			Operator: op,
		}
	}
}

func (e exprArithmetic) BuildSource(w SourceWriter) {
	// To ensure we get the expected evaluation order without having to
	// analyze for precedence, we'll enclose both operands in parentheses.
	// This does mean that we're not actually testing precedence rules here,
	// but that's okay because HCL has its own tests for that.
	w.WriteString("(")
	e.LHS.BuildSource(w)
	w.WriteString(") ")
	w.WriteString(e.Operator)
	w.WriteString(" (")
	e.RHS.BuildSource(w)
	w.WriteString(")")
}

func (e exprArithmetic) ExpectedResult() Expected {
	mode := SpecifiedValue
	lhsExpect := e.LHS.ExpectedResult()
	rhsExpect := e.RHS.ExpectedResult()
	sensitive := lhsExpect.Sensitive || rhsExpect.Sensitive
	if lhsExpect.Mode == UnknownValue || rhsExpect.Mode == UnknownValue {
		mode = UnknownValue
		return Expected{
			Type:      cty.Number,
			Mode:      mode,
			Sensitive: sensitive,
		}
	}

	var specialNum SpecialNumber

	// Some of our operators behave differently for certain special numbers,
	// and most of the operators also have an identity value that causes
	// them to pass through the "specialness" of the other operand, so we'll
	// handle those here.
	switch e.Operator {
	case "/":
		switch {
		case rhsExpect.SpecialNumber == NumberZero:
			// Dividing by zero produces infinity.
			// (Dividing _zero_ by zero is an error, but it's the expression
			// generator's responsibility to avoid generating that invalid
			// combination.)
			specialNum = NumberInfinity
		case lhsExpect.SpecialNumber == NumberZero:
			// Dividing zero by anything produces zero.
			specialNum = NumberZero
		case rhsExpect.SpecialNumber == NumberOne:
			// One is the multiplicative identity, so any specialness of
			// LHS passes through.
			specialNum = lhsExpect.SpecialNumber
		}
	case "*":
		switch {
		case lhsExpect.SpecialNumber == NumberZero || rhsExpect.SpecialNumber == NumberZero:
			specialNum = NumberZero

		case lhsExpect.SpecialNumber == NumberInfinity || rhsExpect.SpecialNumber == NumberInfinity:
			specialNum = NumberInfinity

		// One is the multiplicative identity, so if either operand is one
		// then the specialness of the other operand passes through.
		case lhsExpect.SpecialNumber == NumberOne:
			specialNum = rhsExpect.SpecialNumber
		case rhsExpect.SpecialNumber == NumberOne:
			specialNum = lhsExpect.SpecialNumber
		}
	case "+":
		switch {
		case lhsExpect.SpecialNumber == NumberInfinity || rhsExpect.SpecialNumber == NumberInfinity:
			specialNum = NumberInfinity

		// Zero is the additive identity, so if either operand is one
		// then the specialness of the other operand passes through.
		case lhsExpect.SpecialNumber == NumberZero:
			specialNum = rhsExpect.SpecialNumber
		case rhsExpect.SpecialNumber == NumberZero:
			specialNum = lhsExpect.SpecialNumber
		}
	case "-":
		switch {
		case lhsExpect.SpecialNumber == NumberInfinity || rhsExpect.SpecialNumber == NumberInfinity:
			specialNum = NumberInfinity

		// If both operands are "equal" (have the same specialness) then the
		// result is special zero.
		// (Subtracting an infinity from itself is actually an error, but it's
		// the expression generator's responsibility to avoid generating that
		// invalid combination.)
		case lhsExpect.SpecialNumber == rhsExpect.SpecialNumber:
			specialNum = NumberZero

		// Zero is the additive identity, so any specialness of LHS passes
		// through.
		case rhsExpect.SpecialNumber == NumberZero:
			specialNum = rhsExpect.SpecialNumber

		}
	case "%":
		// At the moment we don't fully model modulo, because to do so requires
		// awareness of particular values aside from the limited set of
		// "special" numbers we're tracking. Therefore the expression generator
		// must take special care to only generate modulo expressions that
		// can't produce a "special" numeric result, aside from a few simple
		// cases we handle below.
		// The core arithemtic behaviors for modulo are HCL's responsibility
		// to test, so we accept that limitation here and focus our attention
		// on testing Terraform-specific concerns, such as that sensitive
		// values can pass through (represented by the "sensitive" variable
		// elsewhere in this function.)
		switch {
		case lhsExpect.SpecialNumber == NumberZero || rhsExpect.SpecialNumber == NumberOne:
			specialNum = NumberZero
		case rhsExpect.SpecialNumber == NumberZero:
			specialNum = lhsExpect.SpecialNumber
		}
	}

	return Expected{
		Type:          cty.Number,
		Mode:          mode,
		Sensitive:     sensitive,
		SpecialNumber: specialNum,
	}
}
