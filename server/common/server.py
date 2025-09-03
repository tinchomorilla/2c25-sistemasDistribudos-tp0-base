import socket
import logging
import signal
import threading


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(("", port))
        self._server_socket.listen(listen_backlog)

        # Track active connections for graceful shutdown
        self._client_socket = None
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
            self._cleanup_connections()
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

                    self._client_socket = client_sock

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

        logging.info("action: shutdown | result: success | msg: graceful shutdown completed")

    def __handle_client_connection(self, client_sock):
        """Handle communication with a client and close socket"""
        try:
            # TODO: Modify the receive to avoid short-reads
            msg = client_sock.recv(1024).rstrip().decode("utf-8")
            addr = client_sock.getpeername()
            logging.info(
                f"action: receive_message | result: success | ip: {addr[0]} | msg: {msg}"
            )
            # TODO: Modify the send to avoid short-writes
            client_sock.send(f"{msg}\n".encode("utf-8"))
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        finally:
            client_sock.close()

    def __accept_new_connection(self):
        """Accept new connections (blocking call)"""
        logging.info("action: accept_connections | result: in_progress")
        c, addr = self._server_socket.accept()
        logging.info(f"action: accept_connections | result: success | ip: {addr[0]}")
        return c

    def _cleanup_connections(self):
        """Close active connection if exists"""

        # Close the currently active client connection, if it exists
        if self._client_socket:
            try:
                self._client_socket.shutdown(socket.SHUT_RDWR)
                self._client_socket.close()
                logging.info(
                    "action: shutdown | result: success | msg: client connection closed"
                )
            except Exception as e:
                logging.error(
                    f"action: shutdown | result: fail | msg: error closing client connection | error: {e}"
                )
            self._client_socket = None
