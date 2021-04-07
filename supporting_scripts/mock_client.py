import json
import socket
import os
import threading
import datetime
import csv
import queue
import pytz

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

    description = json.loads(client.recv(4096))
    threads = []
    queues = {}

    for bluno_desc in description.get("bluno_mapping"):
        num = bluno_desc.get("num", -1)
        username = bluno_desc.get("username", "unknown user")

        mq = queue.Queue()
        queues[num] = mq

        threads.append(threading.Thread(target=write_data_to_csv, args=(username, mq)))
        threads[-1].daemon = True
        threads[-1].start()

    while True:
        msg = client.recv(4096)

        # connection has closed
        if len(msg) == 0:
            print("Connection closed. Exiting...")
            break

        try:
            parsed = json.loads(msg)
            if parsed.get('timestamps') is not None:
                print(parsed)
            elif parsed.get('packets') is not None:
                for item in parsed.get('packets'):
                    queues[item.get('bluno')].put(item)
        except json.JSONDecodeError:
            pass
 
    client.close()

    for val in queues.values():
        val.put(None)

    for t in threads:
        t.join()


"""
write_data_to_csv is used to write
data to a specified csv file

to be run within its own thread
"""
def write_data_to_csv(user_name, mq):
    tz = pytz.timezone('Asia/Singapore')
    init_time = (tz.localize(datetime.datetime.utcnow()) + datetime.timedelta(hours=8)).strftime("%b %d %Y %H:%M:%S")
    f_name = f"{init_time}_{user_name}_inprogress.csv"

    with open(f_name, mode='w') as csv_file:
        sensor_writer = csv.writer(csv_file)
        first_row = True
        print(f"writing to csv file: {f_name}")

        start = None
        counts = {-1:0, 0:0, 1: 0}

        while True:
            item = mq.get()

            # connection has closed
            if item is None:
                break

            if item.get('muscle_sensor') is True:
                continue

            if first_row:
                start = datetime.datetime.utcnow()
                sensor_writer.writerow(item.keys())
                first_row = False

            counts[item.get('movement')] += 1

            sensor_writer.writerow(item.values())
   
        if start is None:
            delta = 0.0
        else:
            delta = (datetime.datetime.utcnow() - start).total_seconds()

    print("bluno", user_name, "counts", counts)
    os.rename(f_name, f"{init_time}_{user_name}_{delta}sec.csv")


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
    