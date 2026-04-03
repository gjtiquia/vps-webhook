#!/bin/bash
# Example webhook script
# $1 is the path to the request JSON log file

echo "=== Webhook Script Triggered ==="
echo "Request log file: $1"
echo "Contents:"
cat "$1"
echo ""
echo "=== Done ==="
