package panyl

import (
	"bytes"
	"fmt"
	"github.com/RangelReale/panyl/util"
	"io"
	"os"
)

// Log allows debugging each step of the processing
type Log interface {
	LogSourceLine(n int, line, rawLine string)
	LogProcess(p *Process)
}

// LogOutput writes log to the passed io.Writer
type LogOutput struct {
	w             io.Writer
	IncludeSource bool
}

func NewLogOutput(w io.Writer) *LogOutput {
	return &LogOutput{w: w}
}

func NewStdLogOutput() *LogOutput {
	return &LogOutput{w: os.Stdout}
}

func (l LogOutput) LogSourceLine(n int, line, rawLine string) {
	_, _ = fmt.Fprintf(l.w, "@@@ SOURCE LINE [%d]: '%s' @@@\n", n, line)
}

func (l LogOutput) LogProcess(p *Process) {
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
