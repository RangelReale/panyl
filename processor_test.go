package panyl

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessor_RegisterPlugin(t *testing.T) {
	p := NewProcessor()
	pl := &AllPlugins{}
	p.RegisterPlugin(pl)

	assert.Len(t, p.pluginClean, 1)
	assert.Len(t, p.pluginMetadata, 1)
	assert.Len(t, p.pluginSequence, 1)
	assert.Len(t, p.pluginStructure, 1)
	assert.Len(t, p.pluginParse, 1)
	assert.Len(t, p.pluginConsolidate, 1)
	assert.Len(t, p.pluginParseFormat, 1)
	assert.Len(t, p.pluginPostProcess, 1)
	assert.Len(t, p.pluginCreate, 1)
}

func TestProcessor_CreatePlugin(t *testing.T) {
	ctx := context.Background()

	p := NewProcessor()
	pl := &CreatePluginTest{}
	p.RegisterPlugin(pl)

	res := &OutputArray{}
	err := p.Process(ctx, strings.NewReader(`line`), res)

	assert.NoError(t, err)

	assert.Len(t, res.List, 3)
	assert.Equal(t, res.List[0].Line, "line-before-create")
	assert.Equal(t, res.List[1].Line, "line-default")
	assert.Equal(t, res.List[2].Line, "line-after-create")
}

func TestProcessor_CreatePlugin_LineProvider(t *testing.T) {
	ctx := context.Background()

	p := NewProcessor()
	pl := &CreatePluginTest{}
	p.RegisterPlugin(pl)

	res := &OutputArray{}
	err := p.ProcessProvider(ctx, NewStaticLineProvider([]interface{}{
		InitItem(WithInitLine("line")),
	}), res)

	assert.NoError(t, err)

	assert.Len(t, res.List, 3)
	assert.Equal(t, res.List[0].Line, "line-before-create")
	assert.Equal(t, res.List[1].Line, "line-default")
	assert.Equal(t, res.List[2].Line, "line-after-create")
}

func TestProcessor_PostProcessOrder(t *testing.T) {
	ctx := context.Background()

	p := NewProcessor()
	p.RegisterPlugin(&PostProcessPluginTest{5})
	p.RegisterPlugin(&PostProcessPluginTest{2})
	p.RegisterPlugin(&PostProcessPluginTest{10})
	p.RegisterPlugin(&PostProcessPluginTest{7})
	p.RegisterPlugin(&PostProcessPluginTest{1})
	p.RegisterPlugin(&PostProcessPluginTest{7})

	res := &OutputArray{}
	err := p.Process(ctx, strings.NewReader(`line`), res)

	assert.NoError(t, err)

	assert.Len(t, res.List, 1)
	assert.Equal(t, res.List[0].Line, "line_1_2_5_7_7_10")
}

// AllPlugins
type AllPlugins struct {
}

func (ap AllPlugins) IsPanylPlugin() {}

func (ap AllPlugins) Clean(ctx context.Context, item *Item) (bool, error) {
	return false, nil
}

func (ap AllPlugins) ExtractMetadata(ctx context.Context, item *Item) (bool, error) {
	return false, nil
}

func (ap AllPlugins) ExtractStructure(ctx context.Context, lines ItemLines, item *Item) (bool, error) {
	return false, nil
}

func (ap AllPlugins) ExtractParse(ctx context.Context, lines ItemLines, item *Item) (bool, error) {
	return false, nil
}

func (ap AllPlugins) BlockSequence(ctx context.Context, lastp, item *Item) bool {
	return false
}

func (ap AllPlugins) Consolidate(ctx context.Context, lines ItemLines, item *Item) (bool, int, error) {
	return false, -1, nil
}

func (ap AllPlugins) ParseFormat(ctx context.Context, item *Item) (bool, error) {
	return false, nil
}

func (ap AllPlugins) CreateBefore(ctx context.Context, item *Item) ([]*Item, error) {
	return nil, nil
}

func (ap AllPlugins) CreateAfter(ctx context.Context, item *Item) ([]*Item, error) {
	return nil, nil
}

func (ap AllPlugins) PostProcessOrder() int {
	return PostProcessOrderDefault
}

func (ap AllPlugins) PostProcess(ctx context.Context, item *Item) (bool, error) {
	return false, nil
}

// CreatePluginTest
type CreatePluginTest struct {
}

func (pt CreatePluginTest) IsPanylPlugin() {}

func (pt CreatePluginTest) CreateBefore(ctx context.Context, item *Item) ([]*Item, error) {
	return []*Item{
		InitItem(WithInitLine("line-before")),
	}, nil
}

func (pt CreatePluginTest) CreateAfter(ctx context.Context, item *Item) ([]*Item, error) {
	return []*Item{
		InitItem(WithInitLine("line-after")),
	}, nil
}

func (pt CreatePluginTest) PostProcessOrder() int {
	return PostProcessOrderDefault
}

func (pt CreatePluginTest) PostProcess(ctx context.Context, item *Item) (bool, error) {
	if item.Metadata.BoolValue(MetadataCreated) {
		item.Line += "-create"
	} else {
		item.Line += "-default"
	}
	return true, nil
}

// PostProcessPluginTest
type PostProcessPluginTest struct {
	order int
}

func (pt PostProcessPluginTest) IsPanylPlugin() {}

func (pt PostProcessPluginTest) PostProcessOrder() int {
	return pt.order
}

func (pt PostProcessPluginTest) PostProcess(ctx context.Context, item *Item) (bool, error) {
	item.Line += fmt.Sprintf("_%d", pt.order)
	return true, nil
}
