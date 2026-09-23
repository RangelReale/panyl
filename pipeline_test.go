package panyl_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/RangelReale/panyl/v2"
	"github.com/RangelReale/panyl/v2/plugins/clean"
	"github.com/RangelReale/panyl/v2/plugins/consolidate"
	"github.com/RangelReale/panyl/v2/plugins/metadata"
	"github.com/RangelReale/panyl/v2/plugins/structure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests of the processing pipeline, using the in-repo plugins.

func process(t *testing.T, input string, plugins []panyl.Plugin, options ...panyl.JobOption) []*panyl.Item {
	t.Helper()
	res := &panyl.OutputArray{}
	err := panyl.NewProcessor(panyl.WithPlugins(plugins...)).Process(context.Background(), strings.NewReader(input), res,
		options...)
	require.NoError(t, err)
	return res.List
}

type itemSummary struct {
	LineNo, LineCount int
	Line              string
}

func summarize(items []*panyl.Item) []itemSummary {
	var ret []itemSummary
	for _, item := range items {
		ret = append(ret, itemSummary{item.LineNo, item.LineCount, item.Line})
	}
	return ret
}

func TestPipeline_MultilineStructure(t *testing.T) {
	items := process(t, "before 1\nbefore 2\n{\n  \"a\": 1,\n\n  \"b\": \"x\"\n}\nafter\n",
		[]panyl.Plugin{structure.JSON{}})

	// empty lines are skipped, but still counted in the line numbers
	assert.Equal(t, []itemSummary{
		{1, 1, "before 1"},
		{2, 1, "before 2"},
		{3, 4, ""},
		{8, 1, "after"},
	}, summarize(items))
	assert.Equal(t, 1, items[2].Data.IntValue("a"))
	assert.Equal(t, "x", items[2].Data.StringValue("b"))
	assert.Equal(t, panyl.MetadataStructureJSON, items[2].Metadata.StringValue(panyl.MetadataStructure))
}

func TestPipeline_SingleLineStructuresInSequence(t *testing.T) {
	items := process(t, "{\"n\":1}\n{\"n\":2}\ntext\n{\"n\":3}\n", []panyl.Plugin{structure.JSON{}})

	require.Len(t, items, 4)
	assert.Equal(t, 1, items[0].Data.IntValue("n"))
	assert.Equal(t, 2, items[1].Data.IntValue("n"))
	assert.Equal(t, "text", items[2].Line)
	assert.Equal(t, 3, items[3].Data.IntValue("n"))
}

func TestPipeline_Consolidate(t *testing.T) {
	items := process(t, "one\ntwo\nthree\n{\"a\":1}\nfour\nfive\n",
		[]panyl.Plugin{structure.JSON{}, consolidate.JoinAllLines{}})

	// unmatched lines before a match are consolidated, and the ones left at the end
	assert.Equal(t, []itemSummary{
		{1, 3, "one\ntwo\nthree"},
		{4, 1, ""},
		{5, 2, "four\nfive"},
	}, summarize(items))
}

func TestPipeline_Sequence(t *testing.T) {
	items := process(t, "a| one\na| two\nb| three\nb| four\nfive\n",
		[]panyl.Plugin{prefixApplication{}, metadata.ForceApplication{Application: "default"}, consolidate.JoinAllLines{}})

	// a change of application breaks the sequence, so the lines are consolidated separately
	assert.Equal(t, []itemSummary{
		{1, 2, "one\ntwo"},
		{3, 2, "three\nfour"},
		{5, 1, "five"},
	}, summarize(items))
	assert.Equal(t, "a", items[0].Metadata.StringValue(panyl.MetadataApplication))
	assert.Equal(t, "b", items[1].Metadata.StringValue(panyl.MetadataApplication))
	assert.Equal(t, "default", items[2].Metadata.StringValue(panyl.MetadataApplication))
}

