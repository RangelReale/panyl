package panyl

const (
	PostProcessOrder_First   = 0
	PostProcessOrder_Last    = 10
	PostProcessOrder_Default = 5
)

type Option func(p *Processor)

func WithLineLimit(startLine, lineAmount int) Option {
	return func(p *Processor) {
		p.StartLine = startLine
		p.LineAmount = lineAmount
	}
}

func WithMaxBacklogLines(maxBacklogLines int) Option {
	return func(p *Processor) {
		p.MaxBacklogLines = maxBacklogLines
	}
}

func WithIncludeSource(includeSource bool) Option {
	return func(p *Processor) {
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

func WithScannerBufferSize(scannerBufferSize int) Option {
	return func(p *Processor) {
		p.ScannerBufferSize = scannerBufferSize
	}
}
