package clean_test

import (
	"context"
	"testing"

	"github.com/RangelReale/panyl/v2"
	"github.com/RangelReale/panyl/v2/plugins/clean"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnsiEscape(t *testing.T) {
	ctx := context.Background()

	item := panyl.InitItem(panyl.WithInitLine("\x1b[31mred\x1b[0m text"))
	ok, err := clean.AnsiEscape{}.Clean(ctx, item)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "red text", item.Line)
	assert.Equal(t, []string{panyl.MetadataCleanAnsiEscape}, item.Metadata.ListValue(panyl.MetadataClean))

	// the metadata is not duplicated
	item.Line = "\x1b[1mbold"
	ok, err = clean.AnsiEscape{}.Clean(ctx, item)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "bold", item.Line)
	assert.Equal(t, []string{panyl.MetadataCleanAnsiEscape}, item.Metadata.ListValue(panyl.MetadataClean))
}

func TestAnsiEscape_NoEscapes(t *testing.T) {
	item := panyl.InitItem(panyl.WithInitLine("plain text"))
	ok, err := clean.AnsiEscape{}.Clean(context.Background(), item)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, "plain text", item.Line)
	assert.False(t, item.Metadata.HasValue(panyl.MetadataClean))
}

func TestAnsiEscape_KeepsOtherCleanMetadata(t *testing.T) {
	item := panyl.InitItem(panyl.WithInitLine("\x1b[0mtext"))
	item.Metadata[panyl.MetadataClean] = []string{"other"}
	_, err := clean.AnsiEscape{}.Clean(context.Background(), item)
	require.NoError(t, err)
	assert.Equal(t, []string{"other", panyl.MetadataCleanAnsiEscape}, item.Metadata.ListValue(panyl.MetadataClean))
}