func TestPipeline_SequenceStructure(t *testing.T) {
	// a multi-line structure is not detected across a sequence break
	items := process(t, "a| {\nb| \"x\": 1\nb| }\n", []panyl.Plugin{prefixApplication{}, structure.JSON{},
		metadata.ForceApplication{Application: "default"}})

	assert.Equal(t, []itemSummary{
		{1, 1, "{"},
		{2, 1, "\"x\": 1"},
		{3, 1, "}"},
	}, summarize(items))
}

func TestPipeline_MaxBacklogLines(t *testing.T) {
	items := process(t, "1\n2\n3\n4\n5\n", []panyl.Plugin{consolidate.JoinAllLines{}}, panyl.WithMaxBacklogLines(2))

	// the backlog is flushed when it has more than 2 lines
	assert.Equal(t, []itemSummary{
		{1, 3, "1\n2\n3"},
		{4, 2, "4\n5"},
	}, summarize(items))
}

func TestPipeline_MaxBacklogLinesStructure(t *testing.T) {
	input := "{\n\"a\": 1,\n\"b\": 2\n}\n"

	// fits in the backlog
	items := process(t, input, []panyl.Plugin{structure.JSON{}}, panyl.WithMaxBacklogLines(4))
	require.Len(t, items, 1)
	assert.Equal(t, 4, items[0].LineCount)

	// larger than the backlog
	items = process(t, input, []panyl.Plugin{structure.JSON{}}, panyl.WithMaxBacklogLines(2))
	assert.Len(t, items, 4)
	for _, item := range items {
		assert.False(t, item.Metadata.HasValue(panyl.MetadataStructure))
	}
}

func TestPipeline_IncludeSource(t *testing.T) {
	input := "\x1b[31mred\x1b[0m\n{\n\"a\": 1\n}\n"
	plugins := []panyl.Plugin{clean.AnsiEscape{}, structure.JSON{}}

	items := process(t, input, plugins, panyl.WithIncludeSource(true))
	require.Len(t, items, 2)
	assert.Equal(t, "\x1b[31mred\x1b[0m", items[0].RawSource)
	assert.Equal(t, "red", items[0].Source)
	assert.Equal(t, []string{panyl.MetadataCleanAnsiEscape}, items[0].Metadata.ListValue(panyl.MetadataClean))
	assert.Equal(t, "{\n\"a\": 1\n}", items[1].Source)

	items = process(t, input, plugins)
	require.Len(t, items, 2)
	for _, item := range items {
		assert.Empty(t, item.RawSource)
		assert.Empty(t, item.Source)
	}
}

func TestPipeline_IncludeSourceConsolidate(t *testing.T) {
	items := process(t, "one\ntwo\n", []panyl.Plugin{consolidate.JoinAllLines{}}, panyl.WithIncludeSource(true))
	require.Len(t, items, 1)
	assert.Equal(t, "one\ntwo", items[0].Source)
}

func TestPipeline_IncludeSourceItem(t *testing.T) {
	ctx := context.Background()

	newItem := func() *panyl.Item {
		return panyl.InitItem(panyl.WithInitLine("line"), panyl.WithInitCustom(func(item *panyl.Item) {
			item.RawSource = "raw"
		}))
	}

	for _, includeSource := range []bool{true, false} {
		res := &panyl.OutputArray{}
		err := panyl.NewProcessor().ProcessProvider(ctx, panyl.NewStaticLineProvider([]any{newItem()}), res,
			panyl.WithIncludeSource(includeSource))
		require.NoError(t, err)
		require.Len(t, res.List, 1)
		if includeSource {
			assert.Equal(t, "raw", res.List[0].RawSource)
			assert.Equal(t, "line", res.List[0].Source)
		} else {
			assert.Empty(t, res.List[0].RawSource)
			assert.Empty(t, res.List[0].Source)
		}
	}
}

