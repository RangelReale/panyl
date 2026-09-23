package panyl

import (
	"fmt"
	"slices"
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

// Clone returns a deep copy of the Item. Nested maps and slices are copied, other values are shared.
// The error result is always nil, and is kept for compatibility.
func (p *Item) Clone() (*Item, error) {
	item, err := p.CloneData()
	if err != nil {
		return nil, err
	}
	item.LineNo = p.LineNo
	item.LineCount = p.LineCount
	item.Source = p.Source
	item.RawSource = p.RawSource
	return item, nil
}

// CloneData returns a deep copy of the Item's Line, Metadata and Data. Nested maps and slices are copied, other
// values are shared.
// The error result is always nil, and is kept for compatibility.
func (p *Item) CloneData() (*Item, error) {
	return &Item{
		Line:     p.Line,
		Metadata: cloneMap(p.Metadata),
		Data:     cloneMap(p.Data),
	}, nil
}

// cloneMap returns a deep copy of a map. It never returns nil.
func cloneMap(m map[string]any) MapValue {
	ret := make(MapValue, len(m))
	for k, v := range m {
		ret[k] = cloneValue(v)
	}
	return ret
}

// cloneValue returns a deep copy of maps and slices, other values are returned as-is.
func cloneValue(v any) any {
	switch vv := v.(type) {
	case MapValue:
		return cloneMap(vv)
	case map[string]any:
		return map[string]any(cloneMap(vv))
	case []any:
		if vv == nil {
			return vv
		}
		ret := make([]any, len(vv))
		for i, item := range vv {
			ret[i] = cloneValue(item)
		}
		return ret
	case []string:
		return slices.Clone(vv)
	default:
		return v
	}
}

// itemSnapshot stores a copy of an Item, to be able to restore it later.
type itemSnapshot struct {
	item Item
}

func newItemSnapshot(item *Item) itemSnapshot {
	s := itemSnapshot{item: *item}
	s.item.Metadata = cloneMap(item.Metadata)
	s.item.Data = cloneMap(item.Data)
	return s
}

// restore sets the item to the state of the snapshot. The snapshot can be restored multiple times.
func (s itemSnapshot) restore(item *Item) {
	*item = s.item
	item.Metadata = cloneMap(s.item.Metadata)
	item.Data = cloneMap(s.item.Data)
}

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
