import multiprocessing as mp
import random
import os
import logging
import time

import hashlib
import httplib

from BaseHTTPServer import BaseHTTPRequestHandler, BaseHTTPServer

class ObservationUnit():
	def __init__(self, address):
		self.address = address
		self.id = hash_value(self.address)

	def camera_sensor(self):
		pass

	def weather_sensor(self):
		pass

	def temperature_sensor(self):
		pass




class ObservationUnitHTTPHandler(BaseHTTPRequestHandler):
	def do_PUT(self):
		pass

	def do_GET(self):
		pass
		


def hash_value(value):
	m = hashlib.sha1(value)
	m.hexdigest()
	short = m.hexdigest()
	short = int(short, 16)
	return short

def random_ip():
	pass

if __name__ == '__main__':
	

	def run_server():
		print("Starting server on port ", )