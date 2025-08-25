#!/usr/bin/env python3

import subprocess
import sys
import time


def test_echo_server():
    """
    Test the echo server using netcat within the Docker network
    """
    try:
        # Test message to send
        test_message = "Hello Echo Server!"

        # Use docker exec to run netcat inside the server container
        # Docker DNS automatically resolves "server" to the container's IP
        cmd = [
            "docker",
            "exec",
            "server",
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
