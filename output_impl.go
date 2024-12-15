package panyl

import "context"

// OutputFunc is a helper to use Output as a function
type OutputFunc func(item *Item)

func (pr OutputFunc) OnItem(ctx context.Context, item *Item) bool {
	pr(item)
	return true
}

func (pr OutputFunc) OnFlush(ctx context.Context) {}

func (pr OutputFunc) OnClose(ctx context.Context) {}

// OutputArray is a Output that accumulates in an array
type OutputArray struct {
	List []*Item
}

var _ Output = (*OutputArray)(nil)

func (pr *OutputArray) OnItem(ctx context.Context, item *Item) bool {
	pr.List = append(pr.List, item)
	return true
}

func (pr OutputArray) OnFlush(ctx context.Context) {}

func (pr OutputArray) OnClose(ctx context.Context) {}

// OutputNull ignores the item and do nothing
type OutputNull struct {
}

var _ Output = (*OutputNull)(nil)

func (pr *OutputNull) OnItem(ctx context.Context, item *Item) bool {
	return true
}

func (pr OutputNull) OnFlush(ctx context.Context) {}

func (pr OutputNull) OnClose(ctx context.Context) {}
