package panyl

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJob_OutputStop(t *testing.T) {
	ctx := context.Background()

	res := &outputStopTest{stopAfter: 1}
	err := NewProcessor(WithPlugins(&parseAnyTest{})).Process(ctx, strings.NewReader("a\nb\nc\n"), res)

	require.NoError(t, err)
	require.Len(t, res.List, 1)
	assert.Equal(t, "a", res.List[0].Line)
	assert.True(t, res.flushed)
	assert.True(t, res.closed)
}

func TestJob_OutputStopBacklog(t *testing.T) {
	ctx := context.Background()

	// unparsed lines stay in the backlog until Finish
	res := &outputStopTest{stopAfter: 1}
	err := NewProcessor(WithPlugins(&parseErrorTest{})).Process(ctx, strings.NewReader("a\nb\nc\n"), res)

	require.NoError(t, err)
	require.Len(t, res.List, 1)
	assert.True(t, res.closed)
}

func TestJob_OutputStopCreate(t *testing.T) {
	ctx := context.Background()

	// stops at the item created before the first line
	res := &outputStopTest{stopAfter: 1}
	err := NewProcessor(WithPlugins(&CreatePluginTest{})).Process(ctx, strings.NewReader("a\nb\n"), res)

	require.NoError(t, err)
	require.Len(t, res.List, 1)
	assert.Equal(t, "line-before-create", res.List[0].Line)
}

func TestJob_UnsupportedLineType(t *testing.T) {
	ctx := context.Background()

	for _, line := range []any{nil, 12, []byte("a")} {
		var err error
		assert.NotPanics(t, func() {
			err = NewProcessor().ProcessProvider(ctx, NewStaticLineProvider([]any{line}), &OutputNull{})
		})
		assert.ErrorContains(t, err, "unsupported line type", "%T", line)
	}
}

func TestJob_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := &outputCloseTest{}
	err := NewProcessor().Process(ctx, strings.NewReader("a\nb\n"), res)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Len(t, res.List, 0)
	assert.True(t, res.closed)
}

func TestJob_ContextCancelledWhileProcessing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var lines []string
	// parse every line, so items are output while processing instead of at Finish
	err := NewProcessor(WithPlugins(&parseAnyTest{})).Process(ctx, strings.NewReader("a\nb\nc\n"), OutputFunc(func(item *Item) {
		lines = append(lines, item.Line)
		cancel()
	}))

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []string{"a"}, lines)
}

func TestLineProvider_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for _, lp := range []LineProvider{
		NewReaderLineProvider(strings.NewReader("a\n"), DefaultScannerBufferSize),
		NewStaticLineProvider([]any{"a"}),
	} {
		assert.False(t, lp.Scan(ctx))
		assert.ErrorIs(t, lp.Err(), context.Canceled)
	}
}

func TestProcessor_OnJobFinishedError(t *testing.T) {
	ctx := context.Background()

	errCallback := errors.New("callback error")
	called := 0
	p := NewProcessor(
		WithOnJobFinished(func(ctx context.Context, job *Job) error {
			called++
			return errCallback
		}),
		WithOnJobFinished(func(ctx context.Context, job *Job) error {
			called++
			return nil
		}),
	)

	res := &outputCloseTest{}
	err := p.Process(ctx, strings.NewReader("a\n"), res)

	assert.ErrorIs(t, err, errCallback)
	assert.Equal(t, 2, called)
	assert.True(t, res.closed)
}

func TestJob_FailedMatchChangesDiscarded(t *testing.T) {
	ctx := context.Background()

	check := &parseCheckCleanTest{}
	res := &OutputArray{}
	err := NewProcessor(WithPlugins(&structureMutateTest{}, &parseMutateTest{}, check)).
		Process(ctx, strings.NewReader("a\nb\n"), res)

	require.NoError(t, err)
	assert.False(t, check.sawChanges)
	require.Len(t, res.List, 2)
	for _, item := range res.List {
		assert.False(t, item.Metadata.HasValue("mutated"))
		assert.False(t, item.Data.HasValue("mutated"))
	}
	assert.Equal(t, "a", res.List[0].Line)
	assert.Equal(t, "b", res.List[1].Line)
}

