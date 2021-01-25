package bluno

import (
	"context"
	"log"
	"time"

	"github.com/go-ble/ble"
)

// Bluno represents a BLE device
type Bluno struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

// DefaultTimeout is the timeout used per connection
const DefaultTimeout time.Duration = 1 * time.Second

// Connect establishes a connection with the physical bluno
// **Remember to close client when done
func (b *Bluno) Connect() ble.Client {
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), DefaultTimeout))
	client, err := ble.Dial(ctx, ble.NewAddr(b.Address))
	if err != nil {
		log.Printf("client_connection|addr=%s|err=%s", b.Address, err)
		return nil
	}
	return client
}
