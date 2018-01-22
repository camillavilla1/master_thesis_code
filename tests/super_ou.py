import socket
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer



def find_free_port():
	try:
		print("Trying to find a free port!")
		#Create a TCP/IP socket
		tcp = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
		tcp.bind(('localhost', 0))
		host, port = tcp.getsockname()
		print('Starting up on port {}:{}'.format(host, port))
		#tcp.close()
		return tcp, host, port
	except Exception as e:
		print("Could not create socket. Error Code: ", str(e[0]), "Error: ", e[1])
		raise e



def waiting_for_connection():

	s, host, port = find_free_port()
	#address = str(host)+ ':'+ str(port)
	#print(address)

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
"""if __name__ == '__main__':
	

	def run_server():
		print("Starting server on port ", port)
		server.serve_forever()
		print("Server has shut down")

	def shutdown_server():		
		server.shutdown_server()
		print("Server shutting down...")
"""
	#def shutdown_server_on_signal(signum):
	#	print("We get signal (%s). Asking server to shut down." % signum)
	#	server.shutdown()