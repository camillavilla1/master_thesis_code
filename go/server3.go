/*go run server3.go run -SOUport=8080 -host=localhost -port=:8083 */

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
	"strings"
	"math/rand"
	"time"
	"hash/fnv"
)


var ouPort string
var ouHost string

var SOUPort string

//var hostname string
var hostaddress string
//var actualSegments int32

var startedOuServer []string
var reachableHosts []string

var biggestAddress string

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
		//wg.Add(1)
		ret := setMaxProcs()
		fmt.Println(ret)
		//go startServer()
		//defer wg.Done()
		//wg.Wait()
		startServer()
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
	flagset.StringVar(&SOUPort, "SOUport", ":0", "Super OU port (prefix with colon)")
	flagset.StringVar(&ouHost, "host", "localhost", "OU host")
	flagset.StringVar(&ouPort, "port", ":8081", "OU port (prefix with colon)")
}


func errorMsg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

func printSlice(s []string) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}

//Check if a value is in a list/slice and return true or false
func listContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

//Get address, check if it is started nodes slice, if not: append the address
func retrieveAddresses(addr string) []string {

	if listContains(startedOuServer, addr) {
		fmt.Printf("List contains %s.\n", addr)
		return startedOuServer
	} else {
		fmt.Printf("List do not contain, need to append %s.\n", addr)
		startedOuServer = append(startedOuServer, addr)
		return startedOuServer
	}
}


func startServer() {
	//func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	//http.HandleFunc("/broadcastReachablehost", broadcastHandler)

	hostaddress = ouHost + ouPort
	startedOuServer = append(startedOuServer, hostaddress)
	
	log.Printf("Starting segment server on %s%s\n", ouHost, ouPort)
	//getLocalIp()
	//ret_val := GetLocalIP()
	//fmt.Printf("Local IP is: %s\n", ret_val)
	
	pid := os.Getpid()
	fmt.Println("Process id is:", pid)

	//pPid := os.Getppid()
	//fmt.Println("Parent process id is:", pPid)

	tellSuperObservationUnit()
	//Add this hashed address with hostaddress to a map/tuple?

	//for {
	reachableHosts = fetchReachablehosts()
	printSlice(reachableHosts)
	//time.Sleep(4000 * time.Millisecond)	
	//}

	hashAddr := hashAddress(hostaddress)
	clusterHead(hashAddr)
	fmt.Println("Hashed address is: ", hashAddr)
	//weather_sensor()
	//temperature_sensor()

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

	tellSuperObservationUnitDead()

	// Shut down
	log.Printf("Received shutdown command, committing suicide.")
	os.Exit(0)
}


/*
func broadcastHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("broadcastHandler\n")
	var addrString string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &addrString)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing broadcast (%d items): %s", pc, rateErr)
	}

	fmt.Printf(addrString)
	fmt.Printf("\n")
	stringList := strings.Split(addrString, ",")

	for _, addr := range stringList {
		startedOuServer = retrieveAddresses(addr)
	}
	actualSegments = int32(len(startedOuServer))

	//fmt.Println(actualSegments)
	printSlice(startedOuServer)


	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
}*/



func fetchReachablehosts() []string {
	fmt.Printf("fetchReachablehosts.\n")
	url := fmt.Sprintf("http://localhost:%s/fetchReachablehosts", SOUPort)
	resp, err := http.Get(url)

	if err != nil {
			return []string{}
	}

	var bytes []byte
	bytes, err = ioutil.ReadAll(resp.Body)
	body := string(bytes)
	resp.Body.Close()

	/*TrimSpace returns a slice of the string s, with all leading and trailing white space removed, as defined by Unicode.*/
	trimmed := strings.TrimSpace(body)
	nodes := strings.Split(trimmed, "\n")

	return nodes
}

/*Tell SOU who you are with localhost:xxxx..*/
func tellSuperObservationUnit() {
	url := fmt.Sprintf("http://localhost:%s/reachablehosts", SOUPort)
	fmt.Printf("Sending to url: %s", url)
	
	nodeString := ouHost + ouPort
	fmt.Printf("\nWith the string: %s.\n", nodeString)

	addressBody := strings.NewReader(nodeString)
	
	_, err := http.Post(url, "string", addressBody)
	errorMsg("POST request to SOU failed: ", err)
}

/*Tell SOU that you're dead */
func tellSuperObservationUnitDead() {
	url := fmt.Sprintf("http://localhost:%s/removeReachablehost", SOUPort)
	fmt.Printf("Sending 'I'm dead..' to url: %s", url)
	
	nodeString := ouHost + ouPort
	fmt.Printf("\nWith the string: %s\n", nodeString)

	addressBody := strings.NewReader(nodeString)
	
	_, err := http.Post(url, "string", addressBody)
	errorMsg("Dead Post address: ", err)
}


func clusterHead(address uint32) bool {
	var biggest uint32

	for i, addr := range startedOuServer {
		h := fnv.New32a()
		h.Write([]byte(addr))
		bs := h.Sum32()

		//bs := hashAddress(addr)

		if i == 0 {
			biggest = bs
			biggestAddress = addr
		} else {
			if bs > biggest {
				biggest = bs
				biggestAddress = addr
			}
		}
	}

	if biggest == address {
		fmt.Printf("I'm the CH\n")
		return true
	} else {
		fmt.Printf("I'm NOT CH\n")
		return false
	}
}

//Hash address to be ID of node
func hashAddress(address string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(address))
	hashedAddress := h.Sum32()
	return hashedAddress
}

func setMaxProcs() int {
	maxProcs := runtime.GOMAXPROCS(2)
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


func weather_sensor() {
	weather := make([]string, 0)
	weather = append(weather,
    "Sunny",
    "Cloudy",
    "Rain",
    "Windy",
    "Snow")

	rand.Seed(time.Now().Unix()) 
    rand_weather := weather[rand.Intn(len(weather))]
	fmt.Printf("\nRandom weather is: %s, ", rand_weather)
}


func random(min, max int) int {
    rand.Seed(time.Now().Unix())
    return rand.Intn(max - min) + min
}

func temperature_sensor() {
	rand_number := random(-30, 20)
	fmt.Println(rand_number)
}