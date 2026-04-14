package hir

import "fmt"

// MetadataKey identifies a metadata field that passes read or write.
type MetadataKey string

const (
	MDType      MetadataKey = "type"      // TypeId on each node
	MDOwnership MetadataKey = "ownership" // OwnershipKind on each node
	MDDiverges  MetadataKey = "diverges"  // divergence flag
	MDLiveAfter MetadataKey = "live_after" // liveness slots
	MDDestroy   MetadataKey = "destroy"    // destruction flag
)

// PassDecl declares one compiler pass with its metadata contract.
type PassDecl struct {
	Name   string
	Reads  []MetadataKey // metadata this pass consumes
	Writes []MetadataKey // metadata this pass produces
}

// PassManifest is the ordered list of compiler passes with their metadata
// contracts. It validates that every read is satisfied by a prior write.
type PassManifest struct {
	Passes []PassDecl
}

// NewPassManifest creates a manifest and validates ordering.
// Returns the manifest and any validation errors.
func NewPassManifest(passes []PassDecl) (*PassManifest, []error) {
	m := &PassManifest{Passes: passes}
	errs := m.validate()
	return m, errs
}

func (m *PassManifest) validate() []error {
	available := make(map[MetadataKey]string) // key → producing pass name
	var errs []error
	for _, p := range m.Passes {
		// Check that all reads are available.
		for _, r := range p.Reads {
			if _, ok := available[r]; !ok {
				errs = append(errs, fmt.Errorf(
					"pass %q reads %q but no prior pass writes it", p.Name, r))
			}
		}
		// Register writes.
		for _, w := range p.Writes {
			available[w] = p.Name
		}
	}
	return errs
}

// DefaultPasses returns the standard Fuse compiler pass ordering.
func DefaultPasses() []PassDecl {
	return []PassDecl{
		{
			Name:   "resolve",
			Reads:  nil,
			Writes: []MetadataKey{MDType},
		},
		{
			Name:   "check",
			Reads:  []MetadataKey{MDType},
			Writes: []MetadataKey{MDType, MDOwnership, MDDiverges},
		},
		{
			Name:   "liveness",
			Reads:  []MetadataKey{MDType, MDOwnership},
			Writes: []MetadataKey{MDLiveAfter, MDDestroy},
		},
		{
			Name:   "lower",
			Reads:  []MetadataKey{MDType, MDOwnership, MDDiverges, MDLiveAfter, MDDestroy},
			Writes: nil, // produces MIR, not HIR metadata
		},
	}
}
