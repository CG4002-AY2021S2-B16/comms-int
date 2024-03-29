package upstream

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
	"github.com/CG4002-AY2021S2-B16/comms-int/constants"
)

// IOHandler is a wrapper for a IO
type IOHandler struct {
	sync.Mutex
	ReadChan          chan Instruction
	WriteRoutine      func(p *[]commsintconfig.Packet)
	WriteTimestamp    func(uint64)
	WriteBlunoMapping func()
	sent              int
	received          int
}

// Instruction is an incoming message from upstream
type Instruction struct {
	Cmd  string `json:"cmd"`
	Data uint64 `json:"t_one"`
}

// NewUpstreamConnection creates and returns a new wrapper for input and output sockets
func NewUpstreamConnection() (*IOHandler, error) {
	os.Remove(constants.OutgoingDataSock)
	os.Remove(constants.IncomingNotifSock)

	inc := make(chan Instruction)

	ioh := &IOHandler{
		ReadChan: inc,
		sent:     0,
		received: 0,
	}

	outgoingListener, err := net.Listen("unix", constants.OutgoingDataSock)
	if err != nil {
		log.Fatalf("upstream|establishing_data_sock|err=%s", err)
		return &IOHandler{}, err
	}

	incomingListener, err := net.Listen("unix", constants.IncomingNotifSock)
	if err != nil {
		log.Fatalf("upstream|establishing_notif_sock|err=%s", err)
		return &IOHandler{}, err
	}

	outgoing, err := outgoingListener.Accept()
	log.Printf("upstream|outgoing_listener_accept")
	if err != nil {
		log.Fatalf("upstream|outgoing_listener_accept|err=%s", err)
		return &IOHandler{}, err
	}
	ioh.WriteRoutine = writeRoutine(outgoing)
	ioh.WriteTimestamp = writeTimestamps(outgoing)
	ioh.WriteBlunoMapping = writeBlunoMapping(outgoing)

	incoming, err := incomingListener.Accept()
	log.Printf("upstream|incoming_listener_accept")
	if err != nil {
		log.Printf("upstream|incoming_listener_accept|err=%s", err)
		return &IOHandler{}, err
	}

	// Start up a read goroutine
	go readRoutine(incoming, inc)

	return ioh, nil
}

// writeBlunoMapping sends names associated with blunos that are expected to connect
func writeBlunoMapping(oConn net.Conn) func() {
	oConn.SetWriteDeadline(time.Time{}) // Set to zero (no timeout)

	type blunoMapEntry struct {
		Num  uint8  `json:"num"`
		Name string `json:"username"`
	}

	type blunoMapping struct {
		Mapping []blunoMapEntry `json:"bluno_mapping"`
	}

	return func() {
		var bm blunoMapping

		for _, b := range constants.RetrieveValidBlunos() {
			bme := blunoMapEntry{Num: b.Num, Name: fmt.Sprintf("%s_%d", b.User, b.Num)}
			bm.Mapping = append(bm.Mapping, bme)
		}

		bmj, err := json.Marshal(bm)
		if err != nil {
			log.Printf("upstream|write_bluno_count_marshal|err=%s", err)
		} else {
			_, err := oConn.Write(bmj)
			if err != nil {
				log.Printf("upstream|write_bluno_count|err=%s", err)
				return
			}
		}
	}
}

// writeTimestamps sends t2, t3 for each active bluno when a time sync request is received
func writeTimestamps(oConn net.Conn) func(t_one uint64) {
	oConn.SetWriteDeadline(time.Time{}) // Set to zero (no timeout)

	type timestamp struct {
		OriginalTOne uint64 `json:"t_one"`
		BlunoNum     uint8  `json:"num"`
		Ttwo         int64  `json:"t_two"`
		Tthree       int64  `json:"t_three"`
	}

	type blunoTimestamps struct {
		Timestamps []timestamp `json:"timestamps"`
	}

	var bt blunoTimestamps = blunoTimestamps{Timestamps: make([]timestamp, 0)}

	return func(t_one uint64) {
		bt.Timestamps = nil
		for _, b := range constants.RetrieveValidBlunos() {
			if b.Num > 3 { // this is hardcoded to prevent EMG timestamps
				continue
			}

			if b.HandshakedAt.IsZero() {
				bt.Timestamps = append(bt.Timestamps, timestamp{
					OriginalTOne: t_one,
					BlunoNum:     b.Num,
					Ttwo:         b.HandShakeInit.UnixNano() / int64(time.Millisecond),
					Tthree:       b.HandshakedAt.UnixNano() / int64(time.Millisecond),
				})
			} else {
				displacement := time.Now().Sub(b.HandshakedAt)
				bt.Timestamps = append(bt.Timestamps, timestamp{
					OriginalTOne: t_one,
					BlunoNum:     b.Num,
					Ttwo:         b.HandShakeInit.UnixNano()/int64(time.Millisecond) + displacement.Milliseconds(),
					Tthree:       b.HandshakedAt.UnixNano()/int64(time.Millisecond) + displacement.Milliseconds(),
				})
			}
		}
		msg, err := json.Marshal(bt)
		if err != nil {
			log.Printf("upstream|write_timestamps_marshal|err=%s", err)
		} else {
			_, err := oConn.Write(msg)
			if err != nil {
				log.Printf("upstream|write_timestamp|err=%s", err)
				return
			}
		}

	}

}

// writeRoutine listens for incoming write requests from the application
// and writes them out to the unix socket
func writeRoutine(oConn net.Conn) func(p *[]commsintconfig.Packet) {
	oConn.SetWriteDeadline(time.Time{}) // Set to zero (no timeout)
	return func(p *[]commsintconfig.Packet) {
		type packets struct {
			Packets *[]commsintconfig.Packet `json:"packets"`
		}

		msg, err := json.Marshal(packets{Packets: p})
		if err != nil {
			log.Printf("upstream|write_routine_marshal|err=%s", err)
		} else {
			_, err := oConn.Write(msg)
			if err != nil {
				log.Printf("upstream|write_routine|err=%s", err)
				return
			}
		}
	}
}

// readRoutine listens to the incoming notifications and sends them
// out to the main application via the provided channel
func readRoutine(iConn net.Conn, comm chan Instruction) {
	iConn.SetReadDeadline(time.Time{}) // Set to zero (no timeout)
	var i Instruction
	b := make([]byte, constants.UpstreamNotifBufferSize)

	for {
		num, err := iConn.Read(b)
		log.Printf("upstream|read_routine|string=%s", string(b))
		if err != nil {
			log.Printf("upstream|read_routine|err=%s", err)
			return
		}

		err = json.Unmarshal(b[:num], &i)
		if err != nil {
			log.Printf("upstream|read_routine_unmarshal|err=%s|data=%s", err, string(b))
		} else {
			comm <- i
		}
	}
}
