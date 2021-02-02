import json
import socket
import os
import threading
import time

NOTIF_SOCK_PATH = "/tmp/www/comms/notif.sock"
DATA_SOCK_PATH = "/tmp/www/comms/data.sock"

RESUME_CMD = "resume"
PAUSE_CMD = "pause"


"""
To be run after golang server has started

Type 1 to pause, 0 to resume
"""
def control_ble():
    client = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    os.chmod(NOTIF_SOCK_PATH, 0o777)
    client.connect(NOTIF_SOCK_PATH)

    while True:
        user_input = int(input())
        if user_input == 0:
            data = json.dumps({ "cmd": RESUME_CMD })
        else:
            data = json.dumps({ "cmd": PAUSE_CMD })
        
        print("Data = ", data)
        client.send(data.encode('utf-8'))
    client.close()


"""
Read BLE output
prints out data
"""
def read_ble_data():
    client = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    os.chmod(NOTIF_SOCK_PATH, 0o777)
    client.connect(DATA_SOCK_PATH)

    while True:
        # the logic needs to be much more complex here
        print("Waiting...")
        print(client.recv(4096))

    client.close()


def start():
    controller_thread = threading.Thread(target=control_ble)
    read_thread = threading.Thread(target=read_ble_data)

    controller_thread.setDaemon(True)
    read_thread.setDaemon(True)
    controller_thread.start()
    read_thread.start()

    controller_thread.join()
    read_thread.join()

if __name__ == "__main__":
    start()
    