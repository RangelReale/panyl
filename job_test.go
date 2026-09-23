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

func TestJob_StructuredItemWithOnlyData(t *testing.T) {
	ctx := context.Background()

	res := &OutputArray{}
	err := NewProcessor().ProcessProvider(ctx, NewStaticLineProvider([]any{
		InitItem(WithInitCustom(func(item *Item) {
			item.Data["message"] = "hello"
		})),
		InitItem(), // no data at all, is skipped
	}), res)

	require.NoError(t, err)
	require.Len(t, res.List, 1)
	assert.Equal(t, "hello", res.List[0].Data.StringValue("message"))
}

func TestJob_ConsolidateInvalidTopLines(t *testing.T) {
	for _, topLines := range []int{0, -1, 3} {
		ctx := context.Background()

		p := NewProcessor(WithPlugins(&consolidateTopLinesTest{topLines}))

		done := make(chan error, 1)
		go func() {
			done <- p.Process(ctx, strings.NewReader("a\nb\n"), &OutputNull{})
		}()

		select {
		case err := <-done:
			assert.Error(t, err, "topLines=%d", topLines)
		case <-time.After(5 * time.Second):
			t.Fatalf("topLines=%d: processing did not finish", topLines)
		}
	}
}

func TestJob_LineLimit(t *testing.T) {
	for _, test := range []struct {
		startLine, lineAmount int
		expected              []string
	}{
		{0, 2, []string{"1", "2"}},
		{1, 2, []string{"1", "2"}},
		{2, 2, []string{"2", "3"}},
		{5, 10, []string{"5", "6"}},
		{3, 0, []string{"1", "2", "3", "4", "5", "6"}},
	} {
		ctx := context.Background()

		res := &OutputArray{}
		err := NewProcessor().Process(ctx, strings.NewReader("1\n2\n3\n4\n5\n6\n"), res,
			WithLineLimit(test.startLine, test.lineAmount))
		require.NoError(t, err)

		var lines []string
		for _, item := range res.List {
			lines = append(lines, item.Line)
		}
		assert.Equal(t, test.expected, lines, "WithLineLimit(%d, %d)", test.startLine, test.lineAmount)
	}
}

func TestJob_ErrorFinishesJob(t *testing.T) {
	ctx := context.Background()

	res := &outputCloseTest{}
	err := NewProcessor(WithPlugins(&parseErrorTest{})).Process(ctx, strings.NewReader("a\nb\nerror\n"), res)

	assert.ErrorIs(t, err, errParseTest)
	// backlog lines are still output
	require.Len(t, res.List, 3)
	assert.Equal(t, "a", res.List[0].Line)
	assert.Equal(t, "b", res.List[1].Line)
	assert.True(t, res.flushed)
	assert.True(t, res.closed)
}

func TestJob_ScannerErrorFinishesJob(t *testing.T) {
	ctx := context.Background()

	res := &outputCloseTest{}
	err := NewProcessor().ProcessProvider(ctx, &errorLineProviderTest{lines: []string{"a", "b"}}, res)

	assert.ErrorIs(t, err, errScannerTest)
	assert.Len(t, res.List, 2)
	assert.True(t, res.flushed)
	assert.True(t, res.closed)
}

func TestJob_InvalidTimestampType(t *testing.T) {
	ctx := context.Background()

	var err error
	assert.NotPanics(t, func() {
		err = NewProcessor(WithPlugins(&invalidTimestampTest{})).Process(ctx, strings.NewReader("a\n"), &OutputNull{})
	})
	assert.ErrorContains(t, err, "time.Time")
}

func TestJob_CreatedItemWithNilMaps(t *testing.T) {
	ctx := context.Background()

	res := &OutputArray{}
	var err error
	assert.NotPanics(t, func() {
		err = NewProcessor(WithPlugins(&createNilMapsTest{})).Process(ctx, strings.NewReader("a\n"), res)
	})
	require.NoError(t, err)
	require.Len(t, res.List, 2)
	assert.Equal(t, "created", res.List[0].Line)
	assert.True(t, res.List[0].Metadata.BoolValue(MetadataCreated))
	assert.NotNil(t, res.List[0].Data)
	assert.Equal(t, "a", res.List[1].Line)
}

// consolidateTopLinesTest
type consolidateTopLinesTest struct {
	topLines int
}

func (c consolidateTopLinesTest) IsPanylPlugin() {}

func (c consolidateTopLinesTest) Consolidate(ctx context.Context, lines ItemLines, item *Item) (bool, int, error) {
	return true, c.topLines, nil
}

// parseErrorTest
var errParseTest = errors.New("parse error")

type parseErrorTest struct{}

func (p parseErrorTest) IsPanylPlugin() {}

func (p parseErrorTest) ExtractParse(ctx context.Context, lines ItemLines, item *Item) (bool, error) {
	if lines.Line() == "error" {
		return false, errParseTest
	}
	return false, nil
}

// errorLineProviderTest
var errScannerTest = errors.New("scanner error")

type errorLineProviderTest struct {
	lines []string
	line  string
	err   error
}

func (e *errorLineProviderTest) Err() error { return e.err }

func (e *errorLineProviderTest) Line() any { return e.line }

func (e *errorLineProviderTest) Scan(ctx context.Context) bool {
	if len(e.lines) == 0 {
		e.err = errScannerTest
		return false
	}
	e.line, e.lines = e.lines[0], e.lines[1:]
	return true
}

// outputCloseTest
type outputCloseTest struct {
	OutputArray
	flushed, closed bool
}

func (o *outputCloseTest) OnFlush(ctx context.Context) { o.flushed = true }

func (o *outputCloseTest) OnClose(ctx context.Context) { o.closed = true }

// invalidTimestampTest
type invalidTimestampTest struct{}

func (i invalidTimestampTest) IsPanylPlugin() {}

func (i invalidTimestampTest) ExtractParse(ctx context.Context, lines ItemLines, item *Item) (bool, error) {
	item.Metadata[MetadataTimestamp] = "2020-01-01T00:00:00Z"
	return true, nil
}

// createNilMapsTest
type createNilMapsTest struct{}

func (c createNilMapsTest) IsPanylPlugin() {}

func (c createNilMapsTest) CreateBefore(ctx context.Context, item *Item) ([]*Item, error) {
	return []*Item{{Line: "created"}}, nil
}

func (c createNilMapsTest) CreateAfter(ctx context.Context, item *Item) ([]*Item, error) {
	return nil, nil
}
