#!/usr/bin/env python3

import subprocess
import sys
import time


def test_echo_server():
    """
    Test the echo server using netcat from a temporary container
    """
    try:
        # Test message to send
        test_message = "Hello Echo Server!"

        # Create a temporary container with netcat, connect to testing_net, and test the server
        cmd = [
            "docker",
            "run",
            "--rm",
            "--network",
            "tp0_testing_net",  # Connect to the same network as the server
            "busybox:latest",
            "sh",
            "-c",
            f"echo '{test_message}' | nc server 12345",
        ]

        # Run the command and capture output
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=10)

        # Check if the response matches the sent message
        if result.returncode == 0 and test_message in result.stdout:
            print("action: test_echo_server | result: success")
            return True
        else:
            print("action: test_echo_server | result: fail")
            return False

    except Exception as e:
        print("action: test_echo_server | result: fail")
        return False


if __name__ == "__main__":
    test_echo_server()
