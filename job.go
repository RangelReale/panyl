package panyl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Job manages processing lines and detecting information from them.
type Job struct {
	processor               *Processor
	output                  Output
	lineno                  int
	lastTime                time.Time
	lines                   ItemLines
	sortedPluginPostProcess []PluginPostProcess
	m                       sync.Mutex
	outputStopped           bool // Output.OnItem returned false
	finished                bool // Finish was called

	StartLine       int
	LineAmount      int
	IncludeSource   bool
	MaxBacklogLines int
}

var ErrFinished = errors.New("finished")

// NewJob manages processing lines and detecting information from them.
func NewJob(processor *Processor, output Output, options ...JobOption) *Job {
	ret := &Job{
		processor:               processor,
		output:                  output,
		sortedPluginPostProcess: getSortedPluginPostProcess(processor),

		MaxBacklogLines: 50,
	}
	for _, o := range options {
		o(ret)
	}
	return ret
}

// ProcessLine adds a line to be processed. line can be `string` or `*Item`.
// An `*Item` is copied, the passed instance is never modified.
// Returns ErrFinished if the job was finished, or if the Output requested to stop.
func (p *Job) ProcessLine(ctx context.Context, line any) error {
	p.m.Lock()
	defer p.m.Unlock()

	if p.finished || p.outputStopped {
		return ErrFinished
	}

	p.lineno++

	if p.LineAmount > 0 {
		// line numbers start at 1, so a StartLine below 1 means "from the first line"
		startLine := max(p.StartLine, 1)
		if p.lineno < startLine {
			return nil
		}
		if p.lineno >= startLine+p.LineAmount {
			return ErrFinished
		}
	}

	// read line from LineProvider
	var sourceLine string
	var process *Item
	isItem := false
	switch l := line.(type) {
	case string:
		process = p.initItem(p.lineno, l)
		sourceLine = l
	case *Item:
		// copy the item so the caller's instance is not modified
		process = &Item{}
		*process = *l
		process.Metadata = cloneMap(l.Metadata)
		process.Data = cloneMap(l.Data)
		isItem = true
		process.LineNo = p.lineno
		p.ensureItem(process)

		if p.processor.DebugLog != nil {
			// encode source line for Logger
			sourceLineBytes, err := json.Marshal(process.Data)
			if err == nil {
				// ignore errors
				sourceLine = string(sourceLineBytes)
			}
		}
	default:
		return fmt.Errorf("unsupported line type %T, must be string or *Item", line)
	}

	// PROCESS: Clean
	for _, pclean := range p.processor.pluginClean {
		_, err := pclean.Clean(ctx, process)
		if err != nil {
			return err
		}
	}

	// PROCESS: Trim spaces
	process.Line = strings.TrimSpace(process.Line)
	// skip empty lines (structured items are kept if they carry any data)
	if len(process.Line) == 0 && (!isItem || (len(process.Data) == 0 && len(process.Metadata) == 0)) {
		return nil
	}

	// DebugLog source line
	if p.processor.DebugLog != nil {
		p.processor.DebugLog.LogSourceLine(ctx, p.lineno, process.Line, sourceLine)
	}

	// PROCESS: Extract metadata
	for _, pmetadata := range p.processor.pluginMetadata {
		_, err := pmetadata.ExtractMetadata(ctx, process)
		if err != nil {
			return err
		}
	}

	if p.IncludeSource {
		// source with Clean and Metadata plugins applied
		process.Source = process.Line
	}

	// add current process to lines
	p.lines = append(p.lines, process)

	lineProcessed := false
	var lineFound int = -1

	// changes made by plugins that don't match are discarded by restoring this snapshot
	var snapshot itemSnapshot
	if len(p.processor.pluginStructure) > 0 || len(p.processor.pluginParse) > 0 {
		snapshot = newItemSnapshot(process)
	}

	// PROCESS: Extract structure from line
	// loop bottom lines until a match is found
structureloop:
	for curline := len(p.lines) - 1; curline >= 0; curline-- {
		for _, pstructure := range p.processor.pluginStructure {
			if ok, err := pstructure.ExtractStructure(ctx, p.lines[curline:], process); err != nil {
				return err
			} else if ok {
				lineProcessed = true
				lineFound = curline
				// line structure can be found only once
				break structureloop
			}
			snapshot.restore(process)
		}
	}

	// PROCESS: Parse line
	if !lineProcessed {
	lineloop:
		for curline := len(p.lines) - 1; curline >= 0; curline-- {
			for _, pparse := range p.processor.pluginParse {
				if ok, err := pparse.ExtractParse(ctx, p.lines[curline:], process); err != nil {
					return err
				} else if ok {
					lineProcessed = true
					lineFound = curline
					// line parser can be found only once
					break lineloop
				}
				snapshot.restore(process)
			}
		}
	}

	if lineProcessed {
		process.LineNo = p.lines[lineFound].LineNo
		process.LineCount = len(p.lines) - lineFound
		if p.IncludeSource {
			process.Source = ItemLines(p.lines[lineFound:]).Source()
		}
		if p.lastTime.IsZero() {
			// try to get the timestamp from the processed line if time is Zero
			// invalid types are reported by outputItem
			if pts, ok := process.Metadata[MetadataTimestamp].(time.Time); ok {
				p.lastTime = pts
			}
		}
		// process previous lines
		var err error
		p.lastTime, err = p.processResultLines(ctx, p.lines[:lineFound], p.output, p.lastTime, p.sortedPluginPostProcess)
		if err != nil {
			return err
		}
		// process current line
		p.lastTime, err = p.outputItem(ctx, process, p.output, p.lastTime, p.sortedPluginPostProcess)
		if err != nil {
			return err
		}
		p.lines = nil
	} else {
		if len(p.lines) > 1 {
			// check if there is any sequence block in the last 2 lines
			blockSequence := false
			for _, psequence := range p.processor.pluginSequence {
				if bseq := psequence.BlockSequence(ctx, p.lines[len(p.lines)-2], p.lines[len(p.lines)-1]); bseq {
					blockSequence = true
					break
				}
			}

			if blockSequence {
				// process previous lines and leave only the current line
				var err error
				p.lastTime, err = p.processResultLines(ctx, p.lines[:len(p.lines)-1], p.output, p.lastTime, p.sortedPluginPostProcess)
				if err != nil {
					return err
				}
				p.lines = ItemLines{p.lines[len(p.lines)-1]}
			}
		}
	}

	if len(p.lines) > p.MaxBacklogLines {
		var err error
		p.lastTime, err = p.processResultLines(ctx, p.lines, p.output, p.lastTime, p.sortedPluginPostProcess)
		if err != nil {
			return err
		}
		p.lines = nil
	}

	return nil
}

