package panyl

import "context"

// ProcessResult receives the result of each processed line
type ProcessResult interface {
	OnResult(ctx context.Context, p *Process) (cont bool)
	OnFlush(ctx context.Context)
	OnClose(ctx context.Context)
}
