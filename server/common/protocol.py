import json
import socket

BUFFER_SIZE = 4096

# Message types
MESSAGE_TYPE_BET = "bet"
MESSAGE_TYPE_BATCH = "batch"
MESSAGE_TYPE_RESPONSE = "response"


class BetMessage:
    """Represents a betting request from client"""

    def __init__(self, nombre, apellido, documento, nacimiento, numero):
        self.type = MESSAGE_TYPE_BET
        self.nombre = nombre
        self.apellido = apellido
        self.documento = documento
        self.nacimiento = nacimiento
        self.numero = numero

    @classmethod
    def from_dict(cls, data):
        """Create BetMessage from dictionary"""
        return cls(
            nombre=data["nombre"],
            apellido=data["apellido"],
            documento=data["documento"],
            nacimiento=data["nacimiento"],
            numero=data["numero"],
        )


class BatchMessage:
    """Represents multiple bets sent together"""

    def __init__(self, agency, bets):
        self.type = MESSAGE_TYPE_BATCH
        self.agency = agency  # Agency number (1-5)
        self.bets = bets  # List of BetMessage objects

    @classmethod
    def from_dict(cls, data):
        """Create BatchMessage from dictionary"""
        agency = data["agency"]
        bets = [BetMessage.from_dict(bet_data) for bet_data in data["bets"]]
        return cls(agency, bets)


class ResponseMessage:
    """Represents server response to client"""

    def __init__(self, success, error=None):
        self.type = MESSAGE_TYPE_RESPONSE
        self.success = success
        self.error = error

    def to_dict(self):
        """Convert ResponseMessage to dictionary for serialization"""
        result = {"type": self.type, "success": self.success}
        if self.error:
            result["error"] = self.error
        return result


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


def recv_bet_message(sock: socket.socket) -> BetMessage:
    """Receive and parse a bet message"""
    data = recv_message(sock)
    if data.get("type") != MESSAGE_TYPE_BET:
        raise ValueError(f"Expected bet message, got {data.get('type')}")
    return BetMessage.from_dict(data)


def recv_batch_message(sock: socket.socket) -> BatchMessage:
    """Receive and parse a batch message"""
    data = recv_message(sock)
    if data.get("type") != MESSAGE_TYPE_BATCH:
        raise ValueError(f"Expected batch message, got {data.get('type')}")
    return BatchMessage.from_dict(data)


def read_packet_from(sock: socket.socket):
    """Receive any message and return appropriate object"""
    data = recv_message(sock)
    message_type = data.get("type")

    if message_type == MESSAGE_TYPE_BATCH:
        return BatchMessage.from_dict(data)
    else:
        raise ValueError(f"Unknown message type: {message_type}")


def send_response(sock: socket.socket, success: bool, error: str = None):
    """Send a response message to client"""
    response = ResponseMessage(success, error)
    send_message(sock, response.to_dict())
