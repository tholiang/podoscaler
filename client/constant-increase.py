import requests
import string
import random
from datetime import datetime
import time

ENDPOINT = "http://localhost:3000/noop"

def random_string(length):
    characters = string.ascii_letters + string.digits
    data = ''.join(random.choice(characters) for _ in range(length))
    return data


N = 100
while True:
    print("making request of size "+str(N))
    data = random_string(int(N))
    start = datetime.now()
    requests.post(ENDPOINT, data=data)
    diff = datetime.now() - start
    millis = diff.microseconds / 1000
    print("response received in "+str(millis)+" ms")
    N *= 1.1

    time.sleep(1)