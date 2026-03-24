package distributor

// Uniform distributes documents evenly across all targets using round-robin.
type Uniform struct {
	targets []Target
	total   int64
}

// NewUniform creates a Uniform distributor for numDBs×numCols targets.
func NewUniform(numDBs, numCols int) *Uniform {
	t := buildTargets(numDBs, numCols)
	return &Uniform{targets: t, total: int64(len(t))}
}

// Targets returns the full ordered list of write targets.
func (u *Uniform) Targets() []Target {
	return u.targets
}

// Next returns the target at position (index % total).
func (u *Uniform) Next(index int64) Target {
	return u.targets[index%u.total]
}