// Finish outputs any lines left in the backlog, then flushes and closes the output.
// The output is always flushed and closed, even if processing the backlog fails.
// The backlog is not output if the Output requested to stop. Calling Finish more than once does nothing.
func (p *Job) Finish(ctx context.Context) error {
	p.m.Lock()
	defer p.m.Unlock()

	if p.finished {
		return nil
	}
	p.finished = true

	var err error
	if len(p.lines) > 0 && !p.outputStopped {
		// process any lines left
		_, err = p.processResultLines(ctx, p.lines, p.output, p.lastTime, p.sortedPluginPostProcess)
		if errors.Is(err, ErrFinished) {
			err = nil
		}
	}
	p.lines = nil

	// allows output flushing, like flushing network connections
	p.output.OnFlush(ctx)

	// close the output.
	p.output.OnClose(ctx)

	return err
}

func (p *Job) initItem(lineno int, line string) *Item {
	ret := &Item{
		LineNo:   lineno,
		Metadata: map[string]interface{}{},
		Data:     map[string]interface{}{},
		Line:     line,
	}
	if p.IncludeSource {
		ret.RawSource = line
	}
	return ret
}

func (p *Job) ensureItem(process *Item) {
	if process.Metadata == nil {
		process.Metadata = map[string]interface{}{}
	}
	if process.Data == nil {
		process.Data = map[string]interface{}{}
	}
	if !p.IncludeSource {
		process.RawSource = ""
	}
}

// processResultLines process previous lines, trying to consolidate using Consolidate plugins, and outputs each output.
func (p *Job) processResultLines(ctx context.Context, lines ItemLines, output Output, lastTime time.Time,
	sortedPluginPostProcess []PluginPostProcess) (time.Time, error) {
	var rts = lastTime
	startLine := 0
	for startLine < len(lines) {
		processed := false
		for _, pc := range p.processor.pluginConsolidate {
			consolidateProcess := p.initItem(lines[startLine].LineNo, "")
			if ok, topLines, err := pc.Consolidate(ctx, lines[startLine:], consolidateProcess); err != nil {
				return time.Time{}, err
			} else if ok {
				if topLines < 1 || topLines > len(lines)-startLine {
					return time.Time{}, fmt.Errorf("consolidate plugin requested %d top lines but only 1 to %d are allowed",
						topLines, len(lines)-startLine)
				}

				consolidateProcess.LineCount = topLines
				if p.IncludeSource {
					consolidateProcess.Source = ItemLines(lines[startLine : startLine+topLines]).Source()
				}
				rts, err = p.outputItem(ctx, consolidateProcess, output, rts, sortedPluginPostProcess)
				if err != nil {
					return time.Time{}, err
				}
				startLine += topLines
				processed = true
				break
			}
		}
		if !processed {
			lines[startLine].LineCount = 1
			var err error
			rts, err = p.outputItem(ctx, lines[startLine], output, rts, sortedPluginPostProcess)
			if err != nil {
				return time.Time{}, err
			}
			startLine++
		}
	}
	return rts, nil
}

