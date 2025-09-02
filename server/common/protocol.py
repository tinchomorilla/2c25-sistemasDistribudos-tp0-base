import socket
import struct

BUFFER_SIZE = 4096

# Message types (matching Go client)
MESSAGE_TYPE_BET = 1
MESSAGE_TYPE_BATCH = 2
MESSAGE_TYPE_RESPONSE = 3
MESSAGE_TYPE_GET_WINNERS = 4


class BetMessage:
    """Represents a betting request from client"""

    def __init__(self, nombre, apellido, documento, nacimiento, numero):
        self.type = MESSAGE_TYPE_BET
        self.nombre = nombre
        self.apellido = apellido
        self.documento = documento
        self.nacimiento = nacimiento
        self.numero = numero


class BatchMessage:
    """Represents multiple bets sent together"""

    def __init__(self, agency, bets, eof=False):
        self.type = MESSAGE_TYPE_BATCH
        self.agency = agency  # Agency number (1-5)
        self.bets = bets  # List of BetMessage objects
        self.eof = eof

    @classmethod
    def from_data(cls, data):
        """Parse batch message from custom protocol data"""
        if len(data) < 1 or data[0] != MESSAGE_TYPE_BATCH:
            raise ValueError("Invalid batch message")

        content = data[1:].decode("utf-8")
        parts = content.split("|")

        if len(parts) < 3:
            raise ValueError("Invalid batch message format")

        agency = int(parts[0])
        eof = parts[1] == "1"
        bet_count = int(parts[2])

        bets = []
        idx = 3
        for _ in range(bet_count):
            if idx + 4 >= len(parts):
                break
            bet = BetMessage(
                nombre=parts[idx],
                apellido=parts[idx + 1],
                documento=parts[idx + 2],
                nacimiento=parts[idx + 3],
                numero=int(parts[idx + 4]),
            )
            bets.append(bet)
            idx += 5

        return cls(agency, bets, eof)


class GetWinnersMessage:
    """Represents a request to get winners for an agency"""

    def __init__(self, agency):
        self.type = MESSAGE_TYPE_GET_WINNERS
        self.agency = agency

    @classmethod
    def from_data(cls, data):
        """Parse get winners message from custom protocol data"""
        if len(data) < 1 or data[0] != MESSAGE_TYPE_GET_WINNERS:
            raise ValueError("Invalid get winners message")

        content = data[1:].decode("utf-8")
        agency = int(content)
        return cls(agency)


class ResponseMessage:
    """Represents server response to client"""

    def __init__(self, success, error=None, winners=None):
        self.type = MESSAGE_TYPE_RESPONSE
        self.success = success
        self.error = error
        self.winners = winners or []


def read_packet_from(client_socket):
    """Read length-prefixed packet and parse message"""
    # Read length prefix (4 bytes)
    length_data = _read_exact(client_socket, 4)
    length = int.from_bytes(length_data, byteorder='big')

    # Read message data
    data = _read_exact(client_socket, length)

    # Parse based on message type
    if len(data) < 1:
        raise ValueError("Empty message")

    msg_type = data[0]

    if msg_type == MESSAGE_TYPE_BATCH:
        return BatchMessage.from_data(data)
    elif msg_type == MESSAGE_TYPE_GET_WINNERS:
        return GetWinnersMessage.from_data(data)
    else:
        raise ValueError(f"Unknown message type: {msg_type}")


def send_response(client_socket, success, error=None, winners=None):
    """Send response message using custom protocol"""
    # Build response data
    data = bytearray()
    data.append(MESSAGE_TYPE_RESPONSE)

    # Add success flag
    data.extend(("1" if success else "0").encode("utf-8"))
    data.extend(b"|")

    # Add error if present
    if error:
        data.extend(error.encode("utf-8"))
    data.extend(b"|")

    # Add winners if present
    if winners:
        data.extend(str(len(winners)).encode("utf-8"))
        data.extend(b"|")
        for winner in winners:
            data.extend(winner.encode("utf-8"))
            data.extend(b"|")
    else:
        data.extend(b"0|")

    # Send length-prefixed message
    length = len(data)
    length_bytes = length.to_bytes(4, byteorder='big')

    client_socket.send(length_bytes)
    client_socket.send(data)


def _read_exact(sock, n):
    """Read exactly n bytes from socket"""
    data = b""
    while len(data) < n:
        chunk = sock.recv(n - len(data))
        if not chunk:
            raise RuntimeError("Socket connection broken")
        data += chunk
    return data
