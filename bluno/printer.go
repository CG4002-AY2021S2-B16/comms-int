package bluno

import (
	"log"

	"github.com/fatih/color"

	"github.com/CG4002-AY2021S2-B16/comms-int/commsintconfig"
)

func printPacket(p *commsintconfig.Packet, resp []byte) {
	cf := color.New(color.FgCyan).SprintFunc()

	switch p.BlunoNumber {
	case 1:
		cf = color.New(color.FgRed).SprintFunc()
	case 2:
		cf = color.New(color.FgGreen).SprintFunc()
	case 3:
		cf = color.New(color.FgBlue).SprintFunc()
	case 4:
		cf = color.New(color.FgYellow).SprintFunc()
	default:
		cf = color.New(color.FgMagenta).SprintFunc()
	}

	log.Printf("packet processed %v for resp [ % X ]\n", cf(p), resp)
}
