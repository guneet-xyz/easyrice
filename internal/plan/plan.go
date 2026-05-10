package plan

// OpKind is the type of operation.
type OpKind int

const (
	OpCreate OpKind = iota
	OpRemove
)

// Op is a single planned filesystem operation.
type Op struct {
	Kind   OpKind
	Source string // source file in rice repo (empty for OpRemove)
	Target string // symlink path in $HOME
	IsDir  bool   // true if target is a directory symlink, false for file symlink
}

// Conflict describes a target path that cannot be created.
type Conflict struct {
	Target string
	Source string
	Reason string
	IsDir  bool // true if conflict is for a directory symlink, false for file symlink
}

// Plan describes a set of operations to be performed for one package.
type Plan struct {
	PackageName string
	Profile     string
	Ops         []Op
	Conflicts   []Conflict
}

// IsEmpty returns true if the plan has no operations and no conflicts.
func (p *Plan) IsEmpty() bool {
	return len(p.Ops) == 0 && len(p.Conflicts) == 0
}
