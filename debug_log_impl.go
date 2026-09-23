package panyl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/RangelReale/panyl/v2/util"
)

// DebugLogOutput writes log to the passed io.Writer
type DebugLogOutput struct {
	w             io.Writer
	IncludeSource bool
}

var _ DebugLog = DebugLogOutput{}

// NewDebugLogOutput creates a DebugLogOutput using an io.Writer.
func NewDebugLogOutput(w io.Writer) DebugLogOutput {
	return DebugLogOutput{w: w}
}

// NewStdDebugLogOutput creates a DebugLogOutput using an os.Stdout.
func NewStdDebugLogOutput() DebugLogOutput {
	return DebugLogOutput{w: os.Stdout}
}

// WithIncludeSource returns a copy that also logs the raw source lines and the Item.Source of each item.
func (l DebugLogOutput) WithIncludeSource(includeSource bool) DebugLogOutput {
	l.IncludeSource = includeSource
	return l
}

func (l DebugLogOutput) LogSourceLine(ctx context.Context, n int, line, rawLine string) {
	if l.IncludeSource && rawLine != "" && rawLine != line {
		_, _ = fmt.Fprintf(l.w, "@@@ SOURCE LINE [%d]: '%s' (raw: '%s') @@@\n", n, line,
			util.DoAnsiEscapeString(rawLine))
		return
	}
	_, _ = fmt.Fprintf(l.w, "@@@ SOURCE LINE [%d]: '%s' @@@\n", n, line)
}

func (l DebugLogOutput) LogItem(ctx context.Context, item *Item) {
	var lineno string
	if item.LineCount > 1 {
		lineno = fmt.Sprintf("[%d-%d]", item.LineNo, item.LineNo+item.LineCount-1)
	} else {
		lineno = fmt.Sprintf("[%d]", item.LineNo)
	}

	var buf bytes.Buffer

	if len(item.Metadata) > 0 {
		_, _ = fmt.Fprintf(&buf, "Metadata: %+v", item.Metadata)
	}
	if len(item.Data) > 0 {
		if buf.Len() > 0 {
			_, _ = buf.WriteString(" - ")
		}
		_, _ = fmt.Fprintf(&buf, "Data: %+v", item.Data)
	}

	if len(item.Line) > 0 {
		if buf.Len() > 0 {
			_, _ = buf.WriteString(" - ")
		}
		_, _ = fmt.Fprintf(&buf, "Line: \"%s\"", item.Line)
	}

	if l.IncludeSource && len(item.Source) > 0 {
		if buf.Len() > 0 {
			_, _ = buf.WriteString(" - ")
		}
		_, _ = fmt.Fprintf(&buf, "Source: \"%s\"", util.DoAnsiEscapeString(item.Source))
	}

	_, _ = fmt.Fprintf(l.w, "*** PROCESS LINE %s: %s\n", lineno, buf.String())
}
