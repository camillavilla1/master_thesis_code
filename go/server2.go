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


var hostaddress string

var actualSegments int32

//var startedOuServer []string

var reachableHosts []string
var startedNodes []string

var biggestAddress string

var wg sync.WaitGroup

var ObservationUnit struct {
	id int
	addr string
	neighbors []string
	clusterHead string
	temperature int
	weather string
	location int
}

func main() {

	var runMode = flag.NewFlagSet("run", flag.ExitOnError)
	addCommonFlags(runMode)

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

	if listContains(startedNodes, addr) {
		fmt.Printf("List contains %s.\n", addr)
		return startedNodes
	} else {
		fmt.Printf("List do not contain, need to append %s.\n", addr)
		startedNodes = append(startedNodes, addr)
		return startedNodes
	}
}


func startServer() {
	//func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/shutdown", shutdownHandler)

	hostaddress = ouHost + ouPort
	startedNodes = append(startedNodes, hostaddress)
	
	log.Printf("Starting segment server on %s%s\n", ouHost, ouPort)

	
	pid := os.Getpid()
	fmt.Println("Process id is:", pid)

	//tellBaseStationUnit()

	if clusterHead(hostaddress) {
		tellBaseStationUnit()
	} /*else {
		tellSuperObservationUnit()
	}*/

	reachableHosts = fetchReachablehosts()


	actualSegments = int32(len(startedNodes))
	printSlice(startedNodes)

	log.Printf("Reachable hosts: %s", strings.Join(fetchReachablehosts()," "))

	go heartbeat()


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


//Ping all reachable host to check if dead or alive
func heartbeat() {
	for {
		for _, addr := range reachableHosts {
			url := fmt.Sprintf("http://%s", addr)
			fmt.Printf("HEARTBEAT URL IS: %s.\n", url)
			if addr != hostaddress {
				resp, err := http.Get(url)
				if err != nil {
					if listContains(startedNodes, addr) {
						fmt.Printf("Contains in list\n")
						clusterHead(hostaddress)
					}
				} else {
					fmt.Printf("Does not contain in list\n")
					_, err = ioutil.ReadAll(resp.Body)
					startedNodes = retrieveAddresses(addr)
					resp.Body.Close()
				}
			}
		}	
		time.Sleep(2000 * time.Millisecond)
	}
}

func fetchReachablehosts() []string {
	fmt.Printf("\n### fetchReachablehosts ###\n")
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
func tellBaseStationUnit() {
	url := fmt.Sprintf("http://localhost:%s/reachablehosts", SOUPort)
	fmt.Printf("Sending to url: %s", url)
	
	nodeString := ouHost + ouPort
	fmt.Printf("\nWith the string: %s.\n", nodeString)

	addressBody := strings.NewReader(nodeString)
	
	_, err := http.Post(url, "string", addressBody)
	errorMsg("POST request to SOU failed: ", err)
}

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


func clusterHead(address string) bool {
	var biggest uint32

	for i, addr := range startedNodes {
		bs := hashAddress(addr)

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

	hAddress := hashAddress(address)
	if biggest == hAddress {
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