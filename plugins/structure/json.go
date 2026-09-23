package structure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/RangelReale/panyl/v2"
	"github.com/imdario/mergo"
)

// JSON extracts JSON data from the entire line.
// No format detection is made besides being a valid JSON string.
// Parsed values override data with the same keys that came from the source lines.
type JSON struct {
	// UseNumber decodes numbers as json.Number instead of float64, to avoid losing precision on large integers.
	UseNumber bool
}

var _ panyl.PluginStructure = JSON{}

func (m JSON) ExtractStructure(ctx context.Context, lines panyl.ItemLines, item *panyl.Item) (bool, error) {
	jdec := json.NewDecoder(strings.NewReader(lines.Line()))
	if m.UseNumber {
		jdec.UseNumber()
	}
	jdata := map[string]interface{}{}
	if err := jdec.Decode(&jdata); err != nil {
		return false, nil
	}
	// check if the entire string was used, only whitespace may follow
	if _, err := jdec.Token(); !errors.Is(err, io.EOF) {
		return false, nil
	}

	// merge previous data and metadata
	err := item.MergeLinesData(lines)
	if err != nil {
		return false, err
	}
	// clean the line as it was used entirely
	item.Line = ""

	// copy the parsed data to the item
	if err := mergo.Map(&item.Data, jdata, mergo.WithOverride); err != nil {
		return false, fmt.Errorf("Error merging structs: %v", err)
	}

	item.Metadata[panyl.MetadataStructure] = panyl.MetadataStructureJSON

	return true, nil
}

func (m JSON) IsPanylPlugin() {}
