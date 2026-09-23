package panyl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItemLines_LineAndSource(t *testing.T) {
	newItem := func(line string) *Item {
		return InitItem(WithInitLine(line), WithInitSource("src-"+line))
	}

	assert.Equal(t, "", ItemLines{}.Line())
	assert.Equal(t, "", ItemLines{}.Source())
	assert.Equal(t, "a", ItemLines{newItem("a")}.Line())
	assert.Equal(t, "src-a", ItemLines{newItem("a")}.Source())
	assert.Equal(t, "a\n\nc", ItemLines{newItem("a"), newItem(""), newItem("c")}.Line())
	assert.Equal(t, "src-a\nsrc-\nsrc-c", ItemLines{newItem("a"), newItem(""), newItem("c")}.Source())
}

func BenchmarkItemLines_Line(b *testing.B) {
	lines := ItemLines{}
	for i := 0; i < 20; i++ {
		lines = append(lines, InitItem(WithInitLine("some log line text")))
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := range lines {
			_ = lines[j:].Line()
		}
	}
}
