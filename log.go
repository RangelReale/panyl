package panyl

import (
	"context"
)

// Log allows debugging each step of the processing
type Log interface {
	// LogSourceLine receives one receiced raw log line after running PluginClean and strings.TrimSpace.
	LogSourceLine(ctx context.Context, n int, line, rawLine string)
	// LogProcess receives one Process right before it is sent to ProcessResult.
	LogProcess(ctx context.Context, p *Process)
}
