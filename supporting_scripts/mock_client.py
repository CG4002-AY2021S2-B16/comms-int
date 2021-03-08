import json
import socket
import os
import threading
import datetime
import csv

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

    with open(f'data_{datetime.datetime.now()}.csv', mode='w') as csv_file:
        sensor_writer = csv.writer(csv_file)
        first_row = True
        print("Waiting...")

        while True:
            # the logic needs to be much more complex here
            d = client.recv(4096)
            
            # connection has closed
            if len(d) == 0:
                print("Connection closed. Exiting...")
                break

            try:
                parsed = json.loads(d)
                for item in parsed:
                    if first_row:
                        sensor_writer.writerow(item.keys())
                        first_row = False
                    sensor_writer.writerow(item.values())
            except json.JSONDecodeError:
                pass

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
    