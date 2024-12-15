package panyl

import (
	"fmt"
	"strings"

	"github.com/imdario/mergo"
)

// Item is the result of parsing one or more lines
type Item struct {
	LineNo    int
	LineCount int
	Metadata  MapValue // is ALWAYS non-nil
	Data      MapValue // is ALWAYS non-nil
	Line      string   // line is the part of the line that might not be parsed
	RawSource string   // raw source from file
	Source    string   // source with Clean and Metadata plugins applied
}

// InitItem initializes an empty Item.
func InitItem(options ...InitItemOption) *Item {
	ret := &Item{
		Metadata: MapValue{},
		Data:     MapValue{},
	}
	for _, opt := range options {
		opt(ret)
	}
	return ret
}

type InitItemOption func(p *Item)

// WithInitLineNo sets Item.LineNo.
func WithInitLineNo(lineNo int) InitItemOption {
	return func(p *Item) {
		p.LineNo = lineNo
	}
}

// WithInitLineCount sets Item.LineCount.
func WithInitLineCount(lineCount int) InitItemOption {
	return func(p *Item) {
		p.LineCount = lineCount
	}
}

// WithInitLine sets Item.Line.
func WithInitLine(line string) InitItemOption {
	return func(p *Item) {
		p.Line = line
	}
}

// WithInitSource sets Item.Source.
func WithInitSource(source string) InitItemOption {
	return func(p *Item) {
		p.Source = source
	}
}

// WithInitCustom calls a callback to initialize a Item.
func WithInitCustom(f func(*Item)) InitItemOption {
	return func(p *Item) {
		f(p)
	}
}

func (p *Item) mergeData(other *Item) error {
	if err := mergo.Map(&p.Metadata, other.Metadata); err != nil {
		return fmt.Errorf("Error merging structs: %v", err)
	}
	if err := mergo.Map(&p.Data, other.Data); err != nil {
		return fmt.Errorf("Error merging structs: %v", err)
	}
	return nil
}

// MergeLinesData merges the Metadata and Data maps of a list of Item
func (p *Item) MergeLinesData(lines ItemLines) error {
	for _, line := range lines {
		err := p.mergeData(line)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
func (p *Item) CloneData() (*Item, error) {
	ret := &Item{
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

// ItemLines is a list of Item.
type ItemLines []*Item

// Lines returns a list of all lines from each Item.
func (pl ItemLines) Lines() []string {
	var ret []string
	for _, p := range pl {
		ret = append(ret, p.Line)
	}
	return ret
}

// Line returns a list of all lines from each Item joined with "\n".
func (pl ItemLines) Line() string {
	return strings.Join(pl.Lines(), "\n")
}

// Sources returns a list of all sources from each Item.
func (pl ItemLines) Sources() []string {
	var ret []string
	for _, p := range pl {
		ret = append(ret, p.Source)
	}
	return ret
}

// Source returns a list of all sources from each Item joined with "\n".
func (pl ItemLines) Source() string {
	return strings.Join(pl.Sources(), "\n")
}
