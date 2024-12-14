package panyl

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLineProvider_IoReader(t *testing.T) {
	ctx := context.Background()

	lp := NewReaderLineProvider(strings.NewReader("first\nsecond\nthird\n"), DefaultScannerBufferSize)
	ct := 0
	for lp.Scan(ctx) {
		switch ct {
		case 0:
			assert.Equal(t, "first", lp.Line().(string))
		case 1:
			assert.Equal(t, "second", lp.Line().(string))
		case 2:
			assert.Equal(t, "third", lp.Line().(string))
		}
		ct++
	}
	assert.NoError(t, lp.Err())
	assert.Equal(t, 3, ct, "should have 3 lines")
}

func TestLineProvider_Static(t *testing.T) {
	ctx := context.Background()

	lp := NewStaticLineProvider([]interface{}{"first", "second", "third"})
	ct := 0
	for lp.Scan(ctx) {
		switch ct {
		case 0:
			assert.Equal(t, "first", lp.Line().(string))
		case 1:
			assert.Equal(t, "second", lp.Line().(string))
		case 2:
			assert.Equal(t, "third", lp.Line().(string))
		}
		ct++
	}
	assert.NoError(t, lp.Err())
	assert.Equal(t, 3, ct, "should have 3 lines")
}
