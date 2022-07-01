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

func mergeSameChart(batch []*Chart) []*Chart {
	var ans []*Chart

	for i := 0; i < len(batch); i++ {
		if i+1 < len(batch) &&
			batch[i].Symbol == batch[i+1].Symbol &&
			batch[i].Resolution == batch[i+1].Resolution &&
			len(batch[i].T) == 1 &&
			len(batch[i+1].T) == 1 &&
			batch[i].T[0] < batch[i+1].T[0] {
			ans = append(ans, &Chart{
				Symbol:     batch[i].Symbol,
				Resolution: batch[i].Resolution,
				O:          append(batch[i].O, batch[i+1].O...),
				H:          append(batch[i].H, batch[i+1].H...),
				L:          append(batch[i].L, batch[i+1].L...),
				C:          append(batch[i].C, batch[i+1].C...),
				V:          append(batch[i].V, batch[i+1].V...),
				T:          append(batch[i].T, batch[i+1].T...),
			})
			i++
		} else {
			ans = append(ans, batch[i])
		}
	}

	return ans
}
