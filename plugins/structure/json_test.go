package structure_test

import (
	"context"
	"strings"
	"testing"

	"github.com/RangelReale/panyl/v2"
	"github.com/RangelReale/panyl/v2/plugins/structure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	for _, test := range []struct {
		line    string
		isMatch bool
	}{
		{`{"a":1}`, true},
		{`{"a":1}   `, true},
		{`{"a":1} }`, false},
		{`{"a":1}]`, false},
		{`{"a":1} {"b":2}`, false},
		{`{"a":1} x`, false},
		{`{"a":1`, false},
		{`not json`, false},
	} {
		item := panyl.InitItem(panyl.WithInitLine(test.line))
		ok, err := structure.JSON{}.ExtractStructure(context.Background(), panyl.ItemLines{item}, item)
		require.NoError(t, err)
		assert.Equal(t, test.isMatch, ok, test.line)
	}
}

func TestJSON_Multiline(t *testing.T) {
	ctx := context.Background()

	res := &panyl.OutputArray{}
	err := panyl.NewProcessor(panyl.WithPlugins(structure.JSON{})).
		Process(ctx, strings.NewReader("before\n{\n\"a\": 1\n}\n"), res)
	require.NoError(t, err)

	require.Len(t, res.List, 2)
	assert.Equal(t, "before", res.List[0].Line)
	assert.Equal(t, 1, res.List[1].Data.IntValue("a"))
	assert.Equal(t, 3, res.List[1].LineCount)
	assert.Equal(t, panyl.MetadataStructureJSON, res.List[1].Metadata.StringValue(panyl.MetadataStructure))
}
