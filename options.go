package panyl

import "context"

const (
	PostProcessOrderFirst   = 0
	PostProcessOrderLast    = 10
	PostProcessOrderDefault = 5
)

type Option func(p *Processor)

type JobOption func(p *Job)

// WithLineLimit outputs only starting from startLine up to the lineAmount amount of lines.
func WithLineLimit(startLine, lineAmount int) JobOption {
	return func(p *Job) {
		p.StartLine = startLine
		p.LineAmount = lineAmount
	}
}

// WithMaxBacklogLines sets the maximum amount of unprocessed lines to try until giving up.
// This is used to detect multiline logs.
func WithMaxBacklogLines(maxBacklogLines int) JobOption {
	return func(p *Job) {
		p.MaxBacklogLines = maxBacklogLines
	}
}

// WithIncludeSource sets whether to set Item.Source with the source line.
func WithIncludeSource(includeSource bool) JobOption {
	return func(p *Job) {
		p.IncludeSource = includeSource
	}
}

// WithDebugLog sets a DebugLog to be used for debugging.
func WithDebugLog(logger DebugLog) Option {
	return func(p *Processor) {
		p.DebugLog = logger
	}
}

// WithPlugins adds Plugin instances to be registered.
func WithPlugins(plugin ...Plugin) Option {
	return func(p *Processor) {
		for _, pl := range plugin {
			p.RegisterPlugin(pl)
		}
	}
}

// WithOnJobFinished sets a callback to be called when a Job is about to finish.
func WithOnJobFinished(f func(context.Context, *Job) error) Option {
	return func(p *Processor) {
		p.onJobFinished = append(p.onJobFinished, f)
	}
}
