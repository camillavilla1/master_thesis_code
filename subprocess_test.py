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

print("Run subprocess")

#process = subprocess.Popen(['python3', 'super_ou.py', '3'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
#output = process.communicate()
#print(output)


#New for Python 3.5
#when you just want to execute a command and wait until it finishes, but you don't want to do anything else meanwhile

num_of_proc = sys.argv[1]
#process = subprocess.run(['python3', 'super_ou.py', num_of_proc], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

for i in range(int(num_of_proc)):
	process = subprocess.run(['python3', 'sim_one_ou_processes.py'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
	

