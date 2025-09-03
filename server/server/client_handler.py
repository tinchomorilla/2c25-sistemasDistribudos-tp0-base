import socket
import logging
import threading
from threading import Thread
from ..protocol.protocol import (
    read_packet_from,
    send_response,
    MESSAGE_TYPE_BATCH,
    MESSAGE_TYPE_GET_WINNERS,
)


class ClientHandler(Thread):
    def __init__(self, client_socket, client_address, server_callbacks):
        """
        Initialize the client handler

        Args:
            client_socket: The client socket connection
            client_address: The client address tuple (ip, port)
            server_callbacks: Dictionary with callback functions to server methods:
                - handle_batch_message: callback for batch messages
                - handle_get_winners_message: callback for get winners messages
        """
        super().__init__(daemon=True)
        self.client_socket = client_socket
        self.client_address = client_address
        self.server_callbacks = server_callbacks

    def run(self):
        """Handle persistent communication with a client"""
        try:
            addr = self.client_address

            # Keep connection open for multiple messages
            while True:
                try:
                    message = read_packet_from(self.client_socket)

                    # Route message based on type
                    if message.type == MESSAGE_TYPE_BATCH:
                        self.server_callbacks["handle_batch_message"](message)
                    elif message.type == MESSAGE_TYPE_GET_WINNERS:
                        # Try to get winners 
                        success = self.server_callbacks["handle_get_winners_message"](
                            message, self.client_socket
                        )

                        # Only break if we successfully sent winners
                        if success:
                            logging.info(
                                f"action: client_session_complete | result: success | ip: {addr[0]} | agency: {message.agency}"
                            )
                            break
                        # If lottery not ready, continue the loop to wait for more messages
                    else:
                        raise ValueError(f"Unsupported message type: {message.type}")

                except socket.error as e:
                    # Client disconnected or connection error
                    logging.info(
                        f"action: client_disconnected | result: success | ip: {addr[0]} | reason: {e}"
                    )
                    break
                except ValueError as e:
                    # Invalid message format or data
                    logging.error(
                        f"action: receive_message | result: fail | ip: {addr[0]} | error: {e}"
                    )
                    send_response(self.client_socket, success=False, error=str(e))
                    break
                except Exception as e:
                    # Check if this is a connection-related error (common when client disconnects)
                    error_msg = str(e).lower()
                    if any(
                        keyword in error_msg
                        for keyword in ["connection", "broken", "reset", "closed"]
                    ):
                        logging.info(
                            f"action: client_disconnected | result: success | ip: {addr[0]} | reason: {e}"
                        )
                    else:
                        # Other errors (storage, etc.)
                        logging.error(
                            f"action: receive_message | result: fail | ip: {addr[0]} | error: {e}"
                        )
                        send_response(
                            self.client_socket,
                            success=False,
                            error="Internal server error",
                        )
                    break

        except Exception as e:
            logging.error(
                f"action: client_connection | result: fail | ip: {addr[0] if 'addr' in locals() else 'unknown'} | error: {e}"
            )
        finally:
            self.client_socket.close()
