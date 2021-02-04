package upstream

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
)

// OutputBuffer acts as a buffer for outgoing packets in the form of json data
type OutputBuffer struct {
	sync.Mutex
	L           *list.List
	enqueueChan chan commsintconfig.Packet
}

// CreateOutputBuffer initializes and returns an output buffer
func CreateOutputBuffer() *OutputBuffer {
	return &OutputBuffer{
		L:           list.New(),
		enqueueChan: make(chan commsintconfig.Packet),
	}
}

// EnqueueBuffer takes in a packet and sends it to the channel, in order to maintain non-blocking nature of calling func
func (o *OutputBuffer) EnqueueBuffer(c commsintconfig.Packet) {
	o.enqueueChan <- c
}

// EnqueueChannelProcessor listens to the enqueue channel and adds it to the buffer.
// This should be run within a permanent goroutine
func (o *OutputBuffer) EnqueueChannelProcessor(ctx context.Context) {
	for {
		select {
		case p := <-o.enqueueChan:
			o.Lock()
			o.L.PushBack(p)
			o.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// DequeueProcessor periodically wakes up to create output and send it over to the ext comms interface
// This should be run within a permanent goroutine
func (o *OutputBuffer) DequeueProcessor(ctx context.Context, us *IOHandler) {
	t := time.NewTicker(commsintconfig.OutputDequeueInterval)

	for {
		select {
		case <-t.C:
			o.Lock()
			if o.L.Len() >= commsintconfig.OutputSize {
				var arr []commsintconfig.Packet

				for i := 1; i <= commsintconfig.OutputSize; i++ {
					arr = append(arr, o.L.Remove(o.L.Front()).(commsintconfig.Packet))
				}
				go us.WriteRoutine(&arr)
			}
			o.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
