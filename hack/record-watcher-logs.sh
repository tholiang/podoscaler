#!/bin/bash
set -e

# Usage: ./watch_rounds.sh my-deployment [namespace]
DEPLOYMENT="$1"
NAMESPACE="${2:-default}"
OUTPUT_DIR="./tmp/"
OUTPUT_FILE="$OUTPUT_DIR/rounds.txt"

mkdir -p "$OUTPUT_DIR"
> "$OUTPUT_FILE"  # clear old file

echo "Watching logs for deployment '$DEPLOYMENT' in namespace '$NAMESPACE'..."
echo "Appending complete rounds to '$OUTPUT_FILE'"

kubectl logs -f deployment/"$DEPLOYMENT" \
  -n "$NAMESPACE" \
  --all-containers=true \
  --prefix=true \
  --max-log-requests=20 | \
awk -v outfile="$OUTPUT_FILE" '
/^Round / {
    if (in_round) {
        print "---" >> outfile
        for (i = 0; i < line_count; i++) {
            print buffer[i] >> outfile
            print buffer[i]
        }
        print "" >> outfile
        line_count = 0
    }
    in_round = 1
}
{
    if (in_round) {
        buffer[line_count++] = $0
    }
}
'