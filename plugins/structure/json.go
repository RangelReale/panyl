package structure

import (
	"encoding/json"
	"fmt"
	"github.com/RangelReale/panyl"
	"github.com/imdario/mergo"
	"strings"
)

var _ panyl.PluginStructure = (*JSON)(nil)

// JSON extracts JSON data from the entire line.
// No format detection is made besides being a valid JSON string.
type JSON struct {
}

func (m *JSON) ExtractStructure(lines panyl.ProcessLines, result *panyl.Process) (bool, error) {
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

	result.Metadata[panyl.Metadata_Structure] = panyl.MetadataStructure_JSON

	return true, nil
}

/*
func (m *JSON) ExtractStructureLines(lines []*panyl.Process, p *panyl.Process) (bool, int, error) {
	for startLine := 0; startLine < len(lines); startLine++ {
		ok, err := m.extractStructure(strings.Join(lines[startLine:], ""), p)
		if err != nil {
			return false, 0, err
		}
		if ok {
			return true, len(lines) - startLine, nil
		}
	}
	return false, 0, nil
}
*/

func (m JSON) IsPanylPlugin() {}
