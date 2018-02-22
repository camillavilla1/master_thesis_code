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
	//"bytes"
	"encoding/json"

)


var ouPort string
var ouHost string

var SOUPort string

var hostaddress string


var reachableHosts []string
var startedNodes []string

var biggestAddress string

var wg sync.WaitGroup

var clusterHead string

type ObservationUnit struct {
	Addr string
	Id uint32
	Pid int
	Neighbors []string
	Location int
	//clusterHead string
	//temperature int
	//weather string
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
		fmt.Printf("List do not contain %s, need to append.\n", addr)
		startedNodes = append(startedNodes, addr)
		return startedNodes
	}
}


func startServer() {
	ou := new(ObservationUnit)
	//func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/clusterHead", clusterHeadHandler)

	hostaddress = ouHost + ouPort
	startedNodes = append(startedNodes, hostaddress)
	
	log.Printf("Starting segment server on %s%s\n", ouHost, ouPort)

	//ou := ObservationUnit
	ou.Pid = os.Getpid()
	ou.Id = hashAddress(hostaddress)
	ou.Addr = hostaddress
	//fmt.Println("Process id is:", ou.Pid)
	//fmt.Println("Node id is:", ou.Id)
	fmt.Println(ou)

	tellBaseStationUnit(ou)

	reachableHosts = fetchReachablehosts()

	printSlice(startedNodes)

	log.Printf("Reachable hosts: %s", strings.Join(fetchReachablehosts()," "))

	go getRunningNodes()
	if clusterHeadElection(hostaddress) {
		fmt.Printf("I'm the CH!!")
		//tellCH()
		tellNodesaboutClusterHead()
	} else {
		//tellSuperObservationUnit()
		fmt.Printf("NOT CH!!")
	}

	err := http.ListenAndServe(ouPort, nil)
	if err != nil {
		log.Panic(err)
	}

}

func clusterHeadHandler(w http.ResponseWriter, r *http.Request) {
	var addrString string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &addrString)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	clusterHead = addrString

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



//Ping BS reachable host to check which nodes that are (dead or) alive
func getRunningNodes() {
	fmt.Printf("\nGET RUNNING NODES\n")

	for{
		url := fmt.Sprintf("http://localhost:%s/fetchReachablehosts", SOUPort)
		resp, err := http.Get(url)

		if err != nil {
			errorMsg("ERROR: ", err)
		}

		var bytes []byte
		bytes, err = ioutil.ReadAll(resp.Body)
		body := string(bytes)
		resp.Body.Close()

		/*TrimSpace returns a slice of the string s, with all leading and trailing white space removed, as defined by Unicode.*/
		trimmed := strings.TrimSpace(body)
		nodes := strings.Split(trimmed, "\n")

		printSlice(nodes)

		time.Sleep(4000 * time.Millisecond)
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

func tellBaseStationUnit(ou *ObservationUnit) {
	url := fmt.Sprintf("http://localhost:%s/reachablehosts", SOUPort)
	fmt.Printf("Sending to url: %s", url)
	

	b, err := json.Marshal(ou)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("####\n")
	fmt.Println(string(b))
	fmt.Printf("####\n")
//.................
	//ou := new(ObservationUnit)
	//body := ou
	/*b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(ou)
	fmt.Println(ou)
	fmt.Printf("####\n")
	fmt.Printf("\nWith body: %+v.\n", ou)
	fmt.Printf(".................\n")
	fmt.Sprintf("%s", b)
	fmt.Printf("........\n")
	*/
	addressBody := strings.NewReader(string(b))
	res, err := http.Post(url, "bytes", addressBody)
	errorMsg("POST request to SOU failed: ", err)
	io.Copy(os.Stdout, res.Body)
}

/*Tell SOU who you are with localhost:xxxx..*/
func tellBaseStationUnit2() {
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

func tellCH() {
	url := fmt.Sprintf("http://%s%s/chief", biggestAddress, ouPort)	
	message := "You're the CH!"
	addressBody := strings.NewReader(message)
	http.Post(url, "string", addressBody)
}

func tellNodesaboutClusterHead() {
	for _, addr := range startedNodes {
		url := fmt.Sprintf("http://%s/clusterHead", addr)
		fmt.Printf("\nTelling node %s about who is CH.", url)
		message := biggestAddress
		addressBody := strings.NewReader(message)
		http.Post(url, "string", addressBody)
	}
}


func clusterHeadElection(address string) bool {
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
		return true
	} else {
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
	//ObservationUnit.weather = rand_weather
}

func random(min, max int) int {
    rand.Seed(time.Now().Unix())
    return rand.Intn(max - min) + min
}

func temperature_sensor() {
	rand_number := random(-30, 20)
	//ObservationUnit.temperature = rand_number
	fmt.Println(rand_number)
}