package bluno

// // Build profile on first run
// prof, err := b.Client.DiscoverProfile(true)
// if err != nil {
// 	log.Println("ERR-1", err)
// } else {
// 	log.Println("OK-1", prof)
// }

// // Isolate the characteristic
// for _, s := range prof.Services {
// 	for _, c := range s.Characteristics {
// 		if c.CCCD != nil {
// 			log.Println("KNN", c.UUID.String(), c.CCCD.Handle, c.CCCD.Property, c.CCCD.UUID.String())
// 		}
// 	}
// }
// characteristic := prof.FindCharacteristic(ble.NewCharacteristic(charUUID[0]))

// d, err := b.Client.DiscoverDescriptors([]ble.UUID{}, c[1])
// if err != nil {
// 	log.Println("ERR0", err)
// } else {
// 	log.Println("OK0")
// }
// log.Println("d", d)

// go func(ch chan []byte, eCh chan bool) {
// 	log.Printf("repeat")
// 	var msg []byte

// 	msg, err := b.Client.ReadCharacteristic(characteristic)
// 	if err != nil {
// 		if commsintconfig.DebugMode {
// 			log.Printf("client_incoming_msg_err|addr=%s|err=%s", b.Address, err)
// 		}
// 		eCh <- true
// 	} else {
// 		log.Printf("client_incoming_msg_success|addr=%s|length=%d|msg=%s", b.Address, len(msg), string(msg))
// 		fmt.Printf("        Value         %x | %q\n", msg, msg)
// 		ch <- msg
// 	}
// }(msgCh, errorCh)
