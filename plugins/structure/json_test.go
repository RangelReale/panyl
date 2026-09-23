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

func TestJSON_OverridesLineData(t *testing.T) {
	item := panyl.InitItem(panyl.WithInitLine(`{"a":"json"}`), panyl.WithInitCustom(func(item *panyl.Item) {
		item.Data["a"] = "line"
		item.Data["b"] = "line"
	}))
	ok, err := structure.JSON{}.ExtractStructure(context.Background(), panyl.ItemLines{item}, item)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "json", item.Data.StringValue("a"))
	assert.Equal(t, "line", item.Data.StringValue("b"))
}

func TestJSON_UseNumber(t *testing.T) {
	const line = `{"id":9007199254740993,"f":1.5}`

	item := panyl.InitItem(panyl.WithInitLine(line))
	ok, err := structure.JSON{UseNumber: true}.ExtractStructure(context.Background(), panyl.ItemLines{item}, item)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, 9007199254740993, item.Data.IntValue("id"))
	assert.Equal(t, 1.5, item.Data.FloatValue("f"))
	assert.Equal(t, 1, item.Data.IntValue("f"))

	// default decodes as float64, and loses precision
	item = panyl.InitItem(panyl.WithInitLine(line))
	ok, err = structure.JSON{}.ExtractStructure(context.Background(), panyl.ItemLines{item}, item)
	require.NoError(t, err)
	require.True(t, ok)
	assert.IsType(t, float64(0), item.Data["id"])
}
