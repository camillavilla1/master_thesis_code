import os
#from multiprocessing import Process
import multiprocessing as mp
import time
import logging
import random


class ObservationUnit():
	def __init__(self):
		pass

	#For testing
	def do_something(self, i):
		time.sleep(0.2)
		print('%s * %s = %s' % (i, i, i*i))

	""" See information on process """
	def info(self):
		#print("Module name: ", __name__)
		print('Parent process:', os.getppid())
		print("Process id: ", os.getpid())

	""" Run process with traget to do_something """
	def run(self, num_proc):
		processes = []
		#LOG WHETER PROCESS IS STARTING, RUNNING OR EXITING:
		#mp.log_to_stderr()
		#logger = mp.get_logger()
		#logger.setLevel(logging.INFO)

		for i in range(num_proc):
			#p = mp.Process(target=self.do_something, args=(i,))
			p = mp.Process(target=self.temperature_sensor)
			processes.append(p)

		for x in processes:
			x.start()


	def camera_sensor(self):
		pass

	def weather_sensor(self):
		pass

	def temperature_sensor(self):
		#pass
		proc_name = mp.current_process().name
		print("Doing something in %s." % proc_name)
		rand_temp = random.randrange(1,20)
		print(rand_temp)



if __name__ == '__main__':
	num_proc = 4
	nProcess = ObservationUnit()
	#nProcess.info()
	#nProcess.temperature_sensor()
	nProcess.run(num_proc)

	#time.sleep(20)
	#print("Exiting!")


