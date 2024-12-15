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

// NewDebugLogOutput creates a DebugLogOutput using an io.Writer.
func NewDebugLogOutput(w io.Writer) *DebugLogOutput {
	return &DebugLogOutput{w: w}
}

// NewStdDebugLogOutput creates a DebugLogOutput using an os.StdOut.
func NewStdDebugLogOutput() *DebugLogOutput {
	return &DebugLogOutput{w: os.Stdout}
}

func (l DebugLogOutput) LogSourceLine(ctx context.Context, n int, line, rawLine string) {
	_, _ = fmt.Fprintf(l.w, "@@@ SOURCE LINE [%d]: '%s' @@@\n", n, line)
}

func (l DebugLogOutput) LogProcess(ctx context.Context, p *Process) {
	var lineno string
	if p.LineCount > 1 {
		lineno = fmt.Sprintf("[%d-%d]", p.LineNo, p.LineNo+p.LineCount-1)
	} else {
		lineno = fmt.Sprintf("[%d]", p.LineNo)
	}

	var buf bytes.Buffer

	if len(p.Metadata) > 0 {
		_, _ = buf.WriteString(fmt.Sprintf("Metadata: %+v", p.Metadata))
	}
	if len(p.Data) > 0 {
		if buf.Len() > 0 {
			_, _ = buf.WriteString(" - ")
		}
		_, _ = buf.WriteString(fmt.Sprintf("Data: %+v", p.Data))
	}

	if len(p.Line) > 0 {
		if buf.Len() > 0 {
			_, _ = buf.WriteString(" - ")
		}
		_, _ = buf.WriteString(fmt.Sprintf("Line: \"%s\"", p.Line))
	}

	if l.IncludeSource && len(p.Source) > 0 {
		if buf.Len() > 0 {
			_, _ = buf.WriteString(" - ")
		}
		_, _ = buf.WriteString(fmt.Sprintf("Source: \"%s\"", util.DoAnsiEscapeString(p.Source)))
	}

	_, _ = fmt.Fprintf(l.w, "*** PROCESS LINE %s: %s\n", lineno, buf.String())
}
