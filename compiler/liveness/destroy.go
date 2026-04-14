package liveness

import (
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/hir"
)

// InsertDestroyFlags walks the liveness result and marks the last-use nodes
// with DestroyEnd=true. This tells later passes where deterministic
// destruction should occur.
//
// Only owned-value locals are candidates for destruction. Borrows (ref, mutref)
// do not own the value and are not destroyed at last use.
func InsertDestroyFlags(fn *hir.Function, result *LivenessResult) {
	if result == nil {
		return
	}

	// Build a set of slot indices for owned locals.
	ownedSlots := make(map[int]bool)
	for _, slot := range result.Slots {
		if slot.Ownership == hir.OwnerValue || slot.Ownership == hir.OwnerOwned {
			ownedSlots[slot.Slot] = true
		}
	}

	// For each variable with a last use, check if it's an owned local.
	for varName, lastNode := range result.LastUses {
		slotIdx, ok := slotIndex(result.Slots, varName)
		if !ok {
			continue
		}
		if ownedSlots[slotIdx] {
			lastNode.Meta().DestroyEnd = true
		}
	}
}

func slotIndex(slots []VarSlot, name string) (int, bool) {
	for _, s := range slots {
		if s.Name == name {
			return s.Slot, true
		}
	}
	return 0, false
}

// RunAll executes the complete ownership + liveness + destruction pipeline
// on a single function. This is the primary entry point for Wave 06.
func RunAll(fn *hir.Function) (*LivenessResult, []diagnostics.Diagnostic) {
	// Phase 1: Ownership analysis
	oa := &OwnershipAnalyzer{}
	oa.AnalyzeOwnership(fn)

	// Phase 2: Liveness computation (exactly once, Rule 3.8)
	result := ComputeLiveness(fn)

	// Phase 3: Destruction flags
	InsertDestroyFlags(fn, result)

	return result, oa.Errors
}
