package bluno

import (
	"log"

	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
)

// ReconcilePacket attempts to recombine and parse packets that have been fragmented into two pieces, using a LIFO strategy
// It works on a best effort basis and can allow false positives through (albeit unlikely based on observations)
// TODO: Can be enhanced once error detection is also implemented
func (b *Bluno) ReconcilePacket(curr []byte) commsintconfig.Packet {
	// Successful reconciliation if buffer is non-empty, len(prev) + len(curr) == 20, and curr.first is an invalid packetType
	if b.Buffer.Len() > 0 {
		prev := b.Buffer.Remove(b.Buffer.Back()).([]byte)
		if determinePacketType(curr) == commsintconfig.Invalid && (len(prev)+len(curr)) == commsintconfig.ExpectedPacketSize {
			if commsintconfig.DebugMode {
				log.Printf("reconcile_packet|success|% x", append(prev, curr...))
			}
			b.PacketsReconciled++
			return constructPacket(append(prev, curr...))
		}
		b.Buffer.PushBack(prev)
	}
	// Otherwise if the packet appears to be valid, it is likely spliced into 2 pieces
	// Add to buffer if it appears that the currently considered packet can be the first one
	if determinePacketType(curr) != commsintconfig.Invalid {
		b.Buffer.PushBack(curr)
	}

	// Return a default invalid packet (to be discarded unless stored in buffer)
	return commsintconfig.Packet{Type: commsintconfig.Invalid}
}
