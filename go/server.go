package main

import (
	"fmt"
	"flag"
	"log"
	"net"
	"os"
	//"runtime"
	"net/http"
	"io"
	"io/ioutil"
)


var ouPort string
var ouHost string

//var hostname string
var hostaddress string

var startedOuServer []string

func main() {
	
	//hostname, _ = os.Hostname()
	//fmt.Printf(hostname)

	//hostAddress ) string.Split(hostname, ".")[0]
	var runMode = flag.NewFlagSet("run", flag.ExitOnError)
	addCommonFlags(runMode)
	//var runHost = runMode.String("host", "localhost", "Run host")

	if len(os.Args) == 1 {
		log.Fatalf("No mode specified\n")
	}

	switch os.Args[1] {
	case "run":
		runMode.Parse(os.Args[2:])
		startServer()

	default:
		log.Fatalf("Unknown mode %q\n", os.Args[1])
	}

}

func addCommonFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&ouHost, "host", "localhost", "wormgate port (prefix with colon)")
	flagset.StringVar(&ouPort, "port", ":0", "segment port (prefix with colon)")
}


func error_msg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

func startServer() {
	log.Printf("Starting segment server on %s%s\n", ouHost, ouPort)

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/shutdown", shutdownHandler)

	hostaddress = ouHost + ouPort
	startedOuServer = append(startedOuServer, hostaddress)
	//fmt.Printf("%v\n", startedOuServer)


	err := http.ListenAndServe(ouPort, nil)
	if err != nil {
		log.Panic(err)
	}


}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Don't use the request body. But we should consume it anyway.
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()


	fmt.Fprintf(w, "Index Handler\n")
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	// Shut down
	log.Printf("Received shutdown command, committing suicide")
	os.Exit(0)
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
