package constants

import (
	"time"

	"github.com/CG4002-AY2021S2-B16/comms-int/bluno"
)

// IncomingNotifSock refers to the unix socket's pathname used for notifications
// This needs to be mapped to a suitable location outside of the container
var IncomingNotifSock string = "/tmp/www/comms/notif.sock"

// OutgoingDataSock refers to the unix socket's pathname used for sending outgoing data
// This needs to be mapped to a suitable location outside of the container
var OutgoingDataSock string = "/tmp/www/comms/data.sock"

// UpstreamCheckFreq decides how often upstream is checked for incoming messages
var UpstreamCheckFreq time.Duration = 300 * time.Millisecond

// UpstreamResumeMsg is the expected indication to resume the application
var UpstreamResumeMsg string = "resume"

// UpstreamPauseMsg is the expected indication to pause the application
var UpstreamPauseMsg string = "pause"

// UpstreamNotifBufferSize refers to the max number of bytes to be read in for an incoming notif
var UpstreamNotifBufferSize int = 1000

// Bluno configurations
var blunoOne bluno.Bluno = bluno.Bluno{
	Address: "80:30:DC:E9:1C:34",
	Name:    "BlunoOne",
}
