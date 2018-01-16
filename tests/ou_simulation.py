import multiprocessing
import os
import time


class ObservationUnit(multiprocessing.Process):
	"""docstring for ObservationUnit"""
	def __init__(self, ):
		multiprocessing.Process.__init__(self)
		self.exit = multiprocessing.Event()

	def run(self):
		while not self.exit.is_set():
			pass
		print("You exited")

	def shutdown(self):
		print("Shutdown initiated!")
		self.exit.set()


	def info(self):
		print("Parent process: ", os.getppid())
		print("Process id: ", os.getpid())



if __name__ == '__main__':
	ouProcess = ObservationUnit()
	ouProcess.start()
	ouProcess.info()

	time.sleep(2)
	print("Process state: %d" % ouProcess.is_alive())
	ouProcess.shutdown()
	print("Process state: %d" % ouProcess.is_alive())


