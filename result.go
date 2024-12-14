package panyl

import "context"

// ProcessResult receives the result of each processed line
type ProcessResult interface {
	OnResult(ctx context.Context, p *Process) (cont bool)
	OnFlush(ctx context.Context)
	OnClose(ctx context.Context)
}

// ProcessResultFunc is a helper to use ProcessResult as a function
type ProcessResultFunc func(p *Process)

func (pr ProcessResultFunc) OnResult(ctx context.Context, p *Process) bool {
	pr(p)
	return true
}

func (pr ProcessResultFunc) OnFlush(ctx context.Context) {}

func (pr ProcessResultFunc) OnClose(ctx context.Context) {}

// ProcessResultArray is a ProcessResult that accumulates in an array
type ProcessResultArray struct {
	List []*Process
}

func (pr *ProcessResultArray) OnResult(ctx context.Context, p *Process) bool {
	pr.List = append(pr.List, p)
	return true
}

func (pr ProcessResultArray) OnFlush(ctx context.Context) {}

func (pr ProcessResultArray) OnClose(ctx context.Context) {}

// ProcessResultNull ignores the result and do nothing
type ProcessResultNull struct {
}

func (pr *ProcessResultNull) OnResult(ctx context.Context, p *Process) bool {
	return true
}

func (pr ProcessResultNull) OnFlush(ctx context.Context) {}

func (pr ProcessResultNull) OnClose(ctx context.Context) {}
