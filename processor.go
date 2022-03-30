package panyl

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Processor struct {
	pluginClean       []PluginClean
	pluginMetadata    []PluginMetadata
	pluginSequence    []PluginSequence
	pluginStructure   []PluginStructure
	pluginParse       []PluginParse
	pluginConsolidate []PluginConsolidate
	pluginPostProcess []PluginPostProcess

	StartLine  int
	LineAmount int
	Logger     Log
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
	if rp, ok := plugin.(PluginPostProcess); ok {
		p.pluginPostProcess = append(p.pluginPostProcess, rp)
	}
}

func (p *Processor) initProcess(lineno int, line string) *Process {
	return &Process{
		LineNo:   lineno,
		Metadata: map[string]interface{}{},
		Data:     map[string]interface{}{},
		Line:     line,
	}
}

func (p *Processor) Process(r io.Reader, result ProcessResult) error {
	scanner := bufio.NewScanner(r)
	lineno := 0

	var lines ProcessLines

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
			// process previous lines
			err := p.processResultLines(lines[:lineFound], result)
			if err != nil {
				return err
			}
			// process current line
			err = p.outputResult(process, result)
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
					err := p.processResultLines(lines[:len(lines)-1], result)
					if err != nil {
						return err
					}
					lines = ProcessLines{lines[len(lines)-1]}
				}
			}
		}
	}

	if len(lines) > 0 {
		// process any lines left
		err := p.processResultLines(lines, result)
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
func (p *Processor) processResultLines(lines ProcessLines, result ProcessResult) error {
	startLine := 0
	for startLine < len(lines) {
		processed := false
		for _, pc := range p.pluginConsolidate {
			consolidateProcess := p.initProcess(lines[startLine].LineNo, "")
			if ok, topLines, err := pc.Consolidate(lines[startLine:], consolidateProcess); err != nil {
				return err
			} else if ok {
				if topLines > len(lines)-startLine {
					return fmt.Errorf("Plugin requestd %d top lines but only %d are available", topLines, len(lines)-startLine)
				}

				consolidateProcess.LineCount = topLines
				err = p.outputResult(consolidateProcess, result)
				if err != nil {
					return err
				}
				startLine += topLines
				processed = true
				break
			}
		}
		if !processed {
			lines[startLine].LineCount = 1
			err := p.outputResult(lines[startLine], result)
			if err != nil {
				return err
			}
			startLine++
		}
	}
	return nil
}

// outputResult post-processes the Process and outputs the result.
func (p *Processor) outputResult(process *Process, result ProcessResult) error {
	for _, pp := range p.pluginPostProcess {
		_, err := pp.PostProcess(process)
		if err != nil {
			return err
		}
	}
	if p.Logger != nil {
		p.Logger.LogProcess(process)
	}
	result.OnResult(process)
	return nil
}