func TestPipeline_Timestamp(t *testing.T) {
	t1 := time.Date(2020, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2020, 1, 1, 11, 0, 0, 0, time.UTC)

	start := time.Now()
	items := process(t, "no ts\nts=2020-01-01T10:00:00Z first\nno ts\nts=2020-01-01T11:00:00Z second\nno ts\n",
		[]panyl.Plugin{timestampPrefix{}, parseAll{}})
	require.Len(t, items, 5)

	// before any timestamp, the current time is used
	assert.True(t, items[0].Metadata.BoolValue(panyl.MetadataTimestampCalculated))
	assert.False(t, items[0].Metadata[panyl.MetadataTimestamp].(time.Time).Before(start))

	for i, expected := range []struct {
		ts         time.Time
		calculated bool
	}{
		{t1, false},
		{t1, true},
		{t2, false},
		{t2, true},
	} {
		item := items[i+1]
		assert.Equal(t, expected.ts, item.Metadata[panyl.MetadataTimestamp], item.Line)
		assert.Equal(t, expected.calculated, item.Metadata.BoolValue(panyl.MetadataTimestampCalculated), item.Line)
	}
}

func TestPipeline_TimestampBacklog(t *testing.T) {
	t1 := time.Date(2020, 1, 1, 10, 0, 0, 0, time.UTC)

	// the unmatched lines before the first timestamp get the timestamp of the matched line
	items := process(t, "no ts 1\nno ts 2\nts=2020-01-01T10:00:00Z {\"a\":1}\n",
		[]panyl.Plugin{timestampPrefix{}, structure.JSON{}})
	require.Len(t, items, 3)
	for _, item := range items {
		assert.Equal(t, t1, item.Metadata[panyl.MetadataTimestamp], item.Line)
	}
	assert.True(t, items[0].Metadata.BoolValue(panyl.MetadataTimestampCalculated))
	assert.False(t, items[2].Metadata.BoolValue(panyl.MetadataTimestampCalculated))
}

func TestPipeline_Skip(t *testing.T) {
	t1 := time.Date(2020, 1, 1, 10, 0, 0, 0, time.UTC)

	items := process(t, "ts=2020-01-01T10:00:00Z one\nts=2020-01-01T11:00:00Z skip\nthree\n",
		[]panyl.Plugin{timestampPrefix{}, parseAll{}, skipPostProcess{}})

	require.Len(t, items, 2)
	assert.Equal(t, "one", items[0].Line)
	assert.Equal(t, "three", items[1].Line)
	// skipped items don't change the timestamp used for the next items
	assert.Equal(t, t1, items[1].Metadata[panyl.MetadataTimestamp])
}

func TestPipeline_ParseFormat(t *testing.T) {
	items := process(t, "{\"kind\":\"custom\"}\n{\"kind\":\"other\"}\ntext\n",
		[]panyl.Plugin{structure.JSON{}, &formatDetect{}})

	require.Len(t, items, 3)
	assert.Equal(t, "custom", items[0].Metadata.StringValue(panyl.MetadataFormat))
	assert.False(t, items[1].Metadata.HasValue(panyl.MetadataFormat))
	assert.False(t, items[2].Metadata.HasValue(panyl.MetadataFormat))
}

func TestPipeline_ParseFormatNotCalledWhenFormatSet(t *testing.T) {
	fd := &formatDetect{}
	items := process(t, "{\"kind\":\"custom\"}\n", []panyl.Plugin{structure.JSON{}, formatPostMetadata{}, fd})

	require.Len(t, items, 1)
	assert.Equal(t, "preset", items[0].Metadata.StringValue(panyl.MetadataFormat))
	assert.Equal(t, 0, fd.calls)
}

