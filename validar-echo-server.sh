#!/bin/bash

# Script to validate echo server functionality using netcat
# This script calls a Python script to handle the actual netcat interaction

# Check if Python script exists
if [ ! -f "scripts/test_echo_server.py" ]; then
    echo "action: test_echo_server | result: fail"
    echo "Error: scripts/test_echo_server.py not found"
    exit 1
fi

# Call the Python script to perform the validation
python3 scripts/test_echo_server.py

