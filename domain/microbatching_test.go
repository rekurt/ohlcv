package domain

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMicrobatching(t *testing.T) {
	t.Parallel()
	t.Run("many full batches and 1 short", func(t *testing.T) {
		chartStream := make(chan *Chart)
		maxBatchSize := 13
		batchStream := Microbatching(context.Background(), chartStream, maxBatchSize)

		go func() {
			defer close(chartStream)
			for i := 0; i < 2378; i++ {
				chartStream <- &Chart{}
			}
		}()

		var batches [][]*Chart
		for charts := range batchStream {
			batches = append(batches, charts)
		}
		assert.Len(t, batches, 183)
		var i int
		for ; i < 182; i++ {
			assert.Len(t, batches[i], maxBatchSize)
		}
		assert.Len(t, batches[i], 12)
	})
	t.Run("2 seconds wait", func(t *testing.T) {
		chartStream := make(chan *Chart)
		maxBatchSize := 13
		batchStream := Microbatching(context.Background(), chartStream, maxBatchSize)

		go func() {
			for i := 0; i < 3; i++ {
				chartStream <- &Chart{}
			}
		}()

		started := time.Now()
		<-batchStream
		assert.InDelta(t, time.Second*2, time.Since(started), float64(time.Millisecond*200))
	})
}
