package panyl

import (
	"bytes"
	"fmt"
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
	W io.Writer
}

func NewLogOutput(w io.Writer) *LogOutput {
	return &LogOutput{W: w}
}

func NewStdLogOutput() *LogOutput {
	return &LogOutput{W: os.Stdout}
}

func (l LogOutput) LogSourceLine(n int, line, rawLine string) {
	_, _ = fmt.Fprintf(l.W, "@@@ SOURCE LINE [%d]: '%s' @@@\n", n, line)
}

func (l LogOutput) LogProcess(p *Process) {
	var lineno string
	if p.LineCount > 1 {
		lineno = fmt.Sprintf("[%d-%d]", p.LineNo, p.LineNo+p.LineCount)
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

	_, _ = fmt.Fprintf(l.W, "*** PROCESS LINE %s: %s\n", lineno, buf.String())
}
