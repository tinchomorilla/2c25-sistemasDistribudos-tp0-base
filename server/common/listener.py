import socket
import threading
import logging
from .client_handler import ClientHandler


class Listener:
    def __init__(self, server_socket, shutdown_callback, server_callbacks):
        """
        Initialize the listener

        Args:
            server_socket: The server socket to listen on
            shutdown_callback: Callback function to check if shutdown is requested
            server_callbacks: Dictionary with callback functions to server methods
        """
        self._server_socket = server_socket
        self._shutdown_callback = shutdown_callback
        self._server_callbacks = server_callbacks

        # Track active client handlers for graceful shutdown
        self._active_handlers = []
        self._handlers_lock = threading.Lock()

    def run(self):
        """Main listener loop with graceful shutdown support and concurrent connection handling"""
        while not self._shutdown_callback():
            try:
                client_sock = self._accept_new_connection()
                if client_sock and not self._shutdown_callback():
                    # Create a new ClientHandler to handle this connection
                    client_handler = ClientHandler(
                        client_socket=client_sock,
                        client_address=client_sock.getpeername(),
                        server_callbacks=self._server_callbacks,
                    )

                    # Track the handler
                    with self._handlers_lock:
                        self._active_handlers.append(client_handler)

                    # Start the handler thread
                    client_handler.start()

            except socket.error:
                # Server socket was likely closed due to shutdown
                if self._shutdown_callback():
                    break
                else:
                    logging.error(
                        "action: accept_connections | result: fail | error: socket error"
                    )
            except Exception as e:
                if not self._shutdown_callback():
                    logging.error(f"action: server_loop | result: fail | error: {e}")

        # Wait for all handlers to complete and final cleanup
        self._wait_for_handlers()
        logging.info(
            "action: shutdown | result: success | msg: graceful shutdown completed"
        )

    def _accept_new_connection(self):
        """Accept new connections (blocking call)"""
        logging.info("action: accept_connections | result: in_progress")
        c, addr = self._server_socket.accept()
        logging.info(f"action: accept_connections | result: success | ip: {addr[0]}")
        return c

    def _wait_for_handlers(self):
        """Wait for all active client handlers to complete"""
        logging.info(
            "action: shutdown | result: in_progress | msg: waiting for handlers to complete"
        )

        with self._handlers_lock:
            handlers_to_wait = self._active_handlers.copy()

        for handler in handlers_to_wait:
            try:
                # Wait up to 5 seconds for each handler to complete
                handler.join(timeout=5.0)
                if handler.is_alive():
                    logging.warning(
                        f"action: shutdown | result: warning | msg: handler did not complete in time"
                    )
            except Exception as e:
                logging.error(
                    f"action: shutdown | result: fail | msg: error waiting for handler | error: {e}"
                )
