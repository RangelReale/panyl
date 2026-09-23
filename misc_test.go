package panyl

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapValue_HasValue(t *testing.T) {
	m := MapValue{"a": 1, "nil": nil}
	assert.True(t, m.HasValue("a"))
	assert.True(t, m.HasValue("nil"))
	assert.False(t, m.HasValue("b"))
	assert.True(t, m.HasValues("a", "nil"))
	assert.True(t, m.HasValues())
	assert.False(t, m.HasValues("a", "b"))
}

func TestMapValue_StringValue(t *testing.T) {
	m := MapValue{"s": "text", "i": 1}
	assert.Equal(t, "text", m.StringValue("s"))
	assert.Equal(t, "", m.StringValue("i"))
	assert.Equal(t, "", m.StringValue("missing"))
}

func TestMapValue_FloatValue(t *testing.T) {
	for _, test := range []struct {
		value    any
		expected float64
	}{
		{1.5, 1.5},
		{float32(2.5), 2.5},
		{3, 3},
		{int8(-4), -4},
		{int16(5), 5},
		{int32(6), 6},
		{int64(7), 7},
		{uint(8), 8},
		{uint8(9), 9},
		{uint16(10), 10},
		{uint32(11), 11},
		{uint64(12), 12},
		{"13", 0},
	} {
		assert.Equal(t, test.expected, MapValue{"v": test.value}.FloatValue("v"), "%T(%v)", test.value, test.value)
	}
	assert.Equal(t, 0.0, MapValue{}.FloatValue("missing"))
}

func TestItem_MergeLinesData(t *testing.T) {
	item := InitItem(WithInitCustom(func(item *Item) {
		item.Metadata["own"] = "item"
		item.Data["shared"] = "item"
	}))
	err := item.MergeLinesData(ItemLines{
		InitItem(WithInitCustom(func(item *Item) {
			item.Metadata["first"] = 1
			item.Data["shared"] = "line1"
			item.Data["line1"] = true
		})),
		InitItem(WithInitCustom(func(item *Item) {
			item.Metadata["first"] = 2
			item.Data["line2"] = true
		})),
	})
	require.NoError(t, err)

	// existing values are never overridden
	assert.Equal(t, MapValue{"own": "item", "first": 1}, item.Metadata)
	assert.Equal(t, MapValue{"shared": "item", "line1": true, "line2": true}, item.Data)
}

func TestItem_Clone(t *testing.T) {
	item := InitItem(WithInitLine("line"), WithInitLineNo(3), WithInitLineCount(2), WithInitSource("source"),
		WithInitCustom(func(item *Item) {
			item.RawSource = "raw"
			item.Metadata["m"] = 1
			item.Data["d"] = 2
		}))

	clone, err := item.Clone()
	require.NoError(t, err)
	assert.Equal(t, item, clone)
	assert.NotSame(t, item, clone)

	data, err := item.CloneData()
	require.NoError(t, err)
	assert.Equal(t, &Item{Line: "line", Metadata: MapValue{"m": 1}, Data: MapValue{"d": 2}}, data)
}

func TestItemLines_LinesAndSources(t *testing.T) {
	lines := ItemLines{
		InitItem(WithInitLine("a"), WithInitSource("sa")),
		InitItem(WithInitLine("b"), WithInitSource("sb")),
	}
	assert.Equal(t, []string{"a", "b"}, lines.Lines())
	assert.Equal(t, []string{"sa", "sb"}, lines.Sources())
	assert.Nil(t, ItemLines{}.Lines())
}

func TestSLog(t *testing.T) {
	ctx := context.Background()

	// the default logger discards everything
	logger := SLogFromContext(ctx)
	require.NotNil(t, logger)
	assert.False(t, logger.Enabled(ctx, slog.LevelError))
	logger.With("a", 1).WithGroup("g").Error("discarded")

	var buf bytes.Buffer
	custom := slog.New(slog.NewTextHandler(&buf, nil))
	ctx = SLogToContext(ctx, custom)
	assert.Same(t, custom, SLogFromContext(ctx))
	SLogFromContext(ctx).Info("hello")
	assert.Contains(t, buf.String(), "msg=hello")
}

func TestLineProvider_StaticLineBeforeScan(t *testing.T) {
	lp := NewStaticLineProvider([]any{"a"})
	assert.Nil(t, lp.Line())
	assert.Error(t, lp.Err())
	assert.False(t, lp.Scan(context.Background()))
}

func TestLineProvider_ReaderLongLines(t *testing.T) {
	ctx := context.Background()

	// longer than the initial buffer, shorter than the maximum
	long := strings.Repeat("x", 200*1024)
	lp := NewReaderLineProvider(strings.NewReader(long+"\nshort\n"), DefaultScannerBufferSize)
	require.True(t, lp.Scan(ctx))
	assert.Equal(t, long, lp.Line())
	require.True(t, lp.Scan(ctx))
	assert.Equal(t, "short", lp.Line())
	assert.False(t, lp.Scan(ctx))
	assert.NoError(t, lp.Err())

	// longer than the maximum
	lp = NewReaderLineProvider(strings.NewReader(long+"\n"), 1024)
	assert.False(t, lp.Scan(ctx))
	assert.Error(t, lp.Err())

	// the error is returned by Process
	err := NewProcessor().Process(ctx, strings.NewReader(strings.Repeat("x", DefaultScannerBufferSize+1)), &OutputNull{})
	assert.Error(t, err)
}

func TestOutputImpl(t *testing.T) {
	ctx := context.Background()
	item := InitItem(WithInitLine("a"))

	var received []*Item
	outputs := []Output{
		OutputFunc(func(item *Item) { received = append(received, item) }),
		&OutputArray{},
		&OutputNull{},
	}
	for _, output := range outputs {
		assert.True(t, output.OnItem(ctx, item))
		output.OnFlush(ctx)
		output.OnClose(ctx)
	}
	assert.Equal(t, []*Item{item}, received)
	assert.Equal(t, []*Item{item}, outputs[1].(*OutputArray).List)
}
