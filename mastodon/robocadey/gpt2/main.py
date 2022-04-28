#!/usr/bin/env python3

import gpt_2_simple as gpt2
import json
import os
import socket
import sys
from datetime import datetime

sess = gpt2.start_tf_sess()
gpt2.load_gpt2(sess, run_name='run1')

SYSTEMD_FIRST_SOCKET_FD = 3
sock = socket.fromfd(SYSTEMD_FIRST_SOCKET_FD, socket.AF_UNIX, socket.SOCK_STREAM)

sock.listen(1)

while True:
    connection, client_address = sock.accept()
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
    connection.close()

sock.close()
