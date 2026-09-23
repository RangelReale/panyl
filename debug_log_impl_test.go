package panyl

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDebugLogOutput_LogSourceLine(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	NewDebugLogOutput(&buf).LogSourceLine(ctx, 1, "clean", "\x1b[31mraw")
	assert.Equal(t, "@@@ SOURCE LINE [1]: 'clean' @@@\n", buf.String())

	buf.Reset()
	l := NewDebugLogOutput(&buf).WithIncludeSource(true)
	assert.True(t, l.IncludeSource)
	l.LogSourceLine(ctx, 2, "clean", "\x1b[31mraw")
	assert.Equal(t, "@@@ SOURCE LINE [2]: 'clean' (raw: 'raw') @@@\n", buf.String())

	// the raw line is not repeated if equal
	buf.Reset()
	l.LogSourceLine(ctx, 3, "same", "same")
	assert.Equal(t, "@@@ SOURCE LINE [3]: 'same' @@@\n", buf.String())
}

func TestDebugLogOutput_LogItem(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	item := InitItem(WithInitLine("text"), WithInitLineCount(2), WithInitLineNo(4), WithInitSource("\x1b[1msource"))
	item.Metadata["a"] = 1
	item.Data["b"] = 2

	NewDebugLogOutput(&buf).WithIncludeSource(true).LogItem(ctx, item)
	assert.Equal(t, "*** PROCESS LINE [4-5]: Metadata: map[a:1] - Data: map[b:2] - Line: \"text\" - Source: \"source\"\n",
		buf.String())
}
