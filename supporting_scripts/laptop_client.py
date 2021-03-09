import os
import time
import json

import socket
import threading

import base64
from Crypto import Random
from Crypto.Cipher import AES
from Crypto.Util.Padding import pad
import sshtunnel

NOTIF_SOCK_PATH = "/tmp/www/comms/notif.sock"
DATA_SOCK_PATH = "/tmp/www/comms/data.sock"

RESUME_CMD = "resume"
PAUSE_CMD = "pause"
SUNFIRE_USERNAME = "tamelly"
SUNFIRE_PASSWORD = "cg4002b16"

class Client():
    def __init__(self, ip_addr, secret_key):
        super(Client, self).__init__()
        self.ip_addr = ip_addr
        self.secret_key = secret_key

        self.startBlunos = threading.Event()

        #Thread for laptop to pause/recv data from bluno
        self.ultra96toBlunoThread = threading.Thread(target=self.ultra96toBluno)
        #Thread for receiving data from bluno, and sending to ultra96
        self.blunoToUltra96Thread = threading.Thread(target=self.blunoToUltra96)
        
    def encrypt_message(self, plain_text):
        secret_key = bytes(str(self.secret_key), encoding='utf8') 
        iv = Random.new().read(AES.block_size)
        cipher = AES.new(secret_key, AES.MODE_CBC, iv)
        ciphertext = cipher.encrypt(pad(bytes(plain_text, encoding='utf8'), AES.block_size))
        encoded_message = base64.b64encode(iv + ciphertext)
        return encoded_message

    def decrypt_message(self, cipher_text):
        decoded_message = base64.b64decode(cipher_text)
        iv = decoded_message[:16]
        secret_key = bytes(str(self.secret_key), encoding="utf8") 

        cipher = AES.new(secret_key, AES.MODE_CBC, iv)
        decrypted_message = cipher.decrypt(decoded_message[16:]).strip()
        decrypted_message = decrypted_message.decode('utf8')
        return decrypted_message

    def send_ultra96(self, message):
        encrypted_message = self.encrypt_message(message)
        try:
            self.ultra96socket.sendall(encrypted_message)
            print(F"[LAPTOP -> ULTRA96] SENT: {message}")
        except Exception:
            print(F"[LAPTOP -> ULTRA96] FAILED TO SEND: {message}")
    
    # Handle commands from Ultra96. For now, only timestamp will be sent
    def ultra96toBluno(self):
        # Connection to send data to bluno
        self.blunosClient = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        os.chmod(NOTIF_SOCK_PATH, 0o777)
        self.blunosClient.connect(NOTIF_SOCK_PATH)

        while True:
            try: 
                data = self.ultra96socket.recv(1024)
                if data:
                    decrypted_message = self.decrypt_message(data)
                    print(F"[ULTRA96 -> LAPTOP] RECEIVED {decrypted_message}")
                    if "!T" in decrypted_message:
                        #send timestamp to bluno
                        self.send_blunos(data)
            except socket.timeout:
                pass
    
    def send_blunos(self, message):
        #Send 1 to pause, 0 to resume, others for timestamp
        if message == 0:
            data = json.dumps({ "cmd": RESUME_CMD })
            print("[LAPTOP -> BLUNO] RESUME receiving data from blunos")
        elif message == 1:
            data = json.dumps({ "cmd": PAUSE_CMD })
            print("[LAPTOP -> BLUNO] PAUSE receiving data from blunos")
        else:
            #clock synchro packet to send to bluno
            data = json.dumps({ message })
        self.blunosClient.send(data.encode('utf8'))
    
    # Handle data from bluno
    def blunoToUltra96(self):
        # Connection to receive data to bluno
        self.blunosServer = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        os.chmod(DATA_SOCK_PATH, 0o777)
        self.blunosServer.connect(DATA_SOCK_PATH)

        while True:
            data = self.blunosServer.recv(1024).decode('utf8')
            if data:
                parsed_data = json.loads(data)
                decrypted_message = self.decrypt_message(parsed_data)
                print(F"[BLUNO -> LAPTOP] Received {decrypted_message} from Blunos")
                # Send data to Ultra96
                self.send_ultra96(decrypted_message)

    def run(self):
        #Open tunnel to ultra96
        with sshtunnel.open_tunnel(
            ssh_address_or_host=('sunfire.comp.nus.edu.sg', 22),
            remote_bind_address=('137.132.86.239', 22),
            ssh_username=SUNFIRE_USERNAME,
            ssh_password=SUNFIRE_PASSWORD,
        ) as tunnel1:
            print('Connection to tunnel1 (sunfire.comp.nus.edu.sg:22) OK...')
            with sshtunnel.open_tunnel(
                ssh_address_or_host=('localhost', tunnel1.local_bind_port),
                remote_bind_address=('127.0.0.1', 22),
                ssh_username='xilinx',
                ssh_password='xilinx',
            ) as tunnel2:
                print('Connection to tunnel2 (137.132.86.239:22) OK...')
                tunnel2.start()

        #Start sending inputs to bluno thru relay laptop from Ultra96
        self.ultra96toBlunoThread.setDaemon(True)
        self.ultra96toBlunoThread.start()
        #Start receiving data from bluno then pass to ultra96
        self.blunoToUltra96Thread.setDaemon(True)
        self.blunoToUltra96Thread.start()

        # Connect to ultra96
        while True:
            try:
                #Start connection to ultra96 (may need to set flag if ultra96 not ready)
                self.ultra96socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                self.ultra96socket.settimeout(1)
                self.ultra96socket.connect(self.ip_addr)
                print("[SOCKET CREATED] LAPTOP CONNECTED TO ULTRA96")
                time.sleep(1)
            except ConnectionRefusedError:
                print("Can't connect")
            except Exception:
                pass


if __name__ == '__main__':
    client = Client(('localhost', 1235), '0000000000000000')
    client.run()                                                                