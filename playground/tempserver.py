# Eventually we'll use a more standard solution for a development web server,
# but this gets us simple static file serving wihout a lot of fuss.

import BaseHTTPServer
import SimpleHTTPServer
import os.path
import os

port = 8000
print("Running on port %d" % port)

dist_dir = os.path.join(os.path.dirname(__file__), "dist")
os.chdir(dist_dir)

SimpleHTTPServer.SimpleHTTPRequestHandler.extensions_map['.wasm'] = 'application/wasm'

httpd = BaseHTTPServer.HTTPServer(
    ('localhost', port), SimpleHTTPServer.SimpleHTTPRequestHandler)

httpd.serve_forever()
