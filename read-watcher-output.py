import sys
import os
import matplotlib.pyplot as plt
import numpy as np

if len(sys.argv) <= 3 or len(sys.argv) % 2 != 1:
    print("Usage: python read-watcher-output.py <file_path> <scaler_label> <file_path> <scaler_label> ...")
    sys.exit(1)

SLO = 30 # ms

latencies = {}
avg_deployment_usages = {}
avg_deployment_allocs = {}
avg_node_usages = {}

file_name = ""

min_rounds = 1000000

# Prompt the user for the name of the text file
for i in range(1, len(sys.argv), 2):  
    file_path = sys.argv[i]
    scaler_label = sys.argv[i+1]

    content = ""
    try:
        # Open and read the contents of the file
        with open(file_path, 'r') as file:
            content = file.read()
    except FileNotFoundError:
        print(f"Error: The file '{file_path}' was not found.")
    except Exception as e:
        print(f"An error occurred: {e}")

    file_name += os.path.basename(file_path).split(".")[0] + "-"

    latencies_over_time = {}
    node_usage_over_time = {}
    node_allocation_over_time = {}
    deployment_usage_over_time = {}
    deployment_alloc_over_time = {}
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

            if deployment not in deployment_alloc_over_time:
                deployment_alloc_over_time[deployment] = []
            if len(deployment_alloc_over_time[deployment]) != round_number:
                print("error: round number mismatch for deployment alloc for deployment " + deployment)
            deployment_alloc_over_time[deployment].append(allocation)

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

    # get averages
    latencies[scaler_label] = latencies_over_time["p90"]
    avg_deployment_usages[scaler_label] = []
    avg_deployment_allocs[scaler_label] = []
    avg_node_usages[scaler_label] = []
    for i in range(round_number+1):
        avg_deployment_usages[scaler_label].append(0)
        num_active_deployments = 0
        for deployment, usage in deployment_usage_over_time.items():
            if len(usage) > i and usage[i] > 0.05:
                avg_deployment_usages[scaler_label][i] += usage[i] * 100
                num_active_deployments += 1
        avg_deployment_usages[scaler_label][i] /= num_active_deployments if num_active_deployments > 0 else 0

        avg_deployment_allocs[scaler_label].append(0)
        for deployment, alloc in deployment_alloc_over_time.items():
            if len(alloc) > i:
                avg_deployment_allocs[scaler_label][i] += alloc[i]
        avg_deployment_allocs[scaler_label][i] /= len(deployment_alloc_over_time)

        avg_node_usages[scaler_label].append(0)
        num_active_nodes = 0
        for node, usage in node_usage_over_time.items():
            if len(usage) > i and usage[i] > 0:
                avg_node_usages[scaler_label][i] += usage[i] * 100
                num_active_nodes += 1
        avg_node_usages[scaler_label][i] /= num_active_nodes if num_active_nodes > 0 else 0
    
    if round_number < min_rounds:
        min_rounds = round_number

# graph the data

# Create the folder if it doesn't exist
output_folder = "watcher-graphs"
os.makedirs(output_folder, exist_ok=True)

times = np.arange(0, (min_rounds+1) * ROUND_INTERVAL, ROUND_INTERVAL)



# LATENCIES
plt.figure(figsize=(10, 6))
plt.title(scaler_label+" Latencies Per Scaler")
plt.xlabel("Time (minutes)")
plt.ylabel("Latency (ms)")

line_styles = ['-', '--', ':', '-.']
plt.plot(times, [SLO] * len(times), label="SLO", linestyle=':', color='red')
for scaler_label, lat in latencies.items():
    plt.plot(times, lat[:min_rounds+1], label=scaler_label, linestyle=line_styles.pop(0))


plt.legend()
output_file = os.path.join(output_folder, file_name+"latencies.png")
plt.savefig(output_file)
print(f"Plot saved to {output_file}")
plt.close()

# DEPLOYMENT USAGE
plt.figure(figsize=(10, 6))
plt.title("Average Deployment Usage Per Scaler (For Active Deployments)")
plt.xlabel("Time (minutes)")
plt.ylabel("Usage (% of allocation)")

line_styles = ['-', '--', ':', '-.']
for scaler_label, usage in avg_deployment_usages.items():
    plt.plot(times, usage[:min_rounds+1], label=scaler_label, linestyle=line_styles.pop(0))

plt.legend()
output_file = os.path.join(output_folder, file_name+"deployment_usage.png")
plt.savefig(output_file)
print(f"Plot saved to {output_file}")
plt.close()

# DEPLOYMENT ALLOC
plt.figure(figsize=(10, 6))
plt.title("Average Deployment Allocation Per Scaler")
plt.xlabel("Time (minutes)")
plt.ylabel("Allocation (mCPU)")

line_styles = ['-', '--', ':', '-.']
for scaler_label, alloc in avg_deployment_allocs.items():
    plt.plot(times, alloc[:min_rounds+1], label=scaler_label, linestyle=line_styles.pop(0))

plt.legend()
output_file = os.path.join(output_folder, file_name+"deployment_alloc.png")
plt.savefig(output_file)
print(f"Plot saved to {output_file}")
plt.close()

# NODE USAGE
plt.figure(figsize=(10, 6))
plt.title("Average Node Usage Per Scaler")
plt.xlabel("Time (minutes)")
plt.ylabel("Usage (%)")

line_styles = ['-', '--', ':', '-.']
for scaler_label, usage in avg_node_usages.items():
    plt.plot(times, usage[:min_rounds+1], label=scaler_label, linestyle=line_styles.pop(0))

plt.legend()
output_file = os.path.join(output_folder, file_name+"node_usage.png")
plt.savefig(output_file)
print(f"Plot saved to {output_file}")
plt.close()

print()
print("---STATS---")
print()

for scaler_label, lat in latencies.items():
    avg_latency_over_time = sum(lat) / len(lat)
    print(f"Average p90 latency for {scaler_label}: {avg_latency_over_time:.2f} ms")
print()

for scaler_label, alloc in avg_deployment_allocs.items():
    avg_alloc_over_time = sum(alloc) / len(alloc)
    print(f"Average deployment allocation for {scaler_label}: {avg_alloc_over_time:.2f} mCPU")
print()

for scaler_label, usage in avg_deployment_usages.items():
    avg_usage_over_time = sum(usage) / len(usage)
    print(f"Average deployment usage (% of allocation) for active deployments for {scaler_label}: {avg_usage_over_time:.2f} %")
print()

for scaler_label, usage in avg_node_usages.items():
    avg_usage_over_time = sum(usage) / len(usage)
    print(f"Average node usage (% of capacity) for {scaler_label}: {avg_usage_over_time:.2f} %")