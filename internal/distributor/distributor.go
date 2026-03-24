package distributor

import "fmt"

// Target identifies a MongoDB database + collection pair.
type Target struct {
	DB         string
	Collection string
}

// Distributor decides which Target to write to given a document index.
type Distributor interface {
	// Targets returns the full set of write targets in order.
	Targets() []Target
	// Next returns the Target for document at the given index.
	Next(index int64) Target
}

// buildTargets creates a flat list of db×collection targets.
func buildTargets(numDBs, numCols int) []Target {
	targets := make([]Target, 0, numDBs*numCols)
	for d := 0; d < numDBs; d++ {
		for c := 0; c < numCols; c++ {
			targets = append(targets, Target{
				DB:         fmt.Sprintf("db_%d", d),
				Collection: fmt.Sprintf("col_%d", c),
			})
		}
	}
	return targets
}
