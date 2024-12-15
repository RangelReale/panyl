package structure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/RangelReale/panyl/v2"
	"github.com/imdario/mergo"
)

// JSON extracts JSON data from the entire line.
// No format detection is made besides being a valid JSON string.
type JSON struct {
}

var _ panyl.PluginStructure = (*JSON)(nil)

func (m *JSON) ExtractStructure(ctx context.Context, lines panyl.ItemLines, result *panyl.Item) (bool, error) {
	jdec := json.NewDecoder(strings.NewReader(lines.Line()))
	jdata := map[string]interface{}{}
	err := jdec.Decode(&jdata)
	// check if the entire string was used
	if err != nil || jdec.More() {
		return false, nil
	}

	// merge previous data and metadata
	err = result.MergeLinesData(lines)
	if err != nil {
		return false, err
	}
	// clean the line as it was used entirely
	result.Line = ""

	// copy the parsed data to the result
	if err := mergo.Map(&result.Data, jdata); err != nil {
		return false, fmt.Errorf("Error merging structs: %v", err)
	}

	result.Metadata[panyl.MetadataStructure] = panyl.MetadataStructureJSON

	return true, nil
}

func (m JSON) IsPanylPlugin() {}
