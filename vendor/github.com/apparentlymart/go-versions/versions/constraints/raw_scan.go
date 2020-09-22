// line 1 "raw_scan.rl"
// This file is generated from raw_scan.rl. DO NOT EDIT.

// line 5 "raw_scan.rl"

package constraints

// line 12 "raw_scan.go"
var _scan_eof_actions []byte = []byte{
	0, 1, 1, 7, 9, 9, 9, 11,
	14, 15, 11,
}

const scan_start int = 1
const scan_first_final int = 7
const scan_error int = 0

const scan_en_main int = 1

// line 11 "raw_scan.rl"

func scanConstraint(data string) (rawConstraint, string) {
	var constraint rawConstraint
	var numIdx int
	var extra string

	// Ragel state
	p := 0          // "Pointer" into data
	pe := len(data) // End-of-data "pointer"
	cs := 0         // constraint state (will be initialized by ragel-generated code)
	ts := 0
	te := 0
	eof := pe

	// Keep Go compiler happy even if generated code doesn't use these
	_ = ts
	_ = te
	_ = eof

	// line 47 "raw_scan.go"
	{
		cs = scan_start
	}

	// line 52 "raw_scan.go"
	{
		if p == pe {
			goto _test_eof
		}
		if cs == 0 {
			goto _out
		}
	_resume:
		switch cs {
		case 1:
			switch data[p] {
			case 32:
				goto tr1
			case 42:
				goto tr2
			case 46:
				goto tr3
			case 88:
				goto tr2
			case 120:
				goto tr2
			}
			switch {
			case data[p] < 48:
				if 9 <= data[p] && data[p] <= 13 {
					goto tr1
				}
			case data[p] > 57:
				switch {
				case data[p] > 90:
					if 97 <= data[p] && data[p] <= 122 {
						goto tr3
					}
				case data[p] >= 65:
					goto tr3
				}
			default:
				goto tr4
			}
			goto tr0
		case 2:
			switch data[p] {
			case 32:
				goto tr6
			case 42:
				goto tr7
			case 46:
				goto tr3
			case 88:
				goto tr7
			case 120:
				goto tr7
			}
			switch {
			case data[p] < 48:
				if 9 <= data[p] && data[p] <= 13 {
					goto tr6
				}
			case data[p] > 57:
				switch {
				case data[p] > 90:
					if 97 <= data[p] && data[p] <= 122 {
						goto tr3
					}
				case data[p] >= 65:
					goto tr3
				}
			default:
				goto tr8
			}
			goto tr5
		case 3:
			switch data[p] {
			case 32:
				goto tr10
			case 42:
				goto tr11
			case 88:
				goto tr11
			case 120:
				goto tr11
			}
			switch {
			case data[p] > 13:
				if 48 <= data[p] && data[p] <= 57 {
					goto tr12
				}
			case data[p] >= 9:
				goto tr10
			}
			goto tr9
		case 0:
			goto _out
		case 7:
			switch data[p] {
			case 43:
				goto tr19
			case 45:
				goto tr20
			case 46:
				goto tr21
			}
			goto tr18
		case 4:
			switch {
			case data[p] < 48:
				if 45 <= data[p] && data[p] <= 46 {
					goto tr14
				}
			case data[p] > 57:
				switch {
				case data[p] > 90:
					if 97 <= data[p] && data[p] <= 122 {
						goto tr14
					}
				case data[p] >= 65:
					goto tr14
				}
			default:
				goto tr14
			}
			goto tr13
		case 8:
			switch {
			case data[p] < 48:
				if 45 <= data[p] && data[p] <= 46 {
					goto tr14
				}
			case data[p] > 57:
				switch {
				case data[p] > 90:
					if 97 <= data[p] && data[p] <= 122 {
						goto tr14
					}
				case data[p] >= 65:
					goto tr14
				}
			default:
				goto tr14
			}
			goto tr22
		case 5:
			switch {
			case data[p] < 48:
				if 45 <= data[p] && data[p] <= 46 {
					goto tr15
				}
			case data[p] > 57:
				switch {
				case data[p] > 90:
					if 97 <= data[p] && data[p] <= 122 {
						goto tr15
					}
				case data[p] >= 65:
					goto tr15
				}
			default:
				goto tr15
			}
			goto tr13
		case 9:
			if data[p] == 43 {
				goto tr24
			}
			switch {
			case data[p] < 48:
				if 45 <= data[p] && data[p] <= 46 {
					goto tr15
				}
			case data[p] > 57:
				switch {
				case data[p] > 90:
					if 97 <= data[p] && data[p] <= 122 {
						goto tr15
					}
				case data[p] >= 65:
					goto tr15
				}
			default:
				goto tr15
			}
			goto tr23
		case 6:
			switch data[p] {
			case 42:
				goto tr16
			case 88:
				goto tr16
			case 120:
				goto tr16
			}
			if 48 <= data[p] && data[p] <= 57 {
				goto tr17
			}
			goto tr13
		case 10:
			switch data[p] {
			case 43:
				goto tr19
			case 45:
				goto tr20
			case 46:
				goto tr21
			}
			if 48 <= data[p] && data[p] <= 57 {
				goto tr25
			}
			goto tr18
		}

	tr3:
		cs = 0
		goto f0
	tr9:
		cs = 0
		goto f6
	tr13:
		cs = 0
		goto f8
	tr18:
		cs = 0
		goto f10
	tr22:
		cs = 0
		goto f13
	tr23:
		cs = 0
		goto f14
	tr5:
		cs = 2
		goto _again
	tr0:
		cs = 2
		goto f1
	tr10:
		cs = 3
		goto _again
	tr1:
		cs = 3
		goto f2
	tr6:
		cs = 3
		goto f4
	tr19:
		cs = 4
		goto f11
	tr24:
		cs = 4
		goto f15
	tr20:
		cs = 5
		goto f11
	tr21:
		cs = 6
		goto f12
	tr2:
		cs = 7
		goto f3
	tr7:
		cs = 7
		goto f5
	tr11:
		cs = 7
		goto f7
	tr16:
		cs = 7
		goto f9
	tr14:
		cs = 8
		goto _again
	tr15:
		cs = 9
		goto _again
	tr25:
		cs = 10
		goto _again
	tr4:
		cs = 10
		goto f3
	tr8:
		cs = 10
		goto f5
	tr12:
		cs = 10
		goto f7
	tr17:
		cs = 10
		goto f9

	f9:
		// line 38 "raw_scan.rl"

		ts = p

		goto _again
	f12:
		// line 52 "raw_scan.rl"

		te = p
		constraint.numCt++
		if numIdx < len(constraint.nums) {
			constraint.nums[numIdx] = data[ts:p]
			numIdx++
		}

		goto _again
	f8:
		// line 71 "raw_scan.rl"

		extra = data[p:]

		goto _again
	f1:
		// line 33 "raw_scan.rl"

		numIdx = 0
		constraint = rawConstraint{}

		// line 38 "raw_scan.rl"

		ts = p

		goto _again
	f4:
		// line 42 "raw_scan.rl"

		te = p
		constraint.op = data[ts:p]

		// line 38 "raw_scan.rl"

		ts = p

		goto _again
	f7:
		// line 47 "raw_scan.rl"

		te = p
		constraint.sep = data[ts:p]

		// line 38 "raw_scan.rl"

		ts = p

		goto _again
	f6:
		// line 47 "raw_scan.rl"

		te = p
		constraint.sep = data[ts:p]

		// line 71 "raw_scan.rl"

		extra = data[p:]

		goto _again
	f11:
		// line 52 "raw_scan.rl"

		te = p
		constraint.numCt++
		if numIdx < len(constraint.nums) {
			constraint.nums[numIdx] = data[ts:p]
			numIdx++
		}

		// line 38 "raw_scan.rl"

		ts = p

		goto _again
	f10:
		// line 52 "raw_scan.rl"

		te = p
		constraint.numCt++
		if numIdx < len(constraint.nums) {
			constraint.nums[numIdx] = data[ts:p]
			numIdx++
		}

		// line 71 "raw_scan.rl"

		extra = data[p:]

		goto _again
	f15:
		// line 61 "raw_scan.rl"

		te = p
		constraint.pre = data[ts+1 : p]

		// line 38 "raw_scan.rl"

		ts = p

		goto _again
	f14:
		// line 61 "raw_scan.rl"

		te = p
		constraint.pre = data[ts+1 : p]

		// line 71 "raw_scan.rl"

		extra = data[p:]

		goto _again
	f13:
		// line 66 "raw_scan.rl"

		te = p
		constraint.meta = data[ts+1 : p]

		// line 71 "raw_scan.rl"

		extra = data[p:]

		goto _again
	f2:
		// line 33 "raw_scan.rl"

		numIdx = 0
		constraint = rawConstraint{}

		// line 38 "raw_scan.rl"

		ts = p

		// line 42 "raw_scan.rl"

		te = p
		constraint.op = data[ts:p]

		goto _again
	f5:
		// line 42 "raw_scan.rl"

		te = p
		constraint.op = data[ts:p]

		// line 38 "raw_scan.rl"

		ts = p

		// line 47 "raw_scan.rl"

		te = p
		constraint.sep = data[ts:p]

		goto _again
	f0:
		// line 42 "raw_scan.rl"

		te = p
		constraint.op = data[ts:p]

		// line 47 "raw_scan.rl"

		te = p
		constraint.sep = data[ts:p]

		// line 71 "raw_scan.rl"

		extra = data[p:]

		goto _again
	f3:
		// line 33 "raw_scan.rl"

		numIdx = 0
		constraint = rawConstraint{}

		// line 38 "raw_scan.rl"

		ts = p

		// line 42 "raw_scan.rl"

		te = p
		constraint.op = data[ts:p]

		// line 47 "raw_scan.rl"

		te = p
		constraint.sep = data[ts:p]

		goto _again

	_again:
		if cs == 0 {
			goto _out
		}
		if p++; p != pe {
			goto _resume
		}
	_test_eof:
		{
		}
		if p == eof {
			switch _scan_eof_actions[cs] {
			case 9:
				// line 71 "raw_scan.rl"

				extra = data[p:]

			case 7:
				// line 47 "raw_scan.rl"

				te = p
				constraint.sep = data[ts:p]

				// line 71 "raw_scan.rl"

				extra = data[p:]

			case 11:
				// line 52 "raw_scan.rl"

				te = p
				constraint.numCt++
				if numIdx < len(constraint.nums) {
					constraint.nums[numIdx] = data[ts:p]
					numIdx++
				}

				// line 71 "raw_scan.rl"

				extra = data[p:]

			case 15:
				// line 61 "raw_scan.rl"

				te = p
				constraint.pre = data[ts+1 : p]

				// line 71 "raw_scan.rl"

				extra = data[p:]

			case 14:
				// line 66 "raw_scan.rl"

				te = p
				constraint.meta = data[ts+1 : p]

				// line 71 "raw_scan.rl"

				extra = data[p:]

			case 1:
				// line 42 "raw_scan.rl"

				te = p
				constraint.op = data[ts:p]

				// line 47 "raw_scan.rl"

				te = p
				constraint.sep = data[ts:p]

				// line 71 "raw_scan.rl"

				extra = data[p:]

				// line 610 "raw_scan.go"
			}
		}

	_out:
		{
		}
	}

	// line 92 "raw_scan.rl"

	return constraint, extra
}
