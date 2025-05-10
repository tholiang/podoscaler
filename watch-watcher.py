#!/usr/bin/env python3
import subprocess
import sys
import os
import re
import time

def main():
    deployment = "watcher"
    namespace = "default"

    output_dir = "./tmp"
    os.makedirs(output_dir, exist_ok=True)
    output_file = os.path.join(output_dir, "watcher-out.txt")

    # Clear previous output file
    with open(output_file, "w") as f:
        pass

    print("Watching logs for deployment '{}' in namespace '{}'...".format(deployment, namespace))
    print("Appending complete rounds to '{}'".format(output_file))
    
    # Build the kubectl command
    cmd = [
        "kubectl",
        "logs",
        "-f",
        "deployment/{}".format(deployment),
        "-n", namespace,
        "--all-containers=false",
        "--prefix=false",
        "--max-log-requests=20"
    ]

    # Start the subprocess
    proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=1, universal_newlines=True)

    round_pattern = re.compile(r"^Round\s+\d+")
    buffer_lines = []
    in_round = False

    while True:
        try:
            for line in proc.stdout:
                line = line.rstrip("\n")
                # Check if this line starts a new round
                if round_pattern.match(line):
                    # if we were already inside a round, flush the buffered round to file
                    if in_round and buffer_lines:
                        flush_round(buffer_lines, output_file)
                        buffer_lines = []
                    in_round = True

                # Buffer lines if we're in a round
                if in_round:
                    buffer_lines.append(line)
                
                # Also print output live
                print(line)

        except KeyboardInterrupt:
            print("Interrupted. Exiting.")
        finally:
            proc.kill()
        
        time.sleep(300)

def flush_round(buffer_lines, output_file):
    separator = "\n---\n"
    with open(output_file, "a") as out:
        out.write(separator)
        out.write("\n".join(buffer_lines))
        out.write("\n")
    print("\n[Round flushed to {}]\n".format(output_file))

if __name__ == "__main__":
    main()