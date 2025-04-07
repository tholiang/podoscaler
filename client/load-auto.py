import requests
from datetime import datetime
import time
import numpy as np
import os
from collections import defaultdict

ENDPOINT = "http://localhost:3000/noop"
LOG_FILE = "logs/response_times.csv"
SAMPLES_PER_STEP = 200
SLEEP_STEPS = list(range(30, -1, -3))  # 100 → 0 in steps of 10

# Setup log file
os.makedirs(os.path.dirname(LOG_FILE), exist_ok=True)
if os.path.exists(LOG_FILE):
    os.remove(LOG_FILE)

with open(LOG_FILE, "w") as f:
    f.write("response_time,sleepMs\n")

# Request function
def send_requests(sleepMs):
    with open(LOG_FILE, "a") as log_file:
        for i in range(SAMPLES_PER_STEP):
            start = datetime.now()
            try:
                requests.post(ENDPOINT, data="1234")
            except Exception as e:
                print(f"[sleepMs={sleepMs}] Request failed: {e}")
                continue
            diff = datetime.now() - start
            millis = diff.total_seconds() * 1000
            log_file.write(f"{millis:.2f},{sleepMs}\n")
            log_file.flush()
            time.sleep(sleepMs / 1000)

# Analyzer
def analyze_tail_latencies():
    grouped = defaultdict(list)
    with open(LOG_FILE, "r") as f:
        next(f)  # Skip header
        for line in f:
            try:
                rt_str, sm_str = line.strip().split(",")
                rt = float(rt_str)
                sm = int(sm_str)
                grouped[sm].append(rt)
            except:
                continue

    print("\n📊 90th Percentile Latency Summary:")
    for sm in sorted(grouped.keys()):
        values = grouped[sm]
        if len(values) >= 10:
            p90 = np.percentile(values, 90)
            print(f"  sleepMs = {sm:>3} → P90 latency = {p90:.2f} ms ({len(values)} samples)")
        else:
            print(f"  sleepMs = {sm:>3} → not enough samples ({len(values)})")

# Run the sweep
for sleep in SLEEP_STEPS:
    print(f"➡️  Testing sleepMs = {sleep}")
    send_requests(sleep)

analyze_tail_latencies()
