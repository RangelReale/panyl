package panyl

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errPluginTest = errors.New("plugin error")

// errorPlugin implements every plugin interface, returning an error from the stage named in failAt.
// Lines are consolidated one at a time.
type errorPlugin struct {
	failAt string
}

func (e errorPlugin) err(stage string) error {
	if e.failAt == stage {
		return errPluginTest
	}
	return nil
}

func (e errorPlugin) IsPanylPlugin() {}

func (e errorPlugin) Clean(ctx context.Context, item *Item) (bool, error) {
	return false, e.err("clean")
}

func (e errorPlugin) ExtractMetadata(ctx context.Context, item *Item) (bool, error) {
	return false, e.err("metadata")
}

func (e errorPlugin) ExtractStructure(ctx context.Context, lines ItemLines, item *Item) (bool, error) {
	return false, e.err("structure")
}

func (e errorPlugin) ExtractParse(ctx context.Context, lines ItemLines, item *Item) (bool, error) {
	return false, e.err("parse")
}

func (e errorPlugin) BlockSequence(ctx context.Context, lastp, item *Item) bool {
	return false
}

func (e errorPlugin) Consolidate(ctx context.Context, lines ItemLines, item *Item) (bool, int, error) {
	if err := e.err("consolidate"); err != nil {
		return false, 0, err
	}
	item.Line = lines[0].Line
	return true, 1, nil
}

func (e errorPlugin) ParseFormat(ctx context.Context, item *Item) (bool, error) {
	return false, e.err("parseformat")
}

func (e errorPlugin) PostProcessOrder() int { return PostProcessOrderDefault }

func (e errorPlugin) PostProcess(ctx context.Context, item *Item) (bool, error) {
	return false, e.err("postprocess")
}

func (e errorPlugin) CreateBefore(ctx context.Context, item *Item) ([]*Item, error) {
	return nil, e.err("createbefore")
}

func (e errorPlugin) CreateAfter(ctx context.Context, item *Item) ([]*Item, error) {
	return nil, e.err("createafter")
}

func TestJob_PluginErrors(t *testing.T) {
	for _, stage := range []string{"clean", "metadata", "structure", "parse", "consolidate", "parseformat",
		"postprocess", "createbefore", "createafter"} {
		t.Run(stage, func(t *testing.T) {
			res := &outputCloseTest{}
			err := NewProcessor(WithPlugins(errorPlugin{failAt: stage})).Process(context.Background(),
				strings.NewReader("a\nb\n"), res)
			assert.ErrorIs(t, err, errPluginTest)
			assert.True(t, res.closed)
		})
	}
}

func TestJob_PluginNoErrors(t *testing.T) {
	res := &OutputArray{}
	err := NewProcessor(WithPlugins(errorPlugin{})).Process(context.Background(), strings.NewReader("a\nb\n"), res)
	assert.NoError(t, err)
	assert.Len(t, res.List, 2)
}