func TestPipeline_DebugLog(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	p := panyl.NewProcessor(
		panyl.WithPlugins(structure.JSON{}),
		panyl.WithDebugLog(panyl.NewDebugLogOutput(&buf)),
	)
	err := p.Process(ctx, strings.NewReader("text\n{\"a\":1}\n"), &panyl.OutputNull{})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 4)
	assert.Equal(t, "@@@ SOURCE LINE [1]: 'text' @@@", lines[0])
	assert.Equal(t, "@@@ SOURCE LINE [2]: '{\"a\":1}' @@@", lines[1])
	assert.True(t, strings.HasPrefix(lines[2], "*** PROCESS LINE [1]: Metadata: "), lines[2])
	assert.Contains(t, lines[2], "Line: \"text\"")
	assert.True(t, strings.HasPrefix(lines[3], "*** PROCESS LINE [2]: Metadata: "), lines[3])
	assert.Contains(t, lines[3], "Data: map[a:1]")
}

func TestPipeline_DebugLogItem(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	p := panyl.NewProcessor(panyl.WithDebugLog(panyl.NewDebugLogOutput(&buf).WithIncludeSource(true)))
	item := panyl.InitItem(panyl.WithInitLine("line"), panyl.WithInitCustom(func(item *panyl.Item) {
		item.Data["a"] = 1
	}))
	err := p.ProcessProvider(ctx, panyl.NewStaticLineProvider([]any{item}), &panyl.OutputNull{})
	require.NoError(t, err)

	// the raw line of an *Item is its data encoded as JSON
	assert.Contains(t, buf.String(), "@@@ SOURCE LINE [1]: 'line' (raw: '{\"a\":1}') @@@")
}

// prefixApplication sets the application from a "name| " prefix.
type prefixApplication struct{}

func (p prefixApplication) IsPanylPlugin() {}

func (p prefixApplication) ExtractMetadata(ctx context.Context, item *panyl.Item) (bool, error) {
	app, line, ok := strings.Cut(item.Line, "| ")
	if !ok {
		return false, nil
	}
	item.Metadata[panyl.MetadataApplication] = app
	item.Line = line
	return true, nil
}

// timestampPrefix sets the timestamp from a "ts=<RFC3339> " prefix.
type timestampPrefix struct{}

func (p timestampPrefix) IsPanylPlugin() {}

func (p timestampPrefix) ExtractMetadata(ctx context.Context, item *panyl.Item) (bool, error) {
	value, line, ok := strings.Cut(item.Line, " ")
	if !ok || !strings.HasPrefix(value, "ts=") {
		return false, nil
	}
	ts, err := time.Parse(time.RFC3339, strings.TrimPrefix(value, "ts="))
	if err != nil {
		return false, err
	}
	item.Metadata[panyl.MetadataTimestamp] = ts
	item.Line = line
	return true, nil
}

// parseAll matches every single line.
type parseAll struct{}

func (p parseAll) IsPanylPlugin() {}

func (p parseAll) ExtractParse(ctx context.Context, lines panyl.ItemLines, item *panyl.Item) (bool, error) {
	return len(lines) == 1, nil
}

// skipPostProcess skips items with the "skip" line.
type skipPostProcess struct{}

func (p skipPostProcess) IsPanylPlugin() {}

func (p skipPostProcess) PostProcessOrder() int { return panyl.PostProcessOrderDefault }

func (p skipPostProcess) PostProcess(ctx context.Context, item *panyl.Item) (bool, error) {
	if item.Line == "skip" {
		item.Metadata[panyl.MetadataSkip] = true
		return true, nil
	}
	return false, nil
}

// formatDetect sets the format from the "kind" data, if it is "custom".
type formatDetect struct {
	calls int
}

func (p *formatDetect) IsPanylPlugin() {}

func (p *formatDetect) ParseFormat(ctx context.Context, item *panyl.Item) (bool, error) {
	p.calls++
	if item.Data.StringValue("kind") == "custom" {
		item.Metadata[panyl.MetadataFormat] = "custom"
		return true, nil
	}
	return false, nil
}

// formatPostMetadata sets the format on every line.
type formatPostMetadata struct{}

func (p formatPostMetadata) IsPanylPlugin() {}

func (p formatPostMetadata) ExtractMetadata(ctx context.Context, item *panyl.Item) (bool, error) {
	item.Metadata[panyl.MetadataFormat] = "preset"
	return true, nil
}
