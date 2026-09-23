package panyl

import "context"

// Output receives each processed line.
type Output interface {
	// OnItem receives one processed item. Return false to stop processing: no more items are sent, and the
	// output is still flushed and closed.
	OnItem(ctx context.Context, item *Item) (cont bool)
	OnFlush(ctx context.Context)
	OnClose(ctx context.Context)
}
