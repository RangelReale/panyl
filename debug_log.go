package panyl

import (
	"context"
)

// DebugLog allows debugging each step of the processing
type DebugLog interface {
	// LogSourceLine receives one receiced raw log line after running PluginClean and strings.TrimSpace.
	LogSourceLine(ctx context.Context, n int, line, rawLine string)
	// LogItem receives one Item right before it is sent to ProcessResult.
	LogItem(ctx context.Context, item *Item)
}
