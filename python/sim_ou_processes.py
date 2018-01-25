import multiprocessing as mp
import random
import os
import logging
import time
import sys

class ObservationUnit(mp.Process):
	def run(self):
		self.process_info()
		#self.logging()
		self.temperature_sensor()
		self.weather = self.weather_sensor()
		self.camera_sensor()
		print("\n")


	""" LOG WHETER PROCESS IS STARTING, RUNNING OR EXITING """
	def logging(self):
		mp.log_to_stderr()
		logger = mp.get_logger()
		logger.setLevel(logging.INFO)


	""" See information on process """
	def process_info(self):
		#print("Module name: ", __name__)
		print("In {}".format(self.name))
		#print('Parent process:', os.getppid())
		#print("Process id: ", os.getpid())
		

	""" Simulate data from a weather sensor """
	def weather_sensor(self):
		list_of_weather = ['sun', 'rain', 'cloudy', 'windy', 'stormy']

		random_weather = random.choice(list_of_weather)
		print("Random weather is: %s" % random_weather)
		return random_weather


	""" Simulate data from temperature sensor """
	def temperature_sensor(self):
		proc_name = mp.current_process().name
		#print("Making random temp in %s." % proc_name)
		rand_temp = random.randrange(1,20)
		print("Random temperature is %d" % rand_temp)


	""" Simulate data from a camera sensor """
	def camera_sensor(self):
		pass


if __name__ == '__main__':
	NUMBER_OF_PROCESSES = sys.argv[1]
	NUMBER_OF_PROCESSES = int(NUMBER_OF_PROCESSES)
	jobs = []
	for i in range(NUMBER_OF_PROCESSES):
		p = ObservationUnit()
		jobs.append(p)
		p.start()

	for j in jobs:
		j.join()


