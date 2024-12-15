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

// ProcessLine adds a line to be processed. line can be `string` or `ProcessItem`.
func (p *Job) ProcessLine(ctx context.Context, line any) error {
	p.m.Lock()
	defer p.m.Unlock()

	p.lineno++

	if p.LineAmount > 0 {
		if p.lineno < p.StartLine {
			return nil
		}
		if p.lineno > p.StartLine+p.LineAmount {
			return ErrFinished
		}
	}

	// read line from LineProvider
	var sourceLine string
	var process *Item
	switch l := line.(type) {
	case string:
		process = p.initItem(p.lineno, l)
		sourceLine = l
	case *Item:
		process = l
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
	// skip empty lines
	if len(process.Line) == 0 {
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
			if pts, ok := process.Metadata[MetadataTimestamp]; ok {
				p.lastTime = pts.(time.Time)
			}
		}
		// process previous lines
		var err error
		_, err = p.processResultLines(ctx, p.lines[:lineFound], p.output, p.lastTime, p.sortedPluginPostProcess)
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

func (p *Job) Finish(ctx context.Context) error {
	if len(p.lines) > 0 {
		// process any lines left
		_, err := p.processResultLines(ctx, p.lines, p.output, p.lastTime, p.sortedPluginPostProcess)
		if err != nil {
			return err
		}
	}

	// allows output flushing, like flushing network connections
	p.output.OnFlush(ctx)

	// close the output.
	p.output.OnClose(ctx)

	return nil
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
				if topLines > len(lines)-startLine {
					return time.Time{}, fmt.Errorf("Plugin requestd %d top lines but only %d are available", topLines, len(lines)-startLine)
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

// outputItem post-processes the Item and outputs the output.
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
	if _, ok := process.Metadata[MetadataTimestamp]; !ok {
		if lastTime.IsZero() {
			process.Metadata[MetadataTimestamp] = time.Now()
		} else {
			process.Metadata[MetadataTimestamp] = lastTime
		}
		process.Metadata[MetadataTimestampCalculated] = true
	} else {
		retTime = process.Metadata[MetadataTimestamp].(time.Time)
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
					if err != nil {
						return err
					}
				}
				for _, item := range items {
					item.Metadata[MetadataCreated] = true
					_, err = p.internalOutputItem(ctx, item, output, lastTime, false, sortedPluginPostProcess)
					if err != nil {
						if err != nil {
							return err
						}
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
	output.OnItem(ctx, process)

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