func TestJob_TimestampFromBacklog(t *testing.T) {
	ctx := context.Background()

	ts := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	res := &OutputArray{}
	err := NewProcessor(WithPlugins(&timestampPrefixTest{ts}, &parseLineTest{"parsed"})).
		Process(ctx, strings.NewReader("ts line\nparsed\n"), res)

	require.NoError(t, err)
	require.Len(t, res.List, 2)
	assert.Equal(t, ts, res.List[1].Metadata[MetadataTimestamp])
	assert.True(t, res.List[1].Metadata.BoolValue(MetadataTimestampCalculated))
}

func TestJob_TimestampCreateAfter(t *testing.T) {
	ctx := context.Background()

	ts := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	res := &OutputArray{}
	err := NewProcessor(WithPlugins(&timestampPrefixTest{ts}, &CreatePluginTest{})).
		Process(ctx, strings.NewReader("ts line\n"), res)

	require.NoError(t, err)
	require.Len(t, res.List, 3)
	for _, item := range res.List {
		assert.Equal(t, ts, item.Metadata[MetadataTimestamp], item.Line)
	}
}

func TestJob_FinishTwice(t *testing.T) {
	ctx := context.Background()

	res := &outputCountTest{}
	job := NewJob(NewProcessor(WithPlugins(&parseErrorTest{})), res)
	require.NoError(t, job.ProcessLine(ctx, "a"))
	require.NoError(t, job.Finish(ctx))
	require.NoError(t, job.Finish(ctx))

	assert.Equal(t, 1, res.items)
	assert.Equal(t, 1, res.closed)
	assert.ErrorIs(t, job.ProcessLine(ctx, "b"), ErrFinished)
}

func TestJob_CallerItemNotModified(t *testing.T) {
	ctx := context.Background()

	item := InitItem(WithInitLine("  a  "), WithInitCustom(func(item *Item) {
		item.RawSource = "raw"
		item.Data["nested"] = map[string]any{"x": 1}
	}))

	res := &OutputArray{}
	err := NewProcessor(WithPlugins(&dataMutateTest{})).ProcessProvider(ctx, NewStaticLineProvider([]any{item}), res)

	require.NoError(t, err)
	require.Len(t, res.List, 1)
	assert.Equal(t, 1, res.List[0].LineNo)
	assert.Equal(t, "changed", res.List[0].Data.MapValue("nested")["x"])

	assert.Equal(t, 0, item.LineNo)
	assert.Equal(t, "  a  ", item.Line)
	assert.Equal(t, "raw", item.RawSource)
	assert.Equal(t, MapValue{"nested": map[string]any{"x": 1}}, item.Data)
	assert.Len(t, item.Metadata, 0)
}

func TestItem_CloneIsDeep(t *testing.T) {
	item := InitItem(WithInitCustom(func(item *Item) {
		item.Data["map"] = map[string]any{"x": 1}
		item.Data["mapvalue"] = MapValue{"y": 2}
		item.Data["any"] = []any{map[string]any{"z": 3}}
		item.Metadata["list"] = make([]string, 1, 10)
	}))

	clone, err := item.Clone()
	require.NoError(t, err)

	clone.Data.MapValue("map")["x"] = 10
	clone.Data.MapValue("mapvalue")["y"] = 20
	clone.Data["any"].([]any)[0].(map[string]any)["z"] = 30
	clone.Metadata.ListValueAdd("list", "added")

	assert.Equal(t, 1, item.Data.MapValue("map")["x"])
	assert.Equal(t, 2, item.Data.MapValue("mapvalue")["y"])
	assert.Equal(t, 3, item.Data["any"].([]any)[0].(map[string]any)["z"])
	assert.Equal(t, []string{""}, item.Metadata.ListValue("list"))
}

