package main

import (
	"fmt"
	"flag"
	"log"
	"net"
	"os"
	"net/http"
	"io"
	"io/ioutil"
	"runtime"
	"sync"
)


var ouPort string
var ouHost string

//var hostname string
var hostaddress string

var startedOuServer []string

var wg sync.WaitGroup

func main() {

	var runMode = flag.NewFlagSet("run", flag.ExitOnError)
	addCommonFlags(runMode)
	//var runHost = runMode.String("host", "localhost", "Run host")

	if len(os.Args) == 1 {
		log.Fatalf("No mode specified\n")
	}

	switch os.Args[1] {
	case "run":
		runMode.Parse(os.Args[2:])
		wg.Add(1)
		ret := setMaxProcs()
		fmt.Println(ret)
		go startServer()
		wg.Wait()

	default:
		log.Fatalf("Unknown mode %q\n", os.Args[1])
	}

}

func GetLocalIP() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return ""
    }
    for _, address := range addrs {
        // check the address type and if it is not a loopback the display it
        if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String()
            }
        }
    }
    return ""
}

func addCommonFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&ouHost, "host", "localhost", "wormgate port (prefix with colon)")
	flagset.StringVar(&ouPort, "port", ":0", "segment port (prefix with colon)")
}


func errorMsg(s string, err error) {
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

	//getLocalIp()
	ret_val := GetLocalIP()
	fmt.Printf("Local IP is: %s\n", ret_val)
	
	pid := os.Getpid()
	fmt.Println("Process id is:", pid)

	pPid := os.Getppid()
	fmt.Println("Parent process id is:", pPid)


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
	log.Printf("Received shutdown command, committing suicide.")
	os.Exit(0)
}


func setMaxProcs() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}

func getLocalIp() {
	/*Interfaces returns a list of the system's network interfaces. */
	interfaces, err := net.Interfaces()
	errorMsg("Interface: ", err)
	fmt.Println(interfaces)

	/*for _, i := range interfaces {
		fmt.Printf("Name : %v \n", i.Name)
		// see http://golang.org/pkg/net/#Flags
		fmt.Println("Interface type and supports : ", i.Flags.String())
	}*/

	interface_addr, err := net.InterfaceAddrs()
	errorMsg("Interfaceaddr: ", err)
	fmt.Println(interface_addr)
}
