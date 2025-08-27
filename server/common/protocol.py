import json
import socket

BUFFER_SIZE = 4096

def send_message(sock: socket.socket, message: dict):
    """Serialize dict as JSON and send with newline delimiter"""
    data = json.dumps(message).encode("utf-8") + b"\n"
    total_sent = 0
    while total_sent < len(data):
        sent = sock.send(data[total_sent:])
        if sent == 0:
            raise RuntimeError("socket connection broken (short write)")
        total_sent += sent

def recv_message(sock: socket.socket) -> dict:
    """Receive JSON message terminated by newline"""
    chunks = []
    while True:
        chunk = sock.recv(BUFFER_SIZE)
        if not chunk:
            raise RuntimeError("socket connection broken (short read)")
        chunks.append(chunk)
        if b"\n" in chunk:
            break
    data = b"".join(chunks).split(b"\n", 1)[0]
    return json.loads(data.decode("utf-8"))