func TestMapValue_ListValueAddDoesNotShareBackingArray(t *testing.T) {
	list := make([]string, 1, 10)
	a := MapValue{"l": list}
	b := MapValue{"l": list}

	a.ListValueAdd("l", "a")
	b.ListValueAdd("l", "b")

	assert.Equal(t, []string{"", "a"}, a.ListValue("l"))
	assert.Equal(t, []string{"", "b"}, b.ListValue("l"))
}

// outputStopTest
type outputStopTest struct {
	outputCloseTest
	stopAfter int
}

func (o *outputStopTest) OnItem(ctx context.Context, item *Item) bool {
	o.List = append(o.List, item)
	return len(o.List) < o.stopAfter
}

// outputCountTest
type outputCountTest struct {
	items, closed int
}

func (o *outputCountTest) OnItem(ctx context.Context, item *Item) bool {
	o.items++
	return true
}

func (o *outputCountTest) OnFlush(ctx context.Context) {}

func (o *outputCountTest) OnClose(ctx context.Context) { o.closed++ }

// structureMutateTest changes the item but does not match.
type structureMutateTest struct{}

func (s structureMutateTest) IsPanylPlugin() {}

func (s structureMutateTest) ExtractStructure(ctx context.Context, lines ItemLines, item *Item) (bool, error) {
	item.Metadata["mutated"] = true
	item.Data["mutated"] = true
	item.Line = "mutated"
	return false, nil
}

// parseMutateTest changes the item but does not match.
type parseMutateTest struct{}

func (p parseMutateTest) IsPanylPlugin() {}

func (p parseMutateTest) ExtractParse(ctx context.Context, lines ItemLines, item *Item) (bool, error) {
	item.Metadata["mutated"] = true
	item.Line = "mutated"
	return false, nil
}

// parseCheckCleanTest checks that it never receives changes from other plugins.
type parseCheckCleanTest struct {
	sawChanges bool
}

func (p *parseCheckCleanTest) IsPanylPlugin() {}

func (p *parseCheckCleanTest) ExtractParse(ctx context.Context, lines ItemLines, item *Item) (bool, error) {
	if item.Metadata.HasValue("mutated") || item.Data.HasValue("mutated") || item.Line == "mutated" {
		p.sawChanges = true
	}
	for _, line := range lines {
		if line.Line == "mutated" {
			p.sawChanges = true
		}
	}
	return false, nil
}

// timestampPrefixTest sets a timestamp on lines starting with "ts ".
type timestampPrefixTest struct {
	ts time.Time
}

func (tp timestampPrefixTest) IsPanylPlugin() {}

func (tp timestampPrefixTest) ExtractMetadata(ctx context.Context, item *Item) (bool, error) {
	if strings.HasPrefix(item.Line, "ts ") {
		item.Metadata[MetadataTimestamp] = tp.ts
		return true, nil
	}
	return false, nil
}

// parseLineTest matches a single line with a fixed text.
type parseLineTest struct {
	line string
}

func (pl parseLineTest) IsPanylPlugin() {}

func (pl parseLineTest) ExtractParse(ctx context.Context, lines ItemLines, item *Item) (bool, error) {
	return len(lines) == 1 && lines[0].Line == pl.line, nil
}

// dataMutateTest changes nested data.
type dataMutateTest struct{}

func (d dataMutateTest) IsPanylPlugin() {}

func (d dataMutateTest) ExtractMetadata(ctx context.Context, item *Item) (bool, error) {
	item.Data.MapValue("nested")["x"] = "changed"
	item.Metadata["changed"] = true
	return true, nil
}

// parseAnyTest matches every single line.
type parseAnyTest struct{}

func (p parseAnyTest) IsPanylPlugin() {}

func (p parseAnyTest) ExtractParse(ctx context.Context, lines ItemLines, item *Item) (bool, error) {
	return len(lines) == 1, nil
}
