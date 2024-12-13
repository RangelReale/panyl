package panyl

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Job struct {
	processor               *Processor
	result                  ProcessResult
	lineno                  int
	lastTime                time.Time
	lines                   ProcessLines
	sortedPluginPostProcess []PluginPostProcess
	m                       sync.Mutex

	StartLine       int
	LineAmount      int
	IncludeSource   bool
	MaxBacklogLines int
}

var ErrFinished = errors.New("finished")

func NewJob(processor *Processor, result ProcessResult, options ...JobOption) *Job {
	ret := &Job{
		processor:               processor,
		result:                  result,
		sortedPluginPostProcess: getSortedPluginPostProcess(processor),

		MaxBacklogLines: 50,
	}
	for _, o := range options {
		o(ret)
	}
	return ret
}

func (p *Job) ProcessLine(line interface{}) error {
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
	var process *Process
	switch l := line.(type) {
	case string:
		process = p.initProcess(p.lineno, l)
		sourceLine = l
	case *Process:
		process = l
		process.LineNo = p.lineno
		p.ensureProcess(process)

		if p.processor.Logger != nil {
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
		_, err := pclean.Clean(process)
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

	// Log source line
	if p.processor.Logger != nil {
		p.processor.Logger.LogSourceLine(p.lineno, process.Line, sourceLine)
	}

	// PROCESS: Extract metadata
	for _, pmetadata := range p.processor.pluginMetadata {
		_, err := pmetadata.ExtractMetadata(process)
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
			if ok, err := pstructure.ExtractStructure(p.lines[curline:], process); err != nil {
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
				if ok, err := pparse.ExtractParse(p.lines[curline:], process); err != nil {
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
			process.Source = ProcessLines(p.lines[lineFound:]).Source()
		}
		if p.lastTime.IsZero() {
			// try to get the timestamp from the processed line if time is Zero
			if pts, ok := process.Metadata[Metadata_Timestamp]; ok {
				p.lastTime = pts.(time.Time)
			}
		}
		// process previous lines
		var err error
		_, err = p.processResultLines(p.lines[:lineFound], p.result, p.lastTime, p.sortedPluginPostProcess)
		if err != nil {
			return err
		}
		// process current line
		p.lastTime, err = p.outputResult(process, p.result, p.lastTime, p.sortedPluginPostProcess)
		if err != nil {
			return err
		}
		p.lines = nil
	} else {
		if len(p.lines) > 1 {
			// check if there is any sequence block in the last 2 lines
			blockSequence := false
			for _, psequence := range p.processor.pluginSequence {
				if bseq := psequence.BlockSequence(p.lines[len(p.lines)-2], p.lines[len(p.lines)-1]); bseq {
					blockSequence = true
					break
				}
			}

			if blockSequence {
				// process previous lines and leave only the current line
				var err error
				p.lastTime, err = p.processResultLines(p.lines[:len(p.lines)-1], p.result, p.lastTime, p.sortedPluginPostProcess)
				if err != nil {
					return err
				}
				p.lines = ProcessLines{p.lines[len(p.lines)-1]}
			}
		}
	}

	if len(p.lines) > p.MaxBacklogLines {
		var err error
		p.lastTime, err = p.processResultLines(p.lines, p.result, p.lastTime, p.sortedPluginPostProcess)
		if err != nil {
			return err
		}
		p.lines = nil
	}

	return nil
}

func (p *Job) Finish() error {
	if len(p.lines) > 0 {
		// process any lines left
		_, err := p.processResultLines(p.lines, p.result, p.lastTime, p.sortedPluginPostProcess)
		if err != nil {
			return err
		}
	}

	// allows output flushing, like flushing network connections
	p.result.OnFlush()

	return nil
}

func (p *Job) initProcess(lineno int, line string) *Process {
	ret := &Process{
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

func (p *Job) ensureProcess(process *Process) {
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

// processResultLines process previous lines, trying to consolidate using Consolidate plugins, and outputs each result.
func (p *Job) processResultLines(lines ProcessLines, result ProcessResult, lastTime time.Time,
	sortedPluginPostProcess []PluginPostProcess) (time.Time, error) {
	var rts = lastTime
	startLine := 0
	for startLine < len(lines) {
		processed := false
		for _, pc := range p.processor.pluginConsolidate {
			consolidateProcess := p.initProcess(lines[startLine].LineNo, "")
			if ok, topLines, err := pc.Consolidate(lines[startLine:], consolidateProcess); err != nil {
				return time.Time{}, err
			} else if ok {
				if topLines > len(lines)-startLine {
					return time.Time{}, fmt.Errorf("Plugin requestd %d top lines but only %d are available", topLines, len(lines)-startLine)
				}

				consolidateProcess.LineCount = topLines
				if p.IncludeSource {
					consolidateProcess.Source = ProcessLines(lines[startLine : startLine+topLines]).Source()
				}
				rts, err = p.outputResult(consolidateProcess, result, rts, sortedPluginPostProcess)
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
			rts, err = p.outputResult(lines[startLine], result, rts, sortedPluginPostProcess)
			if err != nil {
				return time.Time{}, err
			}
			startLine++
		}
	}
	return rts, nil
}

// outputResult post-processes the Process and outputs the result.
func (p *Job) outputResult(process *Process, result ProcessResult, lastTime time.Time,
	sortedPluginPostProcess []PluginPostProcess) (time.Time, error) {
	// if no format was detected, call the ParseFormat plugins
	if _, ok := process.Metadata[Metadata_Format]; !ok {
		for _, pp := range p.processor.pluginParseFormat {
			ok, err := pp.ParseFormat(process)
			if err != nil {
				return time.Time{}, err
			} else if ok {
				break
			}
		}
	}

	return p.internalOutputResult(process, result, lastTime, true, sortedPluginPostProcess)
}

// outputResult post-processes the Process and outputs the result.
func (p *Job) internalOutputResult(process *Process, result ProcessResult, lastTime time.Time, create bool,
	sortedPluginPostProcess []PluginPostProcess) (time.Time, error) {
	retTime := lastTime
	// check for timestamp in metadata, add the last one if not available
	if _, ok := process.Metadata[Metadata_Timestamp]; !ok {
		if lastTime.IsZero() {
			process.Metadata[Metadata_Timestamp] = time.Now()
		} else {
			process.Metadata[Metadata_Timestamp] = lastTime
		}
		process.Metadata[Metadata_TimestampCalculated] = true
	} else {
		retTime = process.Metadata[Metadata_Timestamp].(time.Time)
	}

	for _, pp := range sortedPluginPostProcess {
		_, err := pp.PostProcess(process)
		if err != nil {
			return time.Time{}, err
		}
	}

	createFunc := func(isBefore bool) error {
		// call create plugins
		if create {
			for _, pp := range p.processor.pluginCreate {
				var items []*Process
				var err error
				if isBefore {
					items, err = pp.CreateBefore(process)
				} else {
					items, err = pp.CreateAfter(process)
				}
				if err != nil {
					if err != nil {
						return err
					}
				}
				for _, item := range items {
					item.Metadata[Metadata_Created] = true
					_, err = p.internalOutputResult(item, result, lastTime, false, sortedPluginPostProcess)
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

	// create "Create" plugin before outputting current result
	err := createFunc(true)
	if err != nil {
		return time.Time{}, err
	}

	if p.processor.Logger != nil {
		p.processor.Logger.LogProcess(process)
	}
	result.OnResult(process)

	// create Create plugin after outputting current result
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
