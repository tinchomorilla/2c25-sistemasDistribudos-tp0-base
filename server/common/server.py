import socket
import logging
import signal
import os
import threading
import time
from .protocol import (
    read_packet_from,
    send_response,
    MESSAGE_TYPE_BATCH,
    MESSAGE_TYPE_GET_WINNERS,
)
from .utils import Bet, store_bets, load_bets, has_won


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(("", port))
        self._server_socket.listen(listen_backlog)

        # Track active connections and threads for graceful shutdown
        self._active_connections = []
        self._active_threads = []
        self._shutdown_requested = False

        # Thread safety - individual locks for each shared data structure
        self._connections_lock = threading.Lock()

        # Shared data structures with their own locks (like Arc<Mutex<T>> in Rust)
        self._finished_agencies_lock = threading.Lock()
        self._finished_agencies = (
            set()
        )  # Track which agencies have finished sending bets

        self._lottery_done_lock = threading.Lock()
        self._lottery_done = False  # Flag to track if lottery has been performed

        self._winners_by_agency = {}  # Dict mapping agency_id -> list of winners DNIs

        # Get expected number of agencies from environment variable
        self._expected_agencies = int(os.environ.get("EXPECTED_AGENCIES", 5))
        logging.info(
            f"action: config | result: success | expected_agencies: {self._expected_agencies}"
        )

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
        """Main server loop with graceful shutdown support and concurrent connection handling"""
        while not self._shutdown_requested:
            try:
                client_sock = self.__accept_new_connection()
                if client_sock and not self._shutdown_requested:
                    # Create a new thread to handle this connection
                    client_thread = threading.Thread(
                        target=self.__handle_client_connection_threaded,
                        args=(client_sock,),
                        daemon=True,
                    )

                    # Track the thread and connection
                    with self._connections_lock:
                        self._active_connections.append(client_sock)
                        self._active_threads.append(client_thread)

                    # Start the thread
                    client_thread.start()

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

        # Wait for all threads to complete and final cleanup
        self._wait_for_threads()
        self._cleanup_connections()
        logging.info(
            "action: shutdown | result: success | msg: graceful shutdown completed"
        )

    def __handle_client_connection_threaded(self, client_sock):
        """Thread-safe wrapper for handling client connections"""
        try:
            self.__handle_client_connection(client_sock)
        finally:
            # Clean up connection and thread tracking
            with self._connections_lock:
                if client_sock in self._active_connections:
                    self._active_connections.remove(client_sock)
                # Remove current thread from tracking
                current_thread = threading.current_thread()
                if current_thread in self._active_threads:
                    self._active_threads.remove(current_thread)

    def __handle_client_connection(self, client_sock):
        """Handle communication with a client and close socket"""
        try:
            addr = client_sock.getpeername()

            message = read_packet_from(client_sock)

            # Route message based on type
            if message.type == MESSAGE_TYPE_BATCH:
                self._handle_batch_message(message, addr)
            elif message.type == MESSAGE_TYPE_GET_WINNERS:
                self._handle_get_winners_message(message, client_sock)
            else:
                raise ValueError(f"Unsupported message type: {message.type}")

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

    def _handle_batch_message(self, batch_message):
        """Handle batch of bets with EOF detection"""
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
        if bets_to_store:
            store_bets(bets_to_store)

        logging.info(
            f"action: apuesta_recibida | result: success | cantidad: {len(batch_message.bets)}"
        )

        # Check if this agency has finished sending bets (EOF flag)
        if batch_message.eof:
            should_perform_lottery = False

            with self._finished_agencies_lock:
                self._finished_agencies.add(batch_message.agency)
                finished_count = len(self._finished_agencies)

                logging.info(
                    f"action: agency_finished | result: success | agency: {batch_message.agency} | total_finished: {finished_count}"
                )

                # Check if all expected agencies have finished
                if finished_count == self._expected_agencies:
                    should_perform_lottery = True

            # (outside the lock to avoid blocking other threads)
            if should_perform_lottery:
                self._perform_lottery()

    def _handle_get_winners_message(self, message, client_sock):
        """Handle request to get winners for an agency - thread safe"""
        agency_id = message.agency

        # Check if lottery has been performed
        with self._lottery_done_lock:
            lottery_done = self._lottery_done

        if not lottery_done:
            logging.error(
                f"action: consulta_ganadores | result: fail | agency: {agency_id} | error: lottery not performed yet"
            )
            send_response(
                client_sock,
                success=False,
                error="Lottery not performed yet. All agencies must finish first.",
            )
            return

        # Check if requesting agency finished their bets
        with self._finished_agencies_lock:
            agency_finished = agency_id in self._finished_agencies

        if not agency_finished:
            logging.error(
                f"action: consulta_ganadores | result: fail | agency: {agency_id} | error: agency did not finish sending bets"
            )
            send_response(
                client_sock, success=False, error="Agency did not finish sending bets"
            )
            return

        # Get winners for this agency
        winners = self._winners_by_agency.get(agency_id, []).copy()

        logging.info(
            f"action: consulta_ganadores | result: success | agency: {agency_id} | cant_ganadores: {len(winners)}"
        )

        send_response(client_sock, success=True, winners=winners)

    def _perform_lottery(self):
        """Perform the lottery once all agencies have finished"""
        # Double-check that lottery hasn't been performed already
        with self._lottery_done_lock:
            if self._lottery_done:
                return

        try:
            logging.info(
                "action: sorteo | result: in_progress | msg: all agencies finished, starting lottery"
            )

            # Load all bets from storage
            all_bets = list(load_bets())

            # Group bets by agency and check for winners
            for bet in all_bets:
                agency_id = bet.agency

                # Initialize agency list if it doesn't exist
                if agency_id not in self._winners_by_agency:
                    self._winners_by_agency[agency_id] = []

                # Check if this bet won
                if has_won(bet):
                    self._winners_by_agency[agency_id].append(bet.document)

            # Mark lottery as done
            with self._lottery_done_lock:
                self._lottery_done = True

            # Log successful lottery completion
            logging.info("action: sorteo | result: success")

        except Exception as e:
            logging.error(f"action: sorteo | result: fail | error: {e}")
            raise

    def __accept_new_connection(self):
        """Accept new connections (blocking call)"""
        logging.info("action: accept_connections | result: in_progress")
        c, addr = self._server_socket.accept()
        logging.info(f"action: accept_connections | result: success | ip: {addr[0]}")
        return c

    def _wait_for_threads(self):
        """Wait for all active threads to complete"""
        logging.info(
            "action: shutdown | result: in_progress | msg: waiting for threads to complete"
        )

        with self._connections_lock:
            threads_to_wait = self._active_threads.copy()

        for thread in threads_to_wait:
            try:
                # Wait up to 5 seconds for each thread to complete
                thread.join(timeout=5.0)
                if thread.is_alive():
                    logging.warning(
                        f"action: shutdown | result: warning | msg: thread did not complete in time"
                    )
            except Exception as e:
                logging.error(
                    f"action: shutdown | result: fail | msg: error waiting for thread | error: {e}"
                )

    def _cleanup_connections(self):
        """Close all active client connections"""
        with self._connections_lock:
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
