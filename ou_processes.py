import os
from multiprocessing import Process
import multiprocessing as mp
from time import sleep

class newProcess(num_proc):
	def __init__(self):
		pass

	def do_something(self, i):
		sleep(0.2)
		print('%s * %s = %s' % (i, i, i*i))

	def run(self):
		processes = []
		for i in range(num_proc):
			p = mp.Process(target=self.do_something, args=(1,))
			processes.append(p)

		[x.start() for x in processes]




if __name__ == '__main__':
	nProcess = newProcess()
	nProcess.run()