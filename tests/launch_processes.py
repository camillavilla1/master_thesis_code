import subprocess
import sys 
import os
import signal
import threading

from http.server import BaseHTTPRequestHandler, HTTPServer
from socketserver import ThreadingMixIn

global_node_list = []

class Node():
	"""docstring for Node"""
	def __init__(self, ip, port):
		self.ip = ip
		self.port = port
		self.server = None


	def add_node(self, ip, port):
		process = subprocess.Popen(['python3', 'launch_processes.py'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

		output = process.communicate()
		print(output[0])

		proc_id = process.pid
		print("Process ID is: ", process.pid)



		

class Handler(BaseHTTPRequestHandler):
	pass
	def do_PUT(self):
		pass

	def do_GET(self):
		pass


def find_free_ip():
	pass


#Implement this in Handler
def start_server():
	pass
	print("Starting server...")

	try:
		while True:
			subprocess.Popen(...)

	except Exception as e:
		raise e

def launch_processes():
	num_of_procs = sys.argv[1]

	for i in range(int(num_of_procs)):
		process = subprocess.Popen(['python3', 'sim_one_ou_processes.py'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

		output = process.communicate()
		print(output[0])

		proc_id = process.pid
		print("Process ID is: ", process.pid)

	process.kill()

	poll = process.poll()
	if poll == None:
		print("Subprocess is alive!!")
	else:
		print("Subprocesses are terminated")



class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
	"""Handle requests in a seperate thread."""


if __name__ == '__main__':
	server = None

	ip = 'localhost'
	port = 8080

	while server is None:
		try:
			server = ThreadedHTTPServer((ip, port), Handler)
			print("Server started on %s" %str(ip) + ':' + str(port))
		except Exception as e:
			#raise e
			print("Caught exception socket.error: %s" % e)
			port += 1
		

	node = Node(ip, port)
	node.add_node(ip, port)
	#start_server()
	#launch_processes()

	def run_server():
		print("Starting server..")
		server.serve_forever()
		print("Server has shut down!")

	def shutdown_server_on_signal(signum, fram):
		print("We get signal (%s). Asking server to shut down" % signum)
		server.shutdown()


	# Start server in a new thread, because server HTTPServer.serve_forever()
	# and HTTPServer.shutdown() must be called from separate threads
	thread = threading.Thread(target=run_server)
	thread.daemon = True
	thread.start()

	# Shut down on kill (SIGTERM) and Ctrl-C (SIGINT)
	signal.signal(signal.SIGTERM, shutdown_server_on_signal)
	signal.signal(signal.SIGINT, shutdown_server_on_signal)
	
	thread.join(10*60)
	if thread.isAlive():
		print("Reached %.3f second timeout. Asking server to shut down" % 60)
		server.shutdown()

	print("Exited cleanly")
