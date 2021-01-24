package constants

import (
	"github.com/CG4002-AY2021S2-B16/comms-int/bluno"
)

// RetrieveValidBlunos retrieves the list of Blunos this central should be concerned with
// connecting to
func RetrieveValidBlunos() []bluno.Bluno {
	return []bluno.Bluno{
		blunoOne,
	}
}
