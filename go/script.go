package main

import (
	"fmt"
	"flag"
	"log"
	"net"
	"net/http"
	//"runtime"
)


var ouPort string
var ouHost string

func main() {
	//number_of_flags := 2
	host := flag.String("host", "localhost", "Host of server")
	port := flagset.StringVar(&ouPort, "port", ":0", "Port of server")

	flag.Parse()

	fmt.Println("Host is", *host)
	//fmt.Println("Port is", *port)

	/*if *port == 0 {
		fmt.Println("Port is not set.")
	}*/

	//number of flags, is 2 when both host and port is set..
	//numFlags := flag.NFlag()


	if f := flag.CommandLine.Lookup("host"); f != nil {
    	fmt.Printf("server set to %v\n", f)
    	
	} else {
    	fmt.Printf("Server not set\n")
	}

	//get_local_ip()
	NewHTTPServer(host, port)


}

func addCommonFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&ouHost, "port", "localhost", "wormgate port (prefix with colon)")
	flagset.StringVar(&segmentPort, "host", ":0", "segment port (prefix with colon)")
}


func error_msg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

func NewHTTPServer(hostname *string, hostPort *string) {
	log.Printf("Starting server on %s%s\n", *hostname, *hostPort)

	err := http.ListenAndServe(*hostPort, nil)
	if err != nil {
		log.Panic(err)
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
