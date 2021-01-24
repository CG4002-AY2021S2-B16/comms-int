package constants

import (
	"github.com/rssujay/golang-ble-test/bluno"
)

// RetrieveValidBlunos retrieves the list of Blunos this central should be concerned with
// connecting to
func RetrieveValidBlunos() []bluno.Bluno {
	return []bluno.Bluno{
		blunoOne,
	}
}
