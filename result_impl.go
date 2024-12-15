package panyl

import "context"

// ProcessResultFunc is a helper to use ProcessResult as a function
type ProcessResultFunc func(item *Item)

func (pr ProcessResultFunc) OnResult(ctx context.Context, item *Item) bool {
	pr(item)
	return true
}

func (pr ProcessResultFunc) OnFlush(ctx context.Context) {}

func (pr ProcessResultFunc) OnClose(ctx context.Context) {}

// ProcessResultArray is a ProcessResult that accumulates in an array
type ProcessResultArray struct {
	List []*Item
}

func (pr *ProcessResultArray) OnResult(ctx context.Context, item *Item) bool {
	pr.List = append(pr.List, item)
	return true
}

func (pr ProcessResultArray) OnFlush(ctx context.Context) {}

func (pr ProcessResultArray) OnClose(ctx context.Context) {}

// ProcessResultNull ignores the result and do nothing
type ProcessResultNull struct {
}

func (pr *ProcessResultNull) OnResult(ctx context.Context, item *Item) bool {
	return true
}

func (pr ProcessResultNull) OnFlush(ctx context.Context) {}

func (pr ProcessResultNull) OnClose(ctx context.Context) {}
