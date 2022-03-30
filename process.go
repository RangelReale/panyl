package panyl

import (
	"fmt"
	"github.com/imdario/mergo"
	"strings"
)

// Process is the result of parsing one or more lines
type Process struct {
	LineNo    int
	LineCount int
	Metadata  MapValue
	Data      MapValue
	Line      string
}

func (p *Process) mergeData(other *Process) error {
	if err := mergo.Map(&p.Metadata, other.Metadata); err != nil {
		return fmt.Errorf("Error merging structs: %v", err)
	}
	if err := mergo.Map(&p.Data, other.Data); err != nil {
		return fmt.Errorf("Error merging structs: %v", err)
	}
	return nil
}

// MergeLinesData merges the Metadata and Data maps of a list of Process
func (p *Process) MergeLinesData(lines ProcessLines) error {
	for _, line := range lines {
		err := p.mergeData(line)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
func (p *Process) CloneData() (*Process, error) {
	ret := &Process{
		Line:     p.Line,
		Metadata: map[string]interface{}{},
		Data:     map[string]interface{}{},
	}
	if err := mergo.Map(&ret.Metadata, p.Metadata); err != nil {
		return nil, fmt.Errorf("Error merging structs: %v", err)
	}
	if err := mergo.Map(&ret.Data, p.Data); err != nil {
		return nil, fmt.Errorf("Error merging structs: %v", err)
	}
	return ret, nil
}
*/

// ProcessLines is a list of Process
type ProcessLines []*Process

// Lines returns a list of all lines from each Process
func (pl ProcessLines) Lines() []string {
	var ret []string
	for _, p := range pl {
		ret = append(ret, p.Line)
	}
	return ret
}

// Line returns a list of all lines from each Process joined with \n
func (pl ProcessLines) Line() string {
	return strings.Join(pl.Lines(), "\n")
}

// ProcessResult receives the result of each processed line
type ProcessResult interface {
	OnResult(p *Process) (cont bool)
}

// ProcessResultFunc is a helper to use ProcessResult as a function
type ProcessResultFunc func(p *Process)

func (pr ProcessResultFunc) OnResult(p *Process) bool {
	pr(p)
	return true
}

// ProcessResultArray is a ProcessResult that accumulates in an array
type ProcessResultArray struct {
	List []*Process
}

func (pr *ProcessResultArray) OnResult(p *Process) bool {
	pr.List = append(pr.List, p)
	return true
}
