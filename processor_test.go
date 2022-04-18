package panyl

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
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
	p := NewProcessor()
	pl := &CreatePluginTest{}
	p.RegisterPlugin(pl)

	res := &ProcessResultArray{}
	err := p.Process(strings.NewReader(`line`), res)

	assert.NoError(t, err)

	assert.Len(t, res.List, 3)
	assert.Equal(t, res.List[0].Line, "line-before-create")
	assert.Equal(t, res.List[1].Line, "line-default")
	assert.Equal(t, res.List[2].Line, "line-after-create")
}

func TestProcessor_CreatePlugin_LineProvider(t *testing.T) {
	p := NewProcessor()
	pl := &CreatePluginTest{}
	p.RegisterPlugin(pl)

	res := &ProcessResultArray{}
	err := p.ProcessProvider(NewStaticLineProvider([]interface{}{
		InitProcess(WithInitLine("line")),
	}), res)

	assert.NoError(t, err)

	assert.Len(t, res.List, 3)
	assert.Equal(t, res.List[0].Line, "line-before-create")
	assert.Equal(t, res.List[1].Line, "line-default")
	assert.Equal(t, res.List[2].Line, "line-after-create")
}

func TestProcessor_PostProcessOrder(t *testing.T) {
	p := NewProcessor()
	p.RegisterPlugin(&PostProcessPluginTest{5})
	p.RegisterPlugin(&PostProcessPluginTest{2})
	p.RegisterPlugin(&PostProcessPluginTest{10})
	p.RegisterPlugin(&PostProcessPluginTest{7})
	p.RegisterPlugin(&PostProcessPluginTest{1})
	p.RegisterPlugin(&PostProcessPluginTest{7})

	res := &ProcessResultArray{}
	err := p.Process(strings.NewReader(`line`), res)

	assert.NoError(t, err)

	assert.Len(t, res.List, 1)
	assert.Equal(t, res.List[0].Line, "line_1_2_5_7_7_10")
}

// AllPlugins
type AllPlugins struct {
}

func (ap AllPlugins) IsPanylPlugin() {}

func (ap AllPlugins) Clean(result *Process) (bool, error) {
	return false, nil
}

func (ap AllPlugins) ExtractMetadata(result *Process) (bool, error) {
	return false, nil
}

func (ap AllPlugins) ExtractStructure(lines ProcessLines, result *Process) (bool, error) {
	return false, nil
}

func (ap AllPlugins) ExtractParse(lines ProcessLines, result *Process) (bool, error) {
	return false, nil
}

func (ap AllPlugins) BlockSequence(lastp, p *Process) bool {
	return false
}

func (ap AllPlugins) Consolidate(lines ProcessLines, result *Process) (bool, int, error) {
	return false, -1, nil
}

func (ap AllPlugins) ParseFormat(result *Process) (bool, error) {
	return false, nil
}

func (ap AllPlugins) CreateBefore(result *Process) ([]*Process, error) {
	return nil, nil
}

func (ap AllPlugins) CreateAfter(result *Process) ([]*Process, error) {
	return nil, nil
}

func (ap AllPlugins) PostProcessOrder() int {
	return PostProcessOrder_Default
}

func (ap AllPlugins) PostProcess(result *Process) (bool, error) {
	return false, nil
}

// CreatePluginTest
type CreatePluginTest struct {
}

func (pt CreatePluginTest) IsPanylPlugin() {}

func (pt CreatePluginTest) CreateBefore(result *Process) ([]*Process, error) {
	return []*Process{
		InitProcess(WithInitLine("line-before")),
	}, nil
}

func (pt CreatePluginTest) CreateAfter(result *Process) ([]*Process, error) {
	return []*Process{
		InitProcess(WithInitLine("line-after")),
	}, nil
}

func (pt CreatePluginTest) PostProcessOrder() int {
	return PostProcessOrder_Default
}

func (pt CreatePluginTest) PostProcess(result *Process) (bool, error) {
	if result.Metadata.BoolValue(Metadata_Created) {
		result.Line += "-create"
	} else {
		result.Line += "-default"
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

func (pt PostProcessPluginTest) PostProcess(result *Process) (bool, error) {
	result.Line += fmt.Sprintf("_%d", pt.order)
	return true, nil
}