// outputItem post-processes the Item and outputs the output.
func (p *Job) outputItem(ctx context.Context, process *Item, output Output, lastTime time.Time,
	sortedPluginPostProcess []PluginPostProcess) (time.Time, error) {
	// if no format was detected, call the ParseFormat plugins
	if _, ok := process.Metadata[MetadataFormat]; !ok {
		for _, pp := range p.processor.pluginParseFormat {
			ok, err := pp.ParseFormat(ctx, process)
			if err != nil {
				return time.Time{}, err
			} else if ok {
				break
			}
		}
	}

	return p.internalOutputItem(ctx, process, output, lastTime, true, sortedPluginPostProcess)
}

// internalOutputItem post-processes the Item and outputs the output.
// Returns ErrFinished if the Output requested to stop.
func (p *Job) internalOutputItem(ctx context.Context, process *Item, output Output, lastTime time.Time, create bool,
	sortedPluginPostProcess []PluginPostProcess) (time.Time, error) {
	for _, pp := range sortedPluginPostProcess {
		_, err := pp.PostProcess(ctx, process)
		if err != nil {
			return time.Time{}, err
		}
	}

	retTime := lastTime
	// check for timestamp in metadata, add the last one if not available
	if pts, ok := process.Metadata[MetadataTimestamp]; !ok {
		if lastTime.IsZero() {
			process.Metadata[MetadataTimestamp] = time.Now()
		} else {
			process.Metadata[MetadataTimestamp] = lastTime
		}
		process.Metadata[MetadataTimestampCalculated] = true
	} else if ts, ok := pts.(time.Time); ok {
		retTime = ts
	} else {
		return time.Time{}, fmt.Errorf("metadata %q must be a time.Time, got %T", MetadataTimestamp, pts)
	}

	if process.Metadata.BoolValue(MetadataSkip) {
		return lastTime, nil
	}

	createFunc := func(isBefore bool) error {
		// call create plugins
		if create {
			for _, pp := range p.processor.pluginCreate {
				var items []*Item
				var err error
				if isBefore {
					items, err = pp.CreateBefore(ctx, process)
				} else {
					items, err = pp.CreateAfter(ctx, process)
				}
				if err != nil {
					return err
				}
				for _, item := range items {
					if item.Metadata == nil {
						item.Metadata = MapValue{}
					}
					if item.Data == nil {
						item.Data = MapValue{}
					}
					item.Metadata[MetadataCreated] = true
					_, err = p.internalOutputItem(ctx, item, output, retTime, false, sortedPluginPostProcess)
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	}

	// create "Create" plugin before outputting current item.
	err := createFunc(true)
	if err != nil {
		return time.Time{}, err
	}

	if p.processor.DebugLog != nil {
		p.processor.DebugLog.LogItem(ctx, process)
	}
	if !output.OnItem(ctx, process) {
		p.outputStopped = true
		return time.Time{}, ErrFinished
	}

	// create Create plugin after outputting current item.
	err = createFunc(false)
	if err != nil {
		return time.Time{}, err
	}

	return retTime, nil
}

func getSortedPluginPostProcess(processor *Processor) []PluginPostProcess {
	orderPlugins := map[int][]PluginPostProcess{}
	var orderList []int

	for _, plugin := range processor.pluginPostProcess {
		order := plugin.PostProcessOrder()
		if _, ok := orderPlugins[order]; !ok {
			orderPlugins[order] = []PluginPostProcess{}
			orderList = append(orderList, order)
		}
		orderPlugins[order] = append(orderPlugins[order], plugin)
	}

	sort.Ints(orderList)

	var ret []PluginPostProcess
	for _, order := range orderList {
		for _, plugin := range orderPlugins[order] {
			ret = append(ret, plugin)
		}
	}
	return ret
}
