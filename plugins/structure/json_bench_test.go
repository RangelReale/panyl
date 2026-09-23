package structure_test

import (
	"context"
	"strings"
	"testing"

	"github.com/RangelReale/panyl/v2"
	"github.com/RangelReale/panyl/v2/plugins/structure"
)

// BenchmarkJSON_Processor processes a log that is mostly plain text lines, with some single and multi-line JSON.
func BenchmarkJSON_Processor(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString("2024-01-01 12:00:00 INFO some plain text log line that is not json\n")
		if i%10 == 0 {
			sb.WriteString(`{"level":"info","msg":"single line json"}` + "\n")
		}
		if i%50 == 0 {
			sb.WriteString("{\n  \"level\": \"info\",\n  \"msg\": \"multi line json\"\n}\n")
		}
	}
	input := sb.String()

	ctx := context.Background()
	p := panyl.NewProcessor(panyl.WithPlugins(structure.JSON{}))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := p.Process(ctx, strings.NewReader(input), &panyl.OutputNull{}); err != nil {
			b.Fatal(err)
		}
	}
}
