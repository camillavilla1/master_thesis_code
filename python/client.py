import http.client
import socket


def main():
	# Create a TCP/IP socket
	sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

	# Connect the socket to the port where the server is listening
	server_address = ('localhost', 10000)
	print('Connecting to {} port {}'.format(*server_address))
	sock.connect(server_address)




if __name__ == '__main__':
	main()