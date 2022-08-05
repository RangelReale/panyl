package panyl

const (
	PostProcessOrder_First   = 0
	PostProcessOrder_Last    = 10
	PostProcessOrder_Default = 5
)

type Option func(p *Processor)

type JobOption func(p *Job)

func WithLineLimit(startLine, lineAmount int) JobOption {
	return func(p *Job) {
		p.StartLine = startLine
		p.LineAmount = lineAmount
	}
}

func WithMaxBacklogLines(maxBacklogLines int) JobOption {
	return func(p *Job) {
		p.MaxBacklogLines = maxBacklogLines
	}
}

func WithIncludeSource(includeSource bool) JobOption {
	return func(p *Job) {
		p.IncludeSource = includeSource
	}
}

func WithLogger(logger Log) Option {
	return func(p *Processor) {
		p.Logger = logger
	}
}

func WithPlugin(plugin Plugin) Option {
	return func(p *Processor) {
		p.RegisterPlugin(plugin)
	}
}

func WithPlugins(plugin ...Plugin) Option {
	return func(p *Processor) {
		for _, pl := range plugin {
			p.RegisterPlugin(pl)
		}
	}
}
