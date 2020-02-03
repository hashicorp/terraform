// This file is generated from raw_scan.rl. DO NOT EDIT.
%%{
	# (except you are actually in raw_scan.rl here, so edit away!)
	machine scan;
}%%

package constraints

%%{
	write data;
}%%

func scanConstraint(data string) (rawConstraint, string) {
	var constraint rawConstraint
	var numIdx int
	var extra string

	// Ragel state
	p := 0  // "Pointer" into data
	pe := len(data) // End-of-data "pointer"
	cs := 0 // constraint state (will be initialized by ragel-generated code)
	ts := 0
	te := 0
	eof := pe

	// Keep Go compiler happy even if generated code doesn't use these
	_ = ts
	_ = te
	_ = eof

	%%{

		action enterConstraint {
			numIdx = 0
			constraint = rawConstraint{}
		}

		action ts {
			ts = p
		}

		action finishOp {
			te = p
			constraint.op = data[ts:p]
		}

		action finishSep {
			te = p
			constraint.sep = data[ts:p]
		}

		action finishNum {
			te = p
			constraint.numCt++
			if numIdx < len(constraint.nums) {
				constraint.nums[numIdx] = data[ts:p]
				numIdx++
			}
		}

		action finishPre {
			te = p
			constraint.pre = data[ts+1:p]
		}

		action finishMeta {
			te = p
			constraint.meta = data[ts+1:p]
		}

		action finishExtra {
			extra = data[p:]
		}

		num = (digit+ | '*' | 'x' | 'X') >ts %finishNum %err(finishNum) %eof(finishNum);

		op = ((any - (digit | space | alpha | '.' | '*'))**) >ts %finishOp %err(finishOp) %eof(finishOp);
		likelyOp = ('^' | '>' | '<' | '-' | '~' | '!');
		sep = (space**) >ts %finishSep %err(finishSep) %eof(finishSep);
		nums = (num ('.' num)*);
		extraStr = (alnum | '.' | '-')+;
		pre = ('-' extraStr) >ts %finishPre %err(finishPre) %eof(finishPre);
		meta = ('+' extraStr) >ts %finishMeta %err(finishMeta) %eof(finishMeta);

		constraint = (op sep nums pre? meta?) >enterConstraint;

		main := (constraint) @/finishExtra %/finishExtra $!finishExtra;

		write init;
		write exec;

	}%%

	return constraint, extra
}
