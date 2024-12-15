package panyl

import (
	"context"
	"errors"
	"io"
)

// Processor sends lines to Job using a LineProvider.
type Processor struct {
	pluginClean       []PluginClean
	pluginMetadata    []PluginMetadata
	pluginSequence    []PluginSequence
	pluginStructure   []PluginStructure
	pluginParse       []PluginParse
	pluginConsolidate []PluginConsolidate
	pluginParseFormat []PluginParseFormat
	pluginPostProcess []PluginPostProcess
	pluginCreate      []PluginCreate
	onJobFinished     []func(context.Context, *Job) error

	DebugLog DebugLog
}

// NewProcessor sends lines to Job using a LineProvider.
func NewProcessor(options ...Option) *Processor {
	ret := &Processor{}
	for _, o := range options {
		o(ret)
	}
	return ret
}

// RegisterPlugin registers a Plugin. One Plugin instance may implement more than one plugin type.
func (p *Processor) RegisterPlugin(plugin Plugin) {
	if rp, ok := plugin.(PluginClean); ok {
		p.pluginClean = append(p.pluginClean, rp)
	}
	if rp, ok := plugin.(PluginMetadata); ok {
		p.pluginMetadata = append(p.pluginMetadata, rp)
	}
	if rp, ok := plugin.(PluginSequence); ok {
		p.pluginSequence = append(p.pluginSequence, rp)
	}
	if rp, ok := plugin.(PluginStructure); ok {
		p.pluginStructure = append(p.pluginStructure, rp)
	}
	if rp, ok := plugin.(PluginParse); ok {
		p.pluginParse = append(p.pluginParse, rp)
	}
	if rp, ok := plugin.(PluginConsolidate); ok {
		p.pluginConsolidate = append(p.pluginConsolidate, rp)
	}
	if rp, ok := plugin.(PluginParseFormat); ok {
		p.pluginParseFormat = append(p.pluginParseFormat, rp)
	}
	if rp, ok := plugin.(PluginPostProcess); ok {
		p.pluginPostProcess = append(p.pluginPostProcess, rp)
	}
	if rp, ok := plugin.(PluginCreate); ok {
		p.pluginCreate = append(p.pluginCreate, rp)
	}
}

// Item reads lines from an [io.Reander] until it returns [io.EOF], sending the items found to Output.
func (p *Processor) Process(ctx context.Context, r io.Reader, output Output, options ...JobOption) error {
	return p.ProcessProvider(ctx, NewReaderLineProvider(r, DefaultScannerBufferSize), output, options...)
}

// ProcessProvider reads lines from a [LineProvider] until [LineProvider.Scan] returns false, sending the items found to
// Output.
func (p *Processor) ProcessProvider(ctx context.Context, scanner LineProvider, output Output,
	options ...JobOption) error {
	job := NewJob(p, output, options...)
	var err error
	for scanner.Scan(ctx) {
		err = job.ProcessLine(ctx, scanner.Line())
		if err != nil {
			if errors.Is(err, ErrFinished) {
				break
			}
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	for _, jobFinished := range p.onJobFinished {
		_ = jobFinished(ctx, job)
	}

	return job.Finish(ctx)
}
