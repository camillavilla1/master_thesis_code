import subprocess
import sys 

num_of_procs = sys.argv[1]
for i in range(int(num_of_procs)):
	process = subprocess.Popen(['python3', 'sim_one_ou_processes.py'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
	output = process.communicate()
	print(output[0])	
#process = subprocess.run(['python3', 'sim_one_ou_processes.py'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)