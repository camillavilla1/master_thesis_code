package main

import (
	"fmt"
	"flag"
	"log"
	//"net"
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
	Addr string `json:"Addr"`
	Id uint32 `json:"Id"`
	//Pid int `json:"Pid"`
	Neighbors []string `json:"-"`
	//LocationDistance float32
	BatteryTime float32 `json:"BatteryTime"`
	Xcor float64 `json:"Xcor"`
	Ycor float64 `json:"Ycor"`
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


func addCommonFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&SOUPort, "Simport", ":0", "Simulation (prefix with colon)")
	flagset.StringVar(&ouHost, "host", "localhost", "OU host")
	flagset.StringVar(&ouPort, "port", ":8081", "OU port (prefix with colon)")
}

func randomInt(min, max int) int {
    rand.Seed(time.Now().UTC().UnixNano())
    return rand.Intn(max - min) + min
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
	//ou := new(ObservationUnit)
	//func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
	hostaddress = ouHost + ouPort
	startedNodes = append(startedNodes, hostaddress)
	
	log.Printf("Starting Observation Unit on %s%s\n", ouHost, ouPort)
	
	ou := &ObservationUnit{
        Addr:           hostaddress,
        Id:				hashAddress(hostaddress),
        BatteryTime:	0.4,
        Neighbors:		[]string{},
        Xcor:			estimateLocation(),
        Ycor:			estimateLocation(),}


	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/neighbor", ou.neighborHandler)


	ou.tellSimulationUnit()


	err := http.ListenAndServe(ouPort, nil)
	
	if err != nil {
		log.Panic(err)
	}

}

func (ou *ObservationUnit) neighborHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("NeighborHandler\n")
	var addrString string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &addrString)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	neighbor := addrString
	fmt.Printf(neighbor)

	ou.Neighbors = append(ou.Neighbors, addrString)
	fmt.Printf("\nAdded neighbor to list..\n")
	printSlice(ou.Neighbors)

	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
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

	tellSimulationUnitDead()

	// Shut down
	log.Printf("Received shutdown command, committing suicide.")
	os.Exit(0)
}


//Ping BS reachable host to check which nodes that are (dead or) alive
/*func getRunningNodes() {
	fmt.Printf("\nGET RUNNING NODES FROM BS\n")

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

		//TrimSpace returns a slice of the string s, with all leading and trailing white space removed, as defined by Unicode.
		trimmed := strings.TrimSpace(body)
		nodes := strings.Split(trimmed, "\n")

		printSlice(nodes)

		for _, addr := range nodes {
			if !listContains(startedNodes, addr) {
				startedNodes = append(startedNodes, addr)
			}
		}

		if clusterHeadElection(hostaddress) {
			fmt.Printf("I'm the CH!!\n")
		} else {
			fmt.Printf("I'm not the CH..\n")
		}

		time.Sleep(5000 * time.Millisecond)
	}	
}
*/


/*Tell BS that node is up and running*/
func (ou *ObservationUnit) tellSimulationUnit() {
	url := fmt.Sprintf("http://localhost:%s/notifySimulation", SOUPort)
	fmt.Printf("Sending to url: %s", url)

	b, err := json.Marshal(ou)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("\nwith data: ")
	fmt.Println(string(b))

	addressBody := strings.NewReader(string(b))

	res, err := http.Post(url, "bytes", addressBody)
	errorMsg("POST request to SOU failed: ", err)
	io.Copy(os.Stdout, res.Body)
}


/*Tell SOU that you're dead */
func tellSimulationUnitDead() {
	url := fmt.Sprintf("http://localhost:%s/removeReachablehost", SOUPort)
	fmt.Printf("Sending 'I'm dead..' to url: %s", url)
	
	nodeString := ouHost + ouPort
	fmt.Printf("\nWith the string: %s\n", nodeString)

	addressBody := strings.NewReader(nodeString)
	
	_, err := http.Post(url, "string", addressBody)
	errorMsg("Dead Post address: ", err)
}

/*func tellCH() {
	url := fmt.Sprintf("http://%s%s/chief", biggestAddress, ouPort)	
	message := "You're the CH!"
	addressBody := strings.NewReader(message)
	http.Post(url, "string", addressBody)
}*/
/*
func tellNodesaboutClusterHead() {
	for _, addr := range startedNodes {
		url := fmt.Sprintf("http://%s/clusterHead", addr)
		fmt.Printf("\nTelling node %s about who is CH.", url)
		message := biggestAddress
		addressBody := strings.NewReader(message)
		http.Post(url, "string", addressBody)
	}
}*/

/*Chose if node is the biggest and become chief..*/
/*func clusterHeadElection(address string) bool {
	var biggest uint32
	hAddress := hashAddress(address)

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

	if biggest == hAddress {
		return true
	} else {
		return false
	}
}*/


//Hash address to be ID of node
func hashAddress(address string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(address))
	hashedAddress := h.Sum32()
	return hashedAddress
}

func estimateLocation() float64 {
	//locDist := rand.Float32()
	//ou.LocationDistance = locDist
	//(rand.Float64() * 500) + 5
	rand.Seed(time.Now().UTC().UnixNano())
	num := (rand.Float64() * 495) + 5

	return num
}

/*func (ou *ObservationUnit) estimateLocation() {
	//locDist := rand.Float32()
	//ou.LocationDistance = locDist
	//(rand.Float64() * 500) + 5
	rand.Seed(time.Now().UTC().UnixNano())
	ou.Xcor = (rand.Float64() * 495) + 5
	ou.Ycor = (rand.Float64() * 495) + 5
}
*/
/**/
func estimateThreshold() {

}

func estimateBattery() {
	//float32
	
}

func setMaxProcs() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
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


func temperature_sensor() {
	rand_number := randomInt(-30, 20)
	//ObservationUnit.temperature = rand_number
	fmt.Println(rand_number)
}