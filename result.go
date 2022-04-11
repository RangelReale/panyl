package panyl

// ProcessResult receives the result of each processed line
type ProcessResult interface {
	OnResult(p *Process) (cont bool)
	OnFlush()
}

// ProcessResultFunc is a helper to use ProcessResult as a function
type ProcessResultFunc func(p *Process)

func (pr ProcessResultFunc) OnResult(p *Process) bool {
	pr(p)
	return true
}

func (pr ProcessResultFunc) OnFlush() {}

// ProcessResultArray is a ProcessResult that accumulates in an array
type ProcessResultArray struct {
	List []*Process
}

func (pr *ProcessResultArray) OnResult(p *Process) bool {
	pr.List = append(pr.List, p)
	return true
}

func (pr ProcessResultArray) OnFlush() {}

// ProcessResultNull ignores the result and do nothing
type ProcessResultNull struct {
}

func (pr *ProcessResultNull) OnResult(p *Process) bool {
	return true
}

func (pr ProcessResultNull) OnFlush() {}
