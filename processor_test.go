package panyl

import (
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

func (ap AllPlugins) PostProcess(result *Process) (bool, error) {
	return false, nil
}

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

func (pt CreatePluginTest) PostProcess(result *Process) (bool, error) {
	if result.Metadata.BoolValue(Metadata_Created) {
		result.Line += "-create"
	} else {
		result.Line += "-default"
	}
	return true, nil
}
