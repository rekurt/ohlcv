package domain

import (
	"context"
	"time"
)

func Microbatching(ctx context.Context, chartStream chan *Chart, maxBatchSize int) chan []*Chart {
	batchStream := make(chan []*Chart)
	go func() {
		defer close(batchStream)
		for {
			select {
			case <-ctx.Done():
				return
			case chart, ok := <-chartStream:
				if !ok {
					return
				}
				batch := []*Chart{chart}
			loop:
				for len(batch) < maxBatchSize {
					select {
					case <-ctx.Done():
						return
					case chart, ok := <-chartStream:
						if !ok {
							break loop
						}
						batch = append(batch, chart)
					case <-time.After(time.Second * 2):
						break loop
					}
				}
				select {
				case <-ctx.Done():
					return
				case batchStream <- batch:
				}
			}
		}
	}()
	return batchStream
}
