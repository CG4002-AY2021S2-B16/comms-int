import json
import socket
import os


NOTIF_SOCK_PATH = "/tmp/www/comms/notif.sock"
DATA_SOCK_PATH = "/tmp/www/comms/data.sock"

RESUME_CMD = "resume"
PAUSE_CMD = "pause"


"""
To be run after golang server has started

type 1 to pause, 0 to resume
"""
client = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
os.chmod(NOTIF_SOCK_PATH, 0o777)
client.connect(NOTIF_SOCK_PATH)

while True:
    user_input = int(input())
    if user_input == 0:
        data = json.dumps({ "cmd": RESUME_CMD })
    else:
        data = json.dumps({ "cmd": PAUSE_CMD })
    
    print("Data=", data)
    client.send(data.encode('utf-8'))