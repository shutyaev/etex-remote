import http.server
import io
import os
import os.path
import tempfile
import urllib.parse
import zipfile

class HTTPRequestHandler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        params = urllib.parse.parse_qs(urllib.parse.urlparse(self.path).query)
        makefile_name = params.get("makefile_name")[0]
        output_path = params.get("output_path")[0]
        # read incoming zip
        file_length = int(self.headers['Content-Length'])
        file_data = self.rfile.read(file_length)
        incoming_zip = zipfile.ZipFile(io.BytesIO(file_data))
        with tempfile.TemporaryDirectory(prefix="etex-server-") as temp_dir:
            # extracting zip to temp dir
            incoming_zip.extractall(path=temp_dir)
            incoming_zip.close()
            makefile_path = os.path.join(temp_dir, makefile_name)
            output_path = os.path.join(temp_dir, output_path)
            # calling etex
            # TODO replace this with etex call(s)
            print("calling etex for makefile " + makefile_path)
            os.mkdir(output_path)
            f = open(os.path.join(output_path, "foo.txt"), "w+")
            f.write("Hello, world!")
            f.close()
            f = open(os.path.join(output_path, "bar.txt"), "w+")
            f.write("Goodbye, world!")
            f.close()
            # End of TODO
            # building outgoing zip
            outgoing_bytes = io.BytesIO()
            outgoing_zip = zipfile.ZipFile(outgoing_bytes, "w")
            for root, _, files in os.walk(output_path):
                for f in files:
                    outgoing_zip.write(os.path.join(root, f), f)
            outgoing_zip.close()
            # send zip
            self.send_response(200)
            self.send_header("Content-Type", "application/zip")
            self.send_header("Content-Length", str(outgoing_bytes.getbuffer().nbytes))
            self.end_headers()
            self.wfile.write(outgoing_bytes.getvalue())

httpd = http.server.HTTPServer(('localhost', 8000), HTTPRequestHandler)
httpd.serve_forever()
