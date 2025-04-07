import requests
from datetime import datetime
import time
import threading
import numpy as np
import os

ENDPOINT = "http://localhost/noop"
sleepMs = 1000
running = True
LOG_FILE = "logs/response_times.csv"

# Start fresh if the log exists
os.makedirs(os.path.dirname(LOG_FILE), exist_ok=True)
if os.path.exists(LOG_FILE):
    os.remove(LOG_FILE)

# Add CSV header
with open(LOG_FILE, "w") as log_file:
    log_file.write("response_time,sleepMs\n")

def request_loop():
    global sleepMs
    with open(LOG_FILE, "a") as log_file:
        while running:
            data = "1234"
            start = datetime.now()
            try:
                requests.post(ENDPOINT, data=data)
            except Exception as e:
                print(f"Request failed: {e}")
                continue
            diff = datetime.now() - start
            millis = diff.microseconds / 1000
            log_entry = f"{millis:.2f},{sleepMs}\n"
            log_file.write(log_entry)
            log_file.flush()
            time.sleep(sleepMs / 1000)

def analyze_tail_latencies():
    from collections import defaultdict
    grouped = defaultdict(list)

    try:
        with open(LOG_FILE, "r") as f:
            next(f)  # skip header
            for line in f:
                try:
                    rt_str, sm_str = line.strip().split(",")
                    rt = float(rt_str)
                    sm = int(sm_str)
                    grouped[sm].append(rt)
                except Exception as parse_err:
                    print(f"Skipping line due to parse error: {line.strip()}")
                    continue
    except FileNotFoundError:
        print("No response log file found.")
        return

    print("\n📊 90th Percentile Latency Summary:")
    for sm in sorted(grouped.keys()):
        values = grouped[sm]
        p90 = np.percentile(values, 90)
        print(f"  sleepMs = {sm} → P90 latency = {p90:.2f} ms ({len(values)} samples)")

def input_loop():
    global sleepMs, running
    while running:
        try:
            user_input = input("Enter new sleepMs (or 'exit' to stop): ")
            if user_input.lower() == 'exit':
                running = False
                break
            new_sleep = int(user_input)
            if new_sleep >= 0:
                sleepMs = new_sleep
            else:
                print("Please enter a non-negative number.")
        except ValueError:
            print("Invalid input. Please enter a number.")

    # Final analysis
    analyze_tail_latencies()

# Start threads
threading.Thread(target=request_loop, daemon=True).start()
input_loop()
