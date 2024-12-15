package panyl

import (
	"bufio"
	"context"
	"fmt"
	"io"
)

// ReaderLineProvider is a LineProvider that reads from an io.Reader
type ReaderLineProvider struct {
	scanner *bufio.Scanner
}

// NewReaderLineProvider is a LineProvider that reads from an io.Reader
func NewReaderLineProvider(r io.Reader, bufferSize int) LineProvider {
	scanner := bufio.NewScanner(r)
	if bufferSize > 0 {
		// adjust the scanner capacity
		buf := make([]byte, bufferSize)
		scanner.Buffer(buf, bufferSize)
	}
	return &ReaderLineProvider{scanner: scanner}
}

func (r *ReaderLineProvider) Err() error {
	return r.scanner.Err()
}

func (r *ReaderLineProvider) Line() interface{} {
	return r.scanner.Text()
}

func (r *ReaderLineProvider) Scan(ctx context.Context) bool {
	return r.scanner.Scan()
}

// StaticLineProvider is a LineProvider that reads from a memory array
type StaticLineProvider struct {
	currentLine int
	lines       []interface{}
	err         error
}

// NewStaticLineProvider is a LineProvider that reads from a memory array
func NewStaticLineProvider(lines []interface{}) LineProvider {
	return &StaticLineProvider{currentLine: -1, lines: lines, err: nil}
}

func (r *StaticLineProvider) Err() error {
	return r.err
}

func (r *StaticLineProvider) Line() interface{} {
	if r.err != nil {
		return nil
	}

	if r.currentLine >= 0 && r.currentLine < len(r.lines) {
		return r.lines[r.currentLine]
	}
	r.err = fmt.Errorf("invalid index being read: %d", r.currentLine)
	return nil
}

func (r *StaticLineProvider) Scan(ctx context.Context) bool {
	if r.err != nil {
		return false
	}

	if r.currentLine >= len(r.lines)-1 {
		return false
	}
	r.currentLine++
	return true
}
