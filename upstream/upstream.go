package upstream

import (
	"encoding/json"
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

	ReadChan  chan string
	WriteChan chan commsintconfig.Packet
	sent      int
	received  int
}

// Instruction is an incoming message from upstream
type Instruction struct {
	Cmd string `json:"cmd"`
}

// NewUpstreamConnection creates and returns a new wrapper for input and output sockets
func NewUpstreamConnection() *IOHandler {
	os.Remove(constants.OutgoingDataSock)
	os.Remove(constants.IncomingNotifSock)

	out := make(chan commsintconfig.Packet)
	inc := make(chan string)

	ioh := &IOHandler{
		ReadChan:  inc,
		WriteChan: out,
		sent:      0,
		received:  0,
	}

	outgoingListener, err := net.Listen("unix", constants.OutgoingDataSock)
	if err != nil {
		log.Fatalf("upstream|establishing_data_sock|err=%s", err)
	}

	incomingListener, err := net.Listen("unix", constants.IncomingNotifSock)
	if err != nil {
		log.Fatalf("upstream|establishing_notif_sock|err=%s", err)
	}

	go func() {
		defer outgoingListener.Close()
		for {
			outgoing, err := outgoingListener.Accept()
			log.Printf("upstream|outgoing_listener_accept")
			if err != nil {
				log.Printf("upstream|outgoing_listener_accept|err=%s", err)
				return
			}
			go writeRoutine(outgoing, out)
		}
	}()

	go func() {
		defer incomingListener.Close()

		for {
			incoming, err := incomingListener.Accept()
			log.Printf("upstream|incoming_listener_accept")
			if err != nil {
				log.Printf("upstream|incoming_listener_accept|err=%s", err)
				return
			}
			go readRoutine(incoming, inc)
		}
	}()

	return ioh
}

// writeRoutine listens for incoming write requests from the application
// and writes them out to the unix socket
func writeRoutine(oConn net.Conn, comm chan commsintconfig.Packet) {
	oConn.SetWriteDeadline(time.Time{}) // Set to zero (no timeout)
	for {
		select {
		case p := <-comm:
			msg, err := json.Marshal(p)
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
}

// readRoutine listens to the incoming notifications and sends them
// out to the main application via the provided channel
func readRoutine(iConn net.Conn, comm chan string) {
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
			comm <- i.Cmd
		}
	}
}
