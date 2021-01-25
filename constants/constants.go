package constants

import (
	"github.com/CG4002-AY2021S2-B16/comms-int/bluno"
)

const defaultConnectionPriority uint8 = 0
const enhancedConnectionPriority uint8 = 1

// Bluno configurations
var blunoOne bluno.Bluno = bluno.Bluno{
	Address:            "80:30:DC:E9:1C:34",
	Name:               "BlunoOne",
	ConnectionPriority: defaultConnectionPriority,
}
