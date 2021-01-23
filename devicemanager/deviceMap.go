package devicemanager

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-ble/ble"
)

// Device represents a BLE device
type Device struct {
	Address     string     `json:"address"`
	Detected    time.Time  `json:"detected"`
	Connectable bool       `json:"connectable"`
	Services    []ble.UUID `json:"services"`
	Name        string     `json:"name"`
	RSSI        int        `json:"rssi"`
}

// DeviceMap is a concurrency-safe store of <MAC address: Device object>
type DeviceMap struct {
	sync.RWMutex
	dm map[string]Device
}

// GetDevice retrieves an entry from the provided device map if one exists,
// otherwise it returns an empty device
func (d *DeviceMap) GetDevice(addr string) Device {
	d.RLock()
	defer d.RUnlock()
	device, exist := d.dm[addr]
	if exist {
		return device
	}
	return Device{}
}

// SetDevice creates an entry in the provided device map
func (d *DeviceMap) SetDevice(addr string, device Device) {
	d.Lock()
	defer d.Unlock()
	d.dm[addr] = device
}

// PrintEntries can be used for debugging entries found and stored during scan
func (d *DeviceMap) PrintEntries() {
	d.RLock()
	defer d.RUnlock()

	for k, v := range d.dm {
		fmt.Println("Key: ", k, " Entry: ", v)
	}
}
