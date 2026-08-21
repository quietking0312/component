package mmongo

import "testing"

func TestOperatorConstants_HaveDollarPrefix(t *testing.T) {
	ops := []string{
		OpSet, OpSetOnInsert, OpUnset, OpRename, OpInc, OpMul, OpMin, OpMax,
		OpCurrentDate, OpPush, OpPop, OpPull, OpPullAll, OpAddToSet, OpEach,
		OpPosition, OpSlice, OpBit,
		OpEq, OpNe, OpGt, OpGte, OpLt, OpLte, OpIn, OpNin, OpExists, OpType,
		OpRegex, OpSize, OpAll, OpElemMatch, OpMod, OpText, OpWhere,
		OpAnd, OpOr, OpNot, OpNor,
		OpMatch, OpGroup, OpProject, OpSort, OpLimit, OpSkip, OpUnwind,
		OpLookup, OpGraphLookup, OpAddFields, OpReplaceRoot, OpFacet, OpBucket,
		OpSortByCount, OpCount, OpSample, OpOut, OpMerge,
	}

	seen := make(map[string]bool, len(ops))
	for _, op := range ops {
		if len(op) == 0 || op[0] != '$' {
			t.Errorf("operator %q does not start with '$'", op)
		}
		if seen[op] {
			t.Errorf("duplicate operator constant value %q", op)
		}
		seen[op] = true
	}
}
