package bluno

import (
	"context"
	"fmt"
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
func (b *Bluno) Connect() ble.Client {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	client, err := ble.Dial(ctx, ble.NewAddr(b.Address))
	if err != nil {
		return nil
	}
	defer client.CancelConnection()
	fmt.Println("Done", client.Addr().String(), client.ReadRSSI())
	return client
}
