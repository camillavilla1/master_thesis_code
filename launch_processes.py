import subprocess
import sys 
import os


def launch_processes():
	num_of_procs = sys.argv[1]

	for i in range(int(num_of_procs)):
		process = subprocess.Popen(['python3', 'sim_one_ou_processes.py'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

		output = process.communicate()
		print(output[0])

		proc_id = process.pid
		print("Process ID is: ", process.pid)

	process.kill()

	poll = process.poll()
	if poll == None:
		print("Subprocess is alive!!")
	else:
		print("Subprocesses are terminated")


if __name__ == '__main__':
	launch_processes()

