import subprocess
import sys


def run_subprocess(num_of_proc):
	print("Run %d subprocess.." % int(num_of_proc))
	for i in range(num_of_proc):
		#process = subprocess.run(['python3', 'sim_one_ou_processes.py'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

		process = subprocess.Popen(['python3', 'sim_one_ou_processes.py'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
		output = process.communicate()
		print(output)


		proc_id = process.pid
		print("Process ID is: ", process.pid)

	poll = process.poll()
	if poll == None:
		print("Subprocess is alive!!")
	else:
		print("Subprocesses are terminated")



if __name__ == '__main__':

	num_of_proc = sys.argv[1]
	run_subprocess(int(num_of_proc))


		

