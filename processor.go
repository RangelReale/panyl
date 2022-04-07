package panyl

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

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

	StartLine         int
	LineAmount        int
	Logger            Log
	IncludeSource     bool
	MaxBacklogLines   int
	ScannerBufferSize int
}

func NewProcessor(options ...Option) *Processor {
	ret := &Processor{
		MaxBacklogLines:   50,
		ScannerBufferSize: 1024 * 1024,
	}
	for _, o := range options {
		o(ret)
	}
	return ret
}

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

func (p *Processor) initProcess(lineno int, line string) *Process {
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

func (p *Processor) Process(r io.Reader, result ProcessResult) error {
	scanner := bufio.NewScanner(r)
	if p.ScannerBufferSize > 0 {
		// adjust the scanner capacity
		buf := make([]byte, p.ScannerBufferSize)
		scanner.Buffer(buf, p.ScannerBufferSize)
	}

	lineno := 0

	var lastTime time.Time
	var lines ProcessLines
	sortedPluginPostProcess := p.sortedPluginPostProcess()

	for scanner.Scan() {
		line := scanner.Text()
		lineno++

		if p.LineAmount > 0 {
			if lineno < p.StartLine {
				continue
			}
			if lineno > p.StartLine+p.LineAmount {
				break
			}
		}

		process := p.initProcess(lineno, line)

		// PROCESS: Clean
		for _, pclean := range p.pluginClean {
			_, err := pclean.Clean(process)
			if err != nil {
				return err
			}
		}

		// PROCESS: Trim spaces
		process.Line = strings.TrimSpace(process.Line)
		// skip empty lines
		if len(process.Line) == 0 {
			continue
		}

		// Log source line
		if p.Logger != nil {
			p.Logger.LogSourceLine(lineno, process.Line, line)
		}

		// PROCESS: Extract metadata
		for _, pmetadata := range p.pluginMetadata {
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
		lines = append(lines, process)

		lineProcessed := false
		var lineFound int = -1

		// PROCESS: Extract structure from line
		// loop bottom lines until a match is found
	structureloop:
		for curline := len(lines) - 1; curline >= 0; curline-- {
			for _, pstructure := range p.pluginStructure {
				if ok, err := pstructure.ExtractStructure(lines[curline:], process); err != nil {
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
			for curline := len(lines) - 1; curline >= 0; curline-- {
				for _, pparse := range p.pluginParse {
					if ok, err := pparse.ExtractParse(lines[curline:], process); err != nil {
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
			process.LineNo = lines[lineFound].LineNo
			process.LineCount = len(lines) - lineFound
			if p.IncludeSource {
				process.Source = ProcessLines(lines[lineFound:]).Source()
			}
			if lastTime.IsZero() {
				// try to get the timestamp from the processed line if time is Zero
				if pts, ok := process.Metadata[Metadata_Timestamp]; ok {
					lastTime = pts.(time.Time)
				}
			}
			// process previous lines
			var err error
			_, err = p.processResultLines(lines[:lineFound], result, lastTime, sortedPluginPostProcess)
			if err != nil {
				return err
			}
			// process current line
			lastTime, err = p.outputResult(process, result, lastTime, sortedPluginPostProcess)
			if err != nil {
				return err
			}
			lines = nil
		} else {
			if len(lines) > 1 {
				// check if there is any sequence block in the last 2 lines
				blockSequence := false
				for _, psequence := range p.pluginSequence {
					if bseq := psequence.BlockSequence(lines[len(lines)-2], lines[len(lines)-1]); bseq {
						blockSequence = true
						break
					}
				}

				if blockSequence {
					// process previous lines and leave only the current line
					var err error
					lastTime, err = p.processResultLines(lines[:len(lines)-1], result, lastTime, sortedPluginPostProcess)
					if err != nil {
						return err
					}
					lines = ProcessLines{lines[len(lines)-1]}
				}
			}
		}

		if len(lines) > p.MaxBacklogLines {
			var err error
			lastTime, err = p.processResultLines(lines, result, lastTime, sortedPluginPostProcess)
			if err != nil {
				return err
			}
			lines = nil
		}
	}

	if len(lines) > 0 {
		// process any lines left
		_, err := p.processResultLines(lines, result, lastTime, sortedPluginPostProcess)
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// processResultLines process previous lines, trying to consolidate using Consolidate plugins, and outputs each result.
func (p *Processor) processResultLines(lines ProcessLines, result ProcessResult, lastTime time.Time,
	sortedPluginPostProcess []PluginPostProcess) (time.Time, error) {
	var rts = lastTime
	startLine := 0
	for startLine < len(lines) {
		processed := false
		for _, pc := range p.pluginConsolidate {
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
func (p *Processor) outputResult(process *Process, result ProcessResult, lastTime time.Time,
	sortedPluginPostProcess []PluginPostProcess) (time.Time, error) {
	// if no format was detected, call the ParseFormat plugins
	if _, ok := process.Metadata[Metadata_Format]; !ok {
		for _, pp := range p.pluginParseFormat {
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
func (p *Processor) internalOutputResult(process *Process, result ProcessResult, lastTime time.Time, create bool,
	sortedPluginPostProcess []PluginPostProcess) (time.Time, error) {
	// check for timestamp in metadata, add the last one if not available
	if _, ok := process.Metadata[Metadata_Timestamp]; !ok {
		process.Metadata[Metadata_Timestamp] = lastTime
		process.Metadata[Metadata_TimestampCalculated] = true
	}
	retTime := process.Metadata[Metadata_Timestamp].(time.Time)

	for _, pp := range p.sortedPluginPostProcess() {
		_, err := pp.PostProcess(process)
		if err != nil {
			return time.Time{}, err
		}
	}

	createFunc := func(isBefore bool) error {
		// call create plugins
		if create {
			for _, pp := range p.pluginCreate {
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

	// create Create plugin before outputting current result
	err := createFunc(true)
	if err != nil {
		return time.Time{}, err
	}

	if p.Logger != nil {
		p.Logger.LogProcess(process)
	}
	result.OnResult(process)

	// create Create plugin after outputting current result
	err = createFunc(false)
	if err != nil {
		return time.Time{}, err
	}

	return retTime, nil
}

func (p *Processor) sortedPluginPostProcess() []PluginPostProcess {
	orderPlugins := map[int][]PluginPostProcess{}
	var orderList []int

	for _, plugin := range p.pluginPostProcess {
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
