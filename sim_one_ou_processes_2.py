import random
import os
import logging
import time
import sys
import signal

import socketserver
import threading

from http.server import BaseHTTPRequestHandler, HTTPServer

class ObservationUnit():
	def run(self):
		self.temperature_sensor()
		self.weather = self.weather_sensor()
		self.camera_sensor()
		#print('Parent process:', os.getppid())
		print("Process ID: ", os.getpid())
		

	""" Simulate data from a weather sensor """
	def weather_sensor(self):
		list_of_weather = ['sun', 'rain', 'cloudy', 'windy', 'stormy']

		random_weather = random.choice(list_of_weather)
		print("Random weather is: %s." % random_weather)
		return random_weather


	""" Simulate data from temperature sensor """
	def temperature_sensor(self):
		rand_temp = random.randrange(1,20)
		print("Random temperature is %d." % rand_temp)


	""" Simulate data from a camera sensor """
	def camera_sensor(self):
		pass



class Handler(BaseHTTPRequestHandler):
	pass

if __name__ == '__main__':
	server = None
	ip = 'localhost'
	port = 8080

	#server_class = HTTPServer
	#server = server_class((ip, port), Handler)
	#server = http.server((ip, port), Handler)

	while server is None:
		try:
			#server = ThreadedHTTPServer((ip, port), Handler)
			server_class = HTTPServer
			server = server_class((ip, port), Handler)
			print("Server started on %s" %str(ip) + ':' + str(port))
		except Exception as e:
			print("Caught exception socket.error: %s. Trying again.." % e)
			port += 1
	#		raise e



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


	ou = ObservationUnit()
	ou.run()



    # Wait on server thread, until timeout has elapsed
    #
    # Note: The timeout parameter here is also important for catching OS
    # signals, so do not remove it.
    #
    # Having a timeout to check for keeps the waiting thread active enough to
    # check for signals too. Without it, the waiting thread will block so
    # completely that it won't respond to Ctrl-C or SIGTERM. You'll only be
    # able to kill it with kill -9.
	thread.join(10*60)
	if thread.isAlive():
		print("Reached %.3f second timeout. Asking server to shut down" % args.die_after_seconds)
		server.shutdown()

	print("Exited cleanly")