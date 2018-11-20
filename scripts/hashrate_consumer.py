#!/usr/bin/python3
"""
Quick and dirty example script on how to consume sentient-miner hashrates
(e.x. like the wallet app would). It prints the received current hash rates
over zmq as well as the historical hash rates written to a log file.

Prerequisites,
- Python 3
- `pip install pyzmq==17.1.2`

If from within docker continer you can run,
```
apt-get install python3-dev
curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py
python3 get-pip.py
pip install pyzmq==17.1.2
```

Running,
```
# SENTIENT_MINER_CURRENT_HASHRATE_ENDPOINT=tcp://localhost:5555 \
# SENTIENT_MINER_HASHRATES_LOG_PATH=../hashrates.log \
python3 -m hashrate_consumer.py
```
"""

import os
import zmq


def clearscreen():
    if os.name == 'nt':
        os.system('cls')
    else:
        os.system('clear')


endpoint = os.environ.get('SENTIENT_MINER_CURRENT_HASHRATE_ENDPOINT', 'tcp://127.0.0.1:5555')
log_path = os.environ.get('SENTIENT_MINER_HASHRATES_LOG_PATH', '../hashrates.log')

context = zmq.Context()
socket = context.socket(zmq.SUB)
socket.connect(endpoint)

socket.setsockopt(zmq.SUBSCRIBE, b'')
socket.setsockopt(zmq.LINGER, 0)

print('Connected to {}'.format(endpoint))

while True:
    hash_rate = float(socket.recv())  # Blocking
    clearscreen()

    print('Current Hash Rate: {:.6f}'.format(hash_rate))

    # Inefficient file reading
    print('Historical Hash Rates')
    with open(log_path, 'rt') as f:
        lines = f.read().splitlines()
        last_lines = lines[-10:]
        for line in ['\t{}'.format(l) for l in last_lines]:
            print(line)
