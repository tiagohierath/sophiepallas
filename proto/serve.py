#!/usr/bin/env python3
# Serves proto/ with the cross-origin isolation headers dough needs for
# SharedArrayBuffer. Plain `python -m http.server` will not run dough.
import http.server, socketserver, sys

PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 8099

class H(http.server.SimpleHTTPRequestHandler):
    def end_headers(self):
        self.send_header("Cross-Origin-Opener-Policy", "same-origin")
        self.send_header("Cross-Origin-Embedder-Policy", "credentialless")
        self.send_header("Cache-Control", "no-store")
        super().end_headers()

socketserver.TCPServer.allow_reuse_address = True
with socketserver.TCPServer(("", PORT), H) as httpd:
    print(f"serving proto/ on http://localhost:{PORT}")
    httpd.serve_forever()
