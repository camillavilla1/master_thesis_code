import subprocess
import sys
"""
completed = subprocess.run(
    ['ls', '-1'],
    stdout=subprocess.PIPE,
)
print('returncode:', completed.returncode)
print('Have {} bytes in stdout:\n{}'.format(
    len(completed.stdout),
    completed.stdout.decode('utf-8'))
)
"""


if __name__ == '__main__':

	num_of_proc = sys.argv[1]
	print("Run %d subprocess.." % int(num_of_proc))

	for i in range(int(num_of_proc)):
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

		

