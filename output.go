package panyl

import "context"

// Output receives each processed line.
type Output interface {
	OnItem(ctx context.Context, item *Item) (cont bool)
	OnFlush(ctx context.Context)
	OnClose(ctx context.Context)
}
