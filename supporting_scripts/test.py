import threading
from time import sleep
from bluepy.btle import Scanner, DefaultDelegate, Peripheral, BTLEDisconnectError

BLUNO_MACS = ["80:30:dc:e9:1c:34"]
connected = {}

SERVICE_UUID = "0000dfb0-0000-1000-8000-00805f9b34fb"
CHARACTERISTIC_UUID = "0000dfb1-0000-1000-8000-00805f9b34fb"

class NotificationDelegate(DefaultDelegate):
    def __init__(self, addr):
        super().__init__()
        self.addr = addr

    def handleNotification(self, cHandle, data):
        try:
            print(self.addr, data, type(data))
        except Exception as e:
            print("error_handling_notif", e)


class ConnectionHandlerThread(threading.Thread):
    def __init__(self, addr):
        super().__init__()
        self.addr = addr
        self.is_connected = True
        self.connection = None
        self.handshake = False

    def run(self):
        print('Thread started', self.addr)
        self.connection = connected[self.addr]
        self.connection.withDelegate(NotificationDelegate(self.addr))
        
        service = self.connection.getServiceByUUID(SERVICE_UUID)
        characteristics = service.getCharacteristics()
        characteristic = characteristics[0]

        while True:
            if self.addr in connected:
                try:
                    if self.handshake:
                        self.connection.waitForNotifications(30)
                    else:
                        characteristic.write(("A").encode())
                        print("Handshake attempt made")
                except BTLEDisconnectError as e:
                    print("Disconnected", self.addr, e)
                    del connected[self.addr]
                    self.connection.disconnect()
            else:
                self.handshake = False
                
                rc_attempt = False
                while not rc_attempt:
                    try:
                        bluno = Peripheral(self.addr)
                        self.connection = bluno
                        self.connection.withDelegate(NotificationDelegate(self.addr))
                        service = self.connection.getServiceByUUID(SERVICE_UUID)
                        characteristic = service.getCharacteristics()[0]

                        connected[self.addr] = self.connection
                        sleep(1.5)
                        rc_attempt = True
                    except Exception:
                        continue
            sleep(2)

def start():
    scanner = Scanner(0)
    devices = scanner.scan(2)

    for device in devices:
        if device.addr in BLUNO_MACS:
            try:
                bluno = Peripheral(device.addr)
                connected[device.addr] = bluno

                t = ConnectionHandlerThread(device.addr)
                t.daemon = True
                t.start()
            except Exception as e:
                print("Unable to connect to device", e)



if __name__ == "__main__":
    start()
    while True:
        sleep(1)
