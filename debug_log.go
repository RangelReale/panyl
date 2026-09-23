package panyl

import (
	"context"
)

// DebugLog allows debugging each step of the processing
type DebugLog interface {
	// LogSourceLine receives one log line after running PluginClean and strings.TrimSpace, and the raw line as it
	// was received (for *Item lines, the JSON encoding of Item.Data).
	LogSourceLine(ctx context.Context, n int, line, rawLine string)
	// LogItem receives one Item right before it is sent to Output.
	LogItem(ctx context.Context, item *Item)
}
