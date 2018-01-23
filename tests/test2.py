from http.server  import HTTPServer, BaseHTTPRequestHandler
from socketserver import ThreadingMixIn
import threading
import time
import re
import argparse
#import httplib
import signal
import socket
import os
import subprocess

class Node(object):
	pass

class Handler(BaseHTTPRequestHandler):
	pass
		
class ThreadedHTTPServer(ThreadingMixIn, HTTPServer):
    """Handle requests in a separate thread."""



def find_free_ip(first_node=False):
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


if __name__ == '__main__':
	new_ip = "localhost"
	new_port = 8080

	server = None

	while server is None:
		try:
			server = ThreadedHTTPServer((new_ip, new_port), Handler)
			print("Server started on %s" % str(new_ip) + ':' + str(new_port))
		except Exception as e:
			new_port += 1
			raise e

	
	node = Node()
