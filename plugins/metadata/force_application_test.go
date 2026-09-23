package metadata_test

import (
	"context"
	"testing"

	"github.com/RangelReale/panyl/v2"
	"github.com/RangelReale/panyl/v2/plugins/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForceApplication_ExtractMetadata(t *testing.T) {
	ctx := context.Background()
	p := metadata.ForceApplication{Application: "forced"}

	item := panyl.InitItem(panyl.WithInitLine("line"))
	ok, err := p.ExtractMetadata(ctx, item)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "forced", item.Metadata.StringValue(panyl.MetadataApplication))
	assert.Equal(t, "line", item.Line)

	// an existing application is kept
	item = panyl.InitItem()
	item.Metadata[panyl.MetadataApplication] = "existing"
	_, err = p.ExtractMetadata(ctx, item)
	require.NoError(t, err)
	assert.Equal(t, "existing", item.Metadata.StringValue(panyl.MetadataApplication))
}

func TestForceApplication_BlockSequence(t *testing.T) {
	ctx := context.Background()
	p := metadata.ForceApplication{Application: "forced"}

	newItem := func(app string) *panyl.Item {
		item := panyl.InitItem()
		if app != "" {
			item.Metadata[panyl.MetadataApplication] = app
		}
		return item
	}

	assert.False(t, p.BlockSequence(ctx, newItem("a"), newItem("a")))
	assert.True(t, p.BlockSequence(ctx, newItem("a"), newItem("b")))
	assert.True(t, p.BlockSequence(ctx, newItem("a"), newItem("")))
	assert.False(t, p.BlockSequence(ctx, newItem(""), newItem("")))
}
