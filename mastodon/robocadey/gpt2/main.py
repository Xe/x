#!/usr/bin/env python3

import gpt_2_simple as gpt2
import json
import os
import socket
import sys
from datetime import datetime

sockpath = "/xe/gpt2/checkpoint/server.sock"

sess = gpt2.start_tf_sess()
gpt2.load_gpt2(sess, run_name='run1')

if os.path.exists(sockpath):
    os.remove(sockpath)

sock = socket.socket(socket.AF_UNIX)
sock.bind(sockpath)

print("Listening on", sockpath)
sock.listen(1)

while True:
    connection, client_address = sock.accept()
    try:
        print("generating shitpost")
        result = gpt2.generate(sess,
                    length=512,
                    temperature=0.8,
                    nsamples=1,
                    batch_size=1,
                    return_as_list=True,
                    top_p=0.9,
                    )[0].split("\n")[1:][:-1]
        print("shitpost generated")
        connection.send(json.dumps(result).encode())
    finally:
        connection.close()

server.close()
os.remove("/xe/gpt2/checkpoint/server.sock")
