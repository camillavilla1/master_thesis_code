import os
#from multiprocessing import Process
import multiprocessing as mp
import time


class newProcess():
	def __init__(self):
		pass

	#For testing
	def do_something(self, i):
		pass
		time.sleep(0.2)
		print('%s * %s = %s' % (i, i, i*i))

	def info(self):
		#print("Module name: ", __name__)
		print('Parent process:', os.getppid())
		print("Process id: ", os.getpid())

	def run(self, num_proc):
		processes = []
		for i in range(num_proc):
			p = mp.Process(target=self.info)#, args=(1,))
			processes.append(p)

		#[x.start() for x in processes]
		for x in processes:
			x.start()
			#print("Child process state: %d" % x.is_alive())
			#x.join()
			#print("Child process state: %d" % x.is_alive())

	def shutdown(self, num_proc):
		print("Shutdown initiated")
		


if __name__ == '__main__':
	nProcess = newProcess()
	num_proc = 4
	nProcess.run(num_proc)

	#time.sleep(3)

	#nProcess.shutdown(num_proc)
	#nProcess.terminate()
	#time.sleep(3)
	#print("Child process state: %d" % nProcess.is_alive())
