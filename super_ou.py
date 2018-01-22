import socket
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer



#Create a TCP/IP socket
try:
	s = socket.socket(socket.AF_INET, socket.SOCK_STREAM) 
except Exception as e:
	print("Could not create socket. Error Code: ", str(e[0]), "Error: ", e[1])
	sys.exit(0)

server_address = ('localhost', 8080)


print('starting up on {} port {}'.format(*server_address))
s.bind(server_address)

s.listen(1)

while True:
	print("Waiting for a connection...")
	connection, client_addr = s.accept()
	try:
		print("Conncetion from ", client_addr)

		while True:
			data = connection.recv(16)
			print("Received {!r}".format(data))

			if data:
				print("Sending data back to client")
				connection.sendall(data)
			else:
				print("No data from ", client_addr)
				break

	finally:
		#Cean up the connection
		connection.close()




s.close()



#class SuperOu(BaseHTTPRequestHandler):
"""docstring for SuperOu"BaseHTTPRequestHandlerf __init__(self, arg):
		super(SuperOu,BaseHTTPRequestHandler.__init__()
		self.arg = arg

	def do_PUT(self):
		pass
		self.send_response(200)
		self.end_headers()

	def do_GET(self):
		pass
		self.send_response(200)
		self.send_header('Content-type', 'text/plain')
		self.end_headers()
		#self.write(value)


	"""
if __name__ == '__main__':
	

	def run_server():
		print("Starting server on port ", port)
		server.serve_forever()
		print("Server has shut down")

	def shutdown_server():		
		server.shutdown_server()
		print("Server shutting down...")

	#def shutdown_server_on_signal(signum):
	#	print("We get signal (%s). Asking server to shut down." % signum)
	#	server.shutdown()