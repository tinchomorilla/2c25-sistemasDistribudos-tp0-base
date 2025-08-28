import socket
import logging
import signal
import threading
from .protocol import (
    read_packet_from,
    send_response,
    MESSAGE_TYPE_BET,
    MESSAGE_TYPE_BATCH,
)
from .utils import Bet, store_bets


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(("", port))
        self._server_socket.listen(listen_backlog)

        # Track active connections for graceful shutdown
        self._active_connections = []
        self._shutdown_requested = False

        # Set up signal handler for graceful shutdown
        signal.signal(signal.SIGTERM, self._signal_handler)

    def _signal_handler(self, signum, frame):
        """Handle SIGTERM signal for graceful shutdown"""
        logging.info("action: shutdown | result: in_progress | msg: received SIGTERM")
        self._shutdown_requested = True

        # Close server socket to stop accepting new connections
        try:
            self._server_socket.close()
            logging.info(
                "action: shutdown | result: success | msg: server socket closed"
            )
        except Exception as e:
            logging.error(
                f"action: shutdown | result: fail | msg: error closing server socket | error: {e}"
            )

    def run(self):
        """Main server loop with graceful shutdown support"""
        while not self._shutdown_requested:
            try:
                client_sock = self.__accept_new_connection()
                if client_sock and not self._shutdown_requested:
                    # Track active connections
                    self._active_connections.append(client_sock)

                    self.__handle_client_connection(client_sock)

                    # Remove connection once handled
                    if client_sock in self._active_connections:
                        self._active_connections.remove(client_sock)

            except socket.error:
                # Server socket was likely closed due to shutdown
                if self._shutdown_requested:
                    break
                else:
                    logging.error(
                        "action: accept_connections | result: fail | error: socket error"
                    )
            except Exception as e:
                if not self._shutdown_requested:
                    logging.error(f"action: server_loop | result: fail | error: {e}")

        # Final cleanup once loop exits
        self._cleanup_connections()
        logging.info(
            "action: shutdown | result: success | msg: graceful shutdown completed"
        )

    def __handle_client_connection(self, client_sock):
        """Handle communication with a client and close socket"""
        try:
            addr = client_sock.getpeername()

            message = read_packet_from(client_sock)

            self._handle_batch_bets(message, addr)

            # Send success response to client
            send_response(client_sock, success=True)

        except ValueError as e:
            # Invalid message format or data
            logging.error(
                f"action: receive_message | result: fail | ip: {addr[0]} | error: {e}"
            )
            send_response(client_sock, success=False, error=str(e))
        except Exception as e:
            # Other errors (connection, storage, etc.)
            logging.error(
                f"action: receive_message | result: fail | ip: {addr[0]} | error: {e}"
            )
            send_response(client_sock, success=False, error="Internal server error")
        finally:
            client_sock.close()

    def _handle_batch_bets(self, batch_message, addr):
        """Handle batch of bets (Exercise 6)"""
        bets_to_store = []

        # Process all bets in the batch
        for bet_message in batch_message.bets:
            bet = Bet(
                agency=str(batch_message.agency),
                first_name=bet_message.nombre,
                last_name=bet_message.apellido,
                document=bet_message.documento,
                birthdate=bet_message.nacimiento,
                number=str(bet_message.numero),
            )
            bets_to_store.append(bet)

        # Store all bets at once
        store_bets(bets_to_store)

        logging.info(
            f"action: apuesta_recibida | result: success | cantidad: {len(batch_message.bets)}"
        )

    def __accept_new_connection(self):
        """Accept new connections (blocking call)"""
        logging.info("action: accept_connections | result: in_progress")
        c, addr = self._server_socket.accept()
        logging.info(f"action: accept_connections | result: success | ip: {addr[0]}")
        return c

    def _cleanup_connections(self):
        """Close all active client connections"""

        connections_to_close = self._active_connections.copy()
        self._active_connections.clear()

        for client_sock in connections_to_close:
            try:
                client_sock.close()
                logging.info(
                    "action: shutdown | result: success | msg: client connection closed"
                )
            except Exception as e:
                logging.error(
                    f"action: shutdown | result: fail | msg: error closing client connection | error: {e}"
                )
