import os
#from multiprocessing import Process
import multiprocessing as mp
import time
import logging


class newProcess():
	def __init__(self):
		pass

	#For testing
	def do_something(self, i):
		time.sleep(0.2)
		print('%s * %s = %s' % (i, i, i*i))

	def info(self):
		#print("Module name: ", __name__)
		print('Parent process:', os.getppid())
		print("Process id: ", os.getpid())

	def run(self, num_proc):
		processes = []
		mp.log_to_stderr()
		logger = mp.get_logger()
		logger.setLevel(logging.INFO)

		for i in range(num_proc):
			p = mp.Process(target=self.do_something, args=(i,))
			processes.append(p)

		for x in processes:
			x.start()



if __name__ == '__main__':
	num_proc = 4
	nProcess = newProcess()
	nProcess.info()
	nProcess.run(num_proc)

	time.sleep(7)
	#print("Exiting!")
