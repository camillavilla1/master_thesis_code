


from http.server import BaseHTTPRequestHandler, HTTPServer



class Handler(BaseHTTPRequestHandler):
	pass
	def do_GET():
		pass

	def do_POST():
		pass



def run(server_class=HTTPServer, handler_class=BaseHTTPRequestHandler):
    server_address = ('', 8000)
    httpd = server_class(server_address, handler_class)
    print("Server started on ", server_address)
    httpd.serve_forever()



if __name__ == '__main__':
	run()