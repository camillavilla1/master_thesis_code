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

var SimPort string

var reachableHosts []string
var startedNodes []string //can be removed..

var biggestAddress string //can be removed?

var wg sync.WaitGroup

var clusterHead string

type ObservationUnit struct {
	Addr string `json:"Addr"`
	Id uint32 `json:"Id"`
	//Pid int `json:"Pid"`
	Neighbours []string `json:"-"`
	//LocationDistance float32
	BatteryTime float32 `json:"BatteryTime"`
	Xcor float64 `json:"Xcor"`
	Ycor float64 `json:"Ycor"`
	//clusterHead string 
	//isClusterHead bool
	//prevClusterHead string
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
	flagset.StringVar(&SimPort, "Simport", ":0", "Simulation (prefix with colon)")
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
	hostaddress := ouHost + ouPort
	startedNodes = append(startedNodes, hostaddress)
	
	log.Printf("Starting Observation Unit on %s%s\n", ouHost, ouPort)
	
	ou := &ObservationUnit{
        Addr:			hostaddress,
        Id:				hashAddress(hostaddress),
        BatteryTime:	0.4,
        Neighbours:		[]string{},
        Xcor:			estimateLocation(),
        Ycor:			estimateLocation(),}


	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/neighbour", ou.neighbourHandler)
	http.HandleFunc("/newNeighbour", ou.newNeighbourHandler)


	ou.tellSimulationUnit()


	err := http.ListenAndServe(ouPort, nil)
	
	if err != nil {
		log.Panic(err)
	}

}

func (ou *ObservationUnit) neighbourHandler2(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\nneighbourHandler2\n")
	var addrString string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &addrString)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	fmt.Printf("Address string is: ")
	fmt.Printf(addrString)
	
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	for _, addr := range(addrString) {
		if !listContains(ou.Neighbours, addrString) {
			fmt.Printf(string(addr))
			//ou.Neighbour = append(ou.Neighbour, addr)
			fmt.Printf("\nAdded neighbour to list..\n")
			printSlice(ou.Neighbours)

			//time.Sleep(1000 * time.Millisecond)
			ou.contactNeighbour()
		}
		
	}

}

func (ou *ObservationUnit) neighbourHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\nneighbourHandler\n")
	var tmpNeighbour []string

	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)
  
    fmt.Printf(string(body))  

	if err := json.Unmarshal(body, &tmpNeighbour); err != nil {
        panic(err)
    }

    io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

    ou.Neighbours = tmpNeighbour

    fmt.Printf("%+v\n", ou)

	for _, addr := range(ou.Neighbours) {
		fmt.Printf(string(addr))
		//ou.Neighbour = append(ou.Neighbour, addr)
		fmt.Printf("\nAdded neighbour to list..\n")
		printSlice(ou.Neighbours)

		//time.Sleep(1000 * time.Millisecond)
	}
	ou.contactNeighbour()
}



/*Receive a new neighbour from OU that wants to connect to the cluster/a neighbour. Need to send this to the leader/CH in the cluster.*/
func (ou *ObservationUnit) newNeighbourHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\nNew neighbour Handler\n")
	var addrString string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &addrString)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	fmt.Printf(addrString)

	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	ou.getInfoToCH()
}

func (ou *ObservationUnit) getInfoToCH() {
	fmt.Printf("\nGet info about new OU to CH!\n")
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


/*Contant neighbour with OUs address as body to tell that it wants to connect if possible. */
func (ou *ObservationUnit) contactNeighbour() {
	fmt.Printf("\nContacting neighbours..\n")
	for _, neighbour := range(ou.Neighbours) {
		url := fmt.Sprintf("http://%s/newNeighbour", neighbour)
		fmt.Printf("\nContacting neighbour url: %s ", url)	

		fmt.Printf("with body: %s", ou.Addr)
		addressBody := strings.NewReader(ou.Addr)

		_, err := http.Post(url, "string", addressBody)
		errorMsg("Error posting to neighbour ", err)
	}
}


/*Tell BS that node is up and running*/
func (ou *ObservationUnit) tellSimulationUnit() {
	url := fmt.Sprintf("http://localhost:%s/notifySimulation", SimPort)
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
	url := fmt.Sprintf("http://localhost:%s/removeReachablehost", SimPort)
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