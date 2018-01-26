package main

import (
	"fmt"
	"flag"
	"log"
	"net"
	//"runtime"
)


func main() {
	host := flag.String("host", "localhost", "Host of server")
	port := flag.Int("port", 0, "Port of server")

	flag.Parse()

	fmt.Println("Host is", *host)
	fmt.Println("Port is", *port)

	if *port == 0 {
		fmt.Println("Port is not set. Port is", *port)
	}
	
	//get_local_ip()

}

func error_msg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}


func set_max_procs() {
	/*NumCPU returns the number of logical CPUs usable by the current process. */
	//numCpu := runtime.numCPU()

	/*GOMAXPROCS sets the maximum number of CPUs that
	can be executing simultaneously and returns the previous setting.
	If n < 1, it does not change the current setting. The number of logical CPUs on the local machine can be queried with NumCPU. This call will go away when the scheduler improves. */
	//runtime.GOPAXPROCS()
}

func get_local_ip() {
	/*Interfaces returns a list of the system's network interfaces. */
	interfaces, err := net.Interfaces()
	error_msg("Interface: ", err)
	fmt.Println(interfaces)

	/*for _, i := range interfaces {
		fmt.Printf("Name : %v \n", i.Name)
		// see http://golang.org/pkg/net/#Flags
		fmt.Println("Interface type and supports : ", i.Flags.String())
	}*/


	interface_addr, err := net.InterfaceAddrs()
	error_msg("Interfaceaddr: ", err)
	fmt.Println(interface_addr)
}
