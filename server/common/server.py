import socket
import logging
import signal
import os
import threading
from .protocol import (
    send_response,
)
from .utils import Bet, store_bets, load_bets, has_won
from .listener import Listener


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(("", port))
        self._server_socket.listen(listen_backlog)

        # Shutdown flag
        self._shutdown_requested = False

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
        """Main server entry point - delegates to the listener"""
        # Create server callbacks for the client handlers
        server_callbacks = {
            "handle_batch_message": self._handle_batch_message,
            "handle_get_winners_message": self._handle_get_winners_message,
        }

        # Create and start the listener
        listener = Listener(
            server_socket=self._server_socket,
            shutdown_callback=lambda: self._shutdown_requested,
            server_callbacks=server_callbacks,
        )

        # Start listening for connections
        listener.run()

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
        """Handle request to get winners for an agency 

        Returns:
            bool: True if winners were successfully sent, False if lottery not ready
        """
        agency_id = message.agency

        # Check if lottery has been performed
        with self._lottery_done_lock:
            lottery_done = self._lottery_done

        if not lottery_done:
            logging.info(
                f"action: consulta_ganadores | result: in_progress | agency: {agency_id} | msg: lottery not performed yet"
            )
            send_response(
                client_sock,
                success=False,
                error="Lottery not performed yet. All agencies must finish first.",
            )
            return False  # Lottery not ready, client should keep trying

        # Check if requesting agency finished their bets
        with self._finished_agencies_lock:
            agency_finished = agency_id in self._finished_agencies

        if not agency_finished:
            logging.info(
                f"action: consulta_ganadores | result: in_progress | agency: {agency_id} | msg: agency did not finish sending bets"
            )
            send_response(
                client_sock, success=False, error="Agency did not finish sending bets"
            )
            return False  # Agency not finished, client should keep trying

        # Get winners for this agency
        winners = self._winners_by_agency.get(agency_id, []).copy()

        logging.info(
            f"action: consulta_ganadores | result: success | agency: {agency_id} | cant_ganadores: {len(winners)}"
        )

        send_response(client_sock, success=True, winners=winners)
        return True  # Successfully sent winners, client session complete

    def _perform_lottery(self):
        """Perform the lottery once all agencies have finished"""

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
