"""import multiprocessing
import time
import os

class MyProcess(multiprocessing.Process):
    def __init__(self, ):
        multiprocessing.Process.__init__(self)
        self.exit = multiprocessing.Event()

  
    def shutdown(self):
        print("Shutdown initiated")
        self.exit.set()

    def info(self):
	    #print("Module name: ", __name__)
	    #print('Parent process:', os.getppid())
	    print("Process id: ", os.getpid())

    def run(self):
    	processes = []
    	for i in range(4):
    		p = multiprocessing.Process(target=self.info)#, args=(1,))
    		processes.append(p)

    	for x in processes:
    		x.start()
    		#print("[0]Process state: %d" % x.is_alive())
    		x.join()
    		print("[1]Process state: %d" % x.is_alive())

		#[x.start() for x in processes]

if __name__ == "__main__":
    process = MyProcess()
    process.start()
    print("Waiting for a while")
    time.sleep(3)
   
    process.shutdown()
    time.sleep(3)
    print("Process state should be dead: %d" % process.is_alive())
    #print("Process pid: ", os.getpid())

"""
from multiprocessing import Process
import os

def info(title):
    print(title)
    print('module name:', __name__)
    print('parent process:', os.getppid())
    print('process id:', os.getpid())

def f(name):
    info('function f')
    print('hello', name)

if __name__ == '__main__':
    info('main line')
    p = Process(target=f, args=('bob',))
    p.start()
    p.join()




"""
from multiprocessing import Process
import multiprocessing


def print_data():
    name = multiprocessing.current_process().name
    id = multiprocessing.current_process().pid
    print(name, id)

if __name__=="__main__":
    proc1 = Process(name='process1', target=print_data)
    proc1.start()

    print("Is proc1 alive ",proc1.is_alive())
    proc1.join()
    print("Is proc1 alive ",proc1.is_alive())

    print("Finished")
 """