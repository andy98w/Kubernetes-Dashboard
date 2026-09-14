"""Synthetic application: readiness remains green during business endpoint failures."""
import os
import time
from http.server import BaseHTTPRequestHandler, HTTPServer

MODE = os.environ.get('MODE', 'healthy')
requests = 0

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        global requests
        if self.path == '/healthz':
            self.send_response(200)
            self.end_headers()
            return
        requests += 1
        if MODE == 'slow':
            time.sleep(.4)
        failed = MODE == 'errors' or (MODE == 'late-failure' and requests > 20)
        self.send_response(503 if failed else 200)
        self.end_headers()
        self.wfile.write((os.environ.get('SLOT','unknown')+':'+MODE).encode())
    def log_message(self, *_):
        pass

HTTPServer(('0.0.0.0', 8080), Handler).serve_forever()
