import random
import os
import logging
import time
import sys

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
		print("Random weather is: %s" % random_weather)
		return random_weather


	""" Simulate data from temperature sensor """
	def temperature_sensor(self):
		proc_name = mp.current_process().name
		rand_temp = random.randrange(1,20)
		print("Random temperature is %d" % rand_temp)


	""" Simulate data from a camera sensor """
	def camera_sensor(self):
		pass


if __name__ == '__main__':
	ou = ObservationUnit()
	ou.run()
