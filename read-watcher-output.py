import sys
import os
import matplotlib.pyplot as plt
import numpy as np

if len(sys.argv) != 3:
    print("Usage: python read-watcher-output.py <file_path> <scaler_label>")
    sys.exit(1)

# Prompt the user for the name of the text file
file_path = sys.argv[1]
scaler_label = sys.argv[2]

content = ""
try:
    # Open and read the contents of the file
    with open(file_path, 'r') as file:
        content = file.read()
except FileNotFoundError:
    print(f"Error: The file '{file_path}' was not found.")
except Exception as e:
    print(f"An error occurred: {e}")

file_name = os.path.basename(file_path).split(".")[0]

latencies_over_time = {}
node_usage_over_time = {}
node_allocation_over_time = {}
deployment_usage_over_time = {}
deployment_pods_over_time = {}

round_number = -1

ROUND_INTERVAL = 1 # minute
# Split the content into lines and process each line
for line in content.splitlines():
    if "ERROR" in line:
        print("error found: " + line)
    elif line.startswith("Round"):
        # Extract the round number from the line
        round_number += 1
    elif line.startswith("percentile"):
        percentile = line.split(" ")[1]
        latency = float(line.split("=")[1][:-1])
        if percentile not in latencies_over_time:
            latencies_over_time[percentile] = []
        if len(latencies_over_time[percentile]) != round_number:
            print("error: round number mismatch for percentile " + percentile + "; got " + str(len(latencies_over_time[percentile])) + " expected " + str(round_number))

        latencies_over_time[percentile].append(latency * 1000)  # Convert to ms
    elif line.startswith("node"):
        node = line.split(" ")[1]
        capacity = int(line.split(" ")[3])
        allocation = int(line.split(" ")[5])
        usage = int(line.split(" ")[7])

        if node not in node_usage_over_time:
            node_usage_over_time[node] = []
        if len(node_usage_over_time[node]) != round_number:
            node_usage_over_time[node] = [0] * round_number
        node_usage_over_time[node].append(float(usage) / capacity)

        if node not in node_allocation_over_time:
            node_allocation_over_time[node] = []
        if len(node_allocation_over_time[node]) != round_number:
            node_allocation_over_time[node] = [0] * round_number
        node_allocation_over_time[node].append(float(allocation) / capacity)
    elif line.startswith("deployment"):
        deployment = line.split(" ")[1]
        allocation = int(line.split(" ")[3])
        usage = int(line.split(" ")[5])
        pods = int(line.split(" ")[7])

        if deployment not in deployment_usage_over_time:
            deployment_usage_over_time[deployment] = []
        if len(deployment_usage_over_time[deployment]) != round_number:
            print("error: round number mismatch for deployment usage for deployment " + deployment)
        deployment_usage_over_time[deployment].append(float(usage) / allocation)

        if deployment not in deployment_pods_over_time:
            deployment_pods_over_time[deployment] = []
        if len(deployment_pods_over_time[deployment]) != round_number:
            print("error: round number mismatch for deployment pods for deployment " + deployment)
        deployment_pods_over_time[deployment].append(pods)

# graph the data

# Create the folder if it doesn't exist
output_folder = "watcher-graphs"
os.makedirs(output_folder, exist_ok=True)

times = np.arange(0, (round_number+1) * ROUND_INTERVAL, ROUND_INTERVAL)

# LATENCIES OVER TIME
plt.figure(figsize=(10, 6))
plt.title(scaler_label+" Latencies Over Time")
plt.xlabel("Time (minutes)")
plt.ylabel("Latency (ms)")

plt.plot(times, latencies_over_time["p90"], label=f"p90")
# plt.plot(times, latencies_over_time["p95"], label=f"p95")
# plt.plot(times, latencies_over_time["p99"], label=f"p99")

plt.legend()
output_file = os.path.join(output_folder, file_name+"-latencies_over_time.png")
plt.savefig(output_file)
print(f"Plot saved to {output_file}")
plt.close()


# NODE USAGE OVER TIME
plt.figure(figsize=(10, 6))
plt.title(scaler_label+" Node Usage Over Time")
plt.xlabel("Time (minutes)")
plt.ylabel("Usage (%)")
for node, usage in node_usage_over_time.items():
    if len(usage) < len(times):
        usage = usage + [0] * (len(times) - len(usage))
    plt.plot(times, usage, label=f"Node {node}")
plt.legend()
output_file = os.path.join(output_folder, file_name+"-node_usage_over_time.png")
plt.savefig(output_file)
print(f"Plot saved to {output_file}")
plt.close()


# NODE ALLOCATION OVER TIME
plt.figure(figsize=(10, 6))
plt.title(scaler_label+" Node Allocation Over Time")
plt.xlabel("Time (minutes)")
plt.ylabel("Allocation (%)")
for node, allocation in node_allocation_over_time.items():
    if len(allocation) < len(times):
        allocation = allocation + [0] * (len(times) - len(allocation))
    plt.plot(times, allocation, label=f"Node {node}")
plt.legend()
output_file = os.path.join(output_folder, file_name+"-node_allocation_over_time.png")
plt.savefig(output_file)
print(f"Plot saved to {output_file}")
plt.close()


# DEPLOYMENT USAGE OVER TIME
plt.figure(figsize=(10, 6))
plt.title(scaler_label+" Deployment Usage Over Time")
plt.xlabel("Time (minutes)")
plt.ylabel("Usage (%)")
for deployment, usage in deployment_usage_over_time.items():
    plt.plot(times, usage, label=f"Deployment {deployment}")
plt.legend()
output_file = os.path.join(output_folder, file_name+"-deployment_usage_over_time.png")
plt.savefig(output_file)
print(f"Plot saved to {output_file}")
plt.close()


# DEPLOYMENT PODS OVER TIME
plt.figure(figsize=(10, 6))
plt.title(scaler_label+" Deployment Pods Over Time")
plt.xlabel("Time (minutes)")
plt.ylabel("Pods")
for deployment, pods in deployment_pods_over_time.items():
    plt.plot(times, pods, label=f"Deployment {deployment}")
plt.legend()
output_file = os.path.join(output_folder, file_name+"-deployment_pods_over_time.png")
plt.savefig(output_file)
print(f"Plot saved to {output_file}")
plt.close()