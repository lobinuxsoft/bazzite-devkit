#!/usr/bin/env python3
"""
Bazzite Devkit Service

A simple HTTP service that emulates the SteamOS Devkit Service,
allowing the Devkit Client to connect to Bazzite systems.

Runs on port 32000 and responds to:
- /properties.json - Device metadata
- /login-name - SSH username
- /register - SSH public key registration
"""

import os
import sys
import json
import socket
import argparse
import logging
from http.server import HTTPServer, BaseHTTPRequestHandler
from pathlib import Path

logging.basicConfig(
    format='%(asctime)s - %(levelname)s - %(message)s',
    level=logging.INFO
)
logger = logging.getLogger(__name__)

# Configuration
DEFAULT_PORT = 32000
DEVKIT_UTILS_PATH = os.path.expanduser('~/devkit-utils')
AUTHORIZED_KEYS_PATH = os.path.expanduser('~/.ssh/authorized_keys')


def get_username():
    """Get the current username"""
    return os.environ.get('USER', os.environ.get('USERNAME', 'deck'))


def get_hostname():
    """Get the system hostname"""
    return socket.gethostname()


def get_settings():
    """Get devkit settings"""
    return {
        'os': 'bazzite',
        'hostname': get_hostname(),
        'devkit_version': '1.0.0',
    }


class DevkitServiceHandler(BaseHTTPRequestHandler):
    """HTTP request handler for devkit service"""

    def log_message(self, format, *args):
        """Override to use our logger"""
        logger.info("%s - %s", self.address_string(), format % args)

    def send_json_response(self, data, status=200):
        """Send a JSON response"""
        response = json.dumps(data).encode('utf-8')
        self.send_response(status)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', len(response))
        self.end_headers()
        self.wfile.write(response)

    def send_text_response(self, text, status=200):
        """Send a plain text response"""
        response = text.encode('utf-8')
        self.send_response(status)
        self.send_header('Content-Type', 'text/plain')
        self.send_header('Content-Length', len(response))
        self.end_headers()
        self.wfile.write(response)

    def do_GET(self):
        """Handle GET requests"""
        logger.info(f"GET {self.path}")

        if self.path == '/properties.json':
            self.handle_properties()
        elif self.path == '/login-name':
            self.handle_login_name()
        elif self.path == '/':
            self.handle_root()
        else:
            self.send_error(404, 'Not Found')

    def do_POST(self):
        """Handle POST requests"""
        logger.info(f"POST {self.path}")

        if self.path == '/register':
            self.handle_register()
        else:
            self.send_error(404, 'Not Found')

    def handle_root(self):
        """Handle root path - basic info"""
        info = {
            'service': 'bazzite-devkit-service',
            'version': '1.0.0',
            'endpoints': ['/properties.json', '/login-name', '/register']
        }
        self.send_json_response(info)

    def handle_properties(self):
        """Handle /properties.json endpoint"""
        username = get_username()

        # devkit1 is the command prefix for running devkit scripts
        # This tells the client how to execute scripts on the remote device
        devkit1 = ['python3']

        properties = {
            'settings': json.dumps(get_settings()),
            'devkit1': devkit1,
            'login': username,
        }

        logger.info(f"Returning properties for user '{username}'")
        self.send_json_response(properties)

    def handle_login_name(self):
        """Handle /login-name endpoint (legacy)"""
        username = get_username()
        logger.info(f"Returning login name: {username}")
        self.send_text_response(username)

    def handle_register(self):
        """Handle /register endpoint - register SSH public key"""
        try:
            content_length = int(self.headers.get('Content-Length', 0))
            body = self.rfile.read(content_length).decode('utf-8')

            # Parse the public key from the request
            data = json.loads(body) if body.startswith('{') else {'pubkey': body.strip()}
            pubkey = data.get('pubkey', body.strip())

            if not pubkey:
                self.send_json_response({'error': 'No public key provided'}, 400)
                return

            # Ensure .ssh directory exists
            ssh_dir = Path(AUTHORIZED_KEYS_PATH).parent
            ssh_dir.mkdir(mode=0o700, exist_ok=True)

            # Check if key already exists
            existing_keys = set()
            if os.path.exists(AUTHORIZED_KEYS_PATH):
                with open(AUTHORIZED_KEYS_PATH, 'r') as f:
                    existing_keys = set(line.strip() for line in f if line.strip())

            if pubkey in existing_keys:
                logger.info("Public key already registered")
                self.send_json_response({'status': 'already_registered'})
                return

            # Append the new key
            with open(AUTHORIZED_KEYS_PATH, 'a') as f:
                f.write(f"\n{pubkey}\n")

            # Set correct permissions
            os.chmod(AUTHORIZED_KEYS_PATH, 0o600)

            logger.info("Public key registered successfully")
            self.send_json_response({'status': 'registered'})

        except Exception as e:
            logger.error(f"Error registering key: {e}")
            self.send_json_response({'error': str(e)}, 500)


def run_service(port=DEFAULT_PORT, bind='0.0.0.0'):
    """Run the devkit service"""
    server_address = (bind, port)
    httpd = HTTPServer(server_address, DevkitServiceHandler)

    logger.info(f"Bazzite Devkit Service starting on {bind}:{port}")
    logger.info(f"Username: {get_username()}")
    logger.info(f"Hostname: {get_hostname()}")
    logger.info("Press Ctrl+C to stop")

    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        logger.info("Shutting down...")
        httpd.shutdown()


def main():
    parser = argparse.ArgumentParser(description='Bazzite Devkit Service')
    parser.add_argument('--port', type=int, default=DEFAULT_PORT,
                        help=f'Port to listen on (default: {DEFAULT_PORT})')
    parser.add_argument('--bind', default='0.0.0.0',
                        help='Address to bind to (default: 0.0.0.0)')
    parser.add_argument('--debug', action='store_true',
                        help='Enable debug logging')

    args = parser.parse_args()

    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)

    run_service(port=args.port, bind=args.bind)


if __name__ == '__main__':
    main()
