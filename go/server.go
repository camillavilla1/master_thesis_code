package main

import (
	"fmt"
	"flag"
	"log"
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

var batteryStart float64

var wg sync.WaitGroup

var biggestAddress string

type ObservationUnit struct {
	Addr string `json:"Addr"`
	Id uint32 `json:"Id"`
	Pid int `json:"Pid"`
	ReachableNeighbours []string `json:"-"`
	Neighbours []string `json:"-"`
	BatteryTime float64 `json:"-"`
	Xcor float64 `json:"Xcor"`
	Ycor float64 `json:"Ycor"`
	ClusterHead string `json:"-"`
	IsClusterHead bool `json:"-"`
	Bandwidth int `json:"-"`
	PathToCh []string `json:"-"`
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
		fmt.Println("Processes:", ret)
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


func startServer() {
	batteryStart = 3000.0
	hostaddress := ouHost + ouPort
	//startedNodes = append(startedNodes, hostaddress)
	
	log.Printf("Starting Observation Unit on %s\n", hostaddress)
	
	ou := &ObservationUnit{
        Addr:					hostaddress,
        Id:						hashAddress(hostaddress),
        Pid:					os.Getpid(),
        BatteryTime:			batteryStart,
        ReachableNeighbours:	[]string{},
        Neighbours:				[]string{},
        Xcor:					estimateLocation(),
        Ycor:					estimateLocation(),
    	ClusterHead:			"", 
		IsClusterHead:			false,
		Bandwidth:				bandwidth(),
		PathToCh:				[]string{}}


	//func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/reachableNeighbours", ou.reachableNeighboursHandler)
	http.HandleFunc("/newNeighbour", ou.newNeighboursHandler)
	http.HandleFunc("/noReachableNeighbours", ou.NoReachableNeighboursHandler)
	http.HandleFunc("/notifyCH", ou.NotifyCHHandler)
	http.HandleFunc("/OuClusterMember", ou.ouClusterMemberHandler)
	http.HandleFunc("/connectingOk", ou.connectionOkHandler)
	http.HandleFunc("/connectingToNeighbourOk", ou.connectingToNeighbourOkHandler)

	go ou.batteryConsumption(1)
	go ou.tellSimulationUnit()


	err := http.ListenAndServe(ouPort, nil)
	
	if err != nil {
		log.Panic(err)
	}

}


/*Receive neighbours from simulation. Contact neighbours to say "Hi, Here I am"*/
func (ou *ObservationUnit) reachableNeighboursHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n### ReachableNeighbours Handler ###\n")
	var tmpNeighbour []string

	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)
  
    //fmt.Printf(string(body))  

	if err := json.Unmarshal(body, &tmpNeighbour); err != nil {
        panic(err)
    }

    io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

    ou.ReachableNeighbours = tmpNeighbour

    //fmt.Printf("%+v\n", ou)

	time.Sleep(1000 * time.Millisecond)
	go ou.contactNeighbour()
}


/*There are no OUs in range of the OU.. Set OU as CH*/
func (ou *ObservationUnit) NoReachableNeighboursHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("\nNo neighbour Handler\n")
	var addrString string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &addrString)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	//fmt.Printf(addrString)

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	go ou.clusterHeadElection()
}



/*Receive a new neighbour from OU that wants to connect to the cluster/a neighbour. Need to send this to the leader/CH in the cluster.*/
func (ou *ObservationUnit) newNeighboursHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n### OU received a new neighbour (Handler) ###\n")
	var newNeighbour string
	//var data []string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &newNeighbour)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	fmt.Printf(newNeighbour)

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	//Notify CH to figure out if the OU can join the cluster. 
	if ou.Addr == ou.ClusterHead {
		fmt.Printf("OU is CH! Tell OU that it is a cluster member\n")
		ou.ReachableNeighbours = append(ou.ReachableNeighbours, newNeighbour)
		//fmt.Printf("OU is: ")
		//fmt.Println(ou)
		//fmt.Printf("\n")
		time.Sleep(1000 * time.Millisecond)
		go ou.tellOuClusterMember(newNeighbour)
	} else {
		fmt.Printf("\nOU is not CH so need forward info to CH!\n")
		time.Sleep(1000 * time.Millisecond)
		//data = append(data, newNeighbour)
		go ou.forwardNewOuToCh(newNeighbour)
	}

}


/*Decide if new OU can join the cluster or not..*/
func (ou *ObservationUnit) NotifyCHHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n### CH Handler receive info about new OU ###\n")

	var data []string
	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)
  
    fmt.Printf(string(body))  

	if err := json.Unmarshal(body, &data); err != nil {
        panic(err)
    }

    io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if ou.IsClusterHead {
		fmt.Printf("\nOU is CH so sends OK to OU\n")
		go ou.tellContactingOuOk(data)
	} else {
		go ou.tellContactingOuOk(data)
	}

	
	/*How to determine if the OU can join or not?? Should be about batteryLevel, bandwidth, number of nodes in the cluster etc..*/

}


/*Receive ok from CH that new OU can join.*/
func (ou *ObservationUnit) connectionOkHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n### Received OK from CH. Connect OU to new OU-neighbour OK (Handler)!!\n")

	var newOu string
	
	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)
  
    fmt.Printf(string(body))  

	if err := json.Unmarshal(body, &newOu); err != nil {
        panic(err)
    }

    //ou.ReachableNeighbours = append(ou.ReachableNeighbours, newOu)
    ou.Neighbours = append(ou.Neighbours, newOu)
    go ou.contactNewOu(newOu)


	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()
}


func (ou *ObservationUnit) connectingToNeighbourOkHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n### Receive from OU that it's OK to connection to neighbour OK (Handler)!! ###\n")

	var neighbour string
	
	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)
  
    //mt.Printf(string(body))  

	if err := json.Unmarshal(body, &neighbour); err != nil {
        panic(err)
    }

    if !listContains(ou.Neighbours, neighbour) {
    	ou.Neighbours = append(ou.Neighbours, neighbour)
    }

    fmt.Println(ou)

    io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()
}


func (ou *ObservationUnit) ouClusterMemberHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\nOU received  vaidation from CH and is now a Cluster Member (Handler)!!\n")
	var clusterHead string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &clusterHead)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	//fmt.Printf(clusterHead)

	/*Add to list over OUs neighbours that he can connect with..*/
	if !listContains(ou.Neighbours, clusterHead) {
		ou.Neighbours = append(ou.Neighbours, clusterHead)
	}

	ou.ClusterHead = clusterHead
	fmt.Println(ou)
	//fmt.Printf("\n")

	//go ou.clusterHeadElection() //???
	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()
}


func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Don't use the request body. But we should consume it anyway.
	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	fmt.Fprintf(w, "Index Handler\n")
}


func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	tellSimulationUnitDead()

	// Shut down
	log.Printf("Received shutdown command, committing suicide.")
	os.Exit(0)
}

func (ou *ObservationUnit) contactNewOu(newOu string) {
	fmt.Printf("\n### Tell contacting OU that it's OK to connect to neighbour ###\n")

	url := fmt.Sprintf("http://%s/connectingToNeighbourOk", newOu)
	fmt.Printf("Sending to url: %s \n", url)

	b, err := json.Marshal(ou.Addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println(string(b))

	addressBody := strings.NewReader(string(b))


	//fmt.Printf("\n")
	_, err = http.Post(url, "string", addressBody)
	errorMsg("Error posting to new OU about connection ok: ", err)
}


/*Tell contaction OU that it is ok to join the cluster*/
func (ou *ObservationUnit) tellContactingOuOk(data []string) {
	fmt.Printf("\n### Tell contacting Neighbour that ok to connect ###\n")

	recOu := strings.Join(data[:1],"") //first element
	//fmt.Println(url2)
	newOu := strings.Join(data[1:2],"") //middle element, nr 2
	//ouAddresses := strings.Join(data[2:],"") //all elements except the two first
 

	url := fmt.Sprintf("http://%s/connectingOk", recOu)
	fmt.Printf("Sending to url: %s", url)

	b, err := json.Marshal(newOu)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("\n")
	fmt.Println(string(b))

	addressBody := strings.NewReader(string(b))


	fmt.Printf("\n")
	_, err = http.Post(url, "string", addressBody)
	errorMsg("Error posting to neighbour about connection ok: ", err)
}

/*Contant neighbours in range with OUs address as body to tell that it wants to connect if possible. */
func (ou *ObservationUnit) contactNeighbour() {
	fmt.Printf("\n### Contacting neighbours.. ###\n")
	for _, neighbour := range ou.ReachableNeighbours {
		url := fmt.Sprintf("http://%s/newNeighbour", neighbour)
		fmt.Printf("\nContacting neighbour url: %s ", url)	
		fmt.Printf(" with body: %s \n", ou.Addr)

		addressBody := strings.NewReader(ou.Addr)

		//fmt.Printf("\n")
		_, err := http.Post(url, "string", addressBody)
		errorMsg("Error posting to neighbour ", err)
	}
}

func (ou *ObservationUnit) tellOuClusterMember(newNeighbour string) {
	fmt.Println("\n### Tell new OU that it is a cluster member! ###\n")
	url := fmt.Sprintf("http://%s/OuClusterMember", newNeighbour)
	fmt.Printf("Sending to url: %s", url)

	fmt.Printf(" with body: %s \n", ou.ClusterHead)
	addressBody := strings.NewReader(ou.ClusterHead)
	//fmt.Printf("\n")

	_, err := http.Post(url, "string", addressBody)
	errorMsg("Error posting to neighbour ", err)
}


/*Tell BS that node is up and running*/
func (ou *ObservationUnit) tellSimulationUnit() {
	fmt.Println("Battery is ", ou.BatteryTime)

	url := fmt.Sprintf("http://localhost:%s/notifySimulation", SimPort)
	fmt.Printf("Sending to url: %s with info about OU.\n", url)

	b, err := json.Marshal(ou)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println(string(b))

	addressBody := strings.NewReader(string(b))

	//fmt.Printf("\n")
	res, err := http.Post(url, "bytes", addressBody)
	errorMsg("POST request to Simulation failed: ", err)
	io.Copy(os.Stdout, res.Body)
}


/*Tell SOU that you're dead */
func tellSimulationUnitDead() {
	url := fmt.Sprintf("http://localhost:%s/removeReachableOu", SimPort)
	fmt.Printf("Sending 'I'm dead..' to url: %s", url)
	
	nodeString := ouHost + ouPort
	fmt.Printf("\nWith the string: %s\n", nodeString)

	addressBody := strings.NewReader(nodeString)

	//fmt.Printf("\n")
	_, err := http.Post(url, "string", addressBody)
	errorMsg("Post request dead OU: ", err)
}


func (ou *ObservationUnit) forwardNewOuToCh(newNeighbour string) {
	fmt.Printf("\n### OU received a new OU-neighbour and need to tell CH! ###\n")
	url := fmt.Sprintf("http://%s/notifyCH", ou.ClusterHead)
	fmt.Printf("Sending to url: %s \n", url)

	var data []string

	data = append(data, ou.Addr)
	data = append(data, newNeighbour)

	b, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Println(string(b))

	addressBody := strings.NewReader(string(b))

	//fmt.Printf("\n")
	_, err = http.Post(url, "string", addressBody)
	errorMsg("Post request info to CH: ", err)
}


/*Chose if node is the biggest and become chief..*/
func (ou *ObservationUnit) biggestId() bool {
	var biggest uint32

	for i, neighbour := range(ou.ReachableNeighbours) {
		hAddress := hashAddress(neighbour)

		if i == 0 {
			biggest = hAddress
			biggestAddress = neighbour
		} else {
			if hAddress > biggest {
				biggest = hAddress
				biggestAddress = neighbour
			}
		}
	}

	if biggest == ou.Id {
		return true
	} else {
		return false
	}
}


func (ou *ObservationUnit) clusterHeadElection() {
	fmt.Println("Battery is ", ou.BatteryTime)

	fmt.Printf("\n### Cluster Head Election ###\n")
	if len(ou.ReachableNeighbours) == 0 {
		fmt.Printf("No Reachable Neighbours.. Be your own CH!\n")
		ou.IsClusterHead = true
		ou.ClusterHead = ou.Addr
	} else {
		//if ou.ClusterHead 
		go ou.clusterHeadCalculation()
	}
}

func (ou *ObservationUnit) clusterHeadCalculation() {
	fmt.Printf("\n### Can OU become CH?? ####\n")
	fmt.Println(ou.Bandwidth)
	fmt.Println(ou.BatteryTime)
	fmt.Println(ou.Id)
	fmt.Println(len(ou.ReachableNeighbours))
	fmt.Printf("Implement an algorithm to evaluate if OU can be CH..\n\n")

	//if battery us under 20% cannot OU be CH
	if ou.BatteryTime < (batteryStart*0.20) {
		fmt.Printf("OU cannot be CH because of low battery\n")
	} else {
		fmt.Printf("Check if OU can be CH..\n")

	}

	//Algorithm to figure out if OU can be CH or not..
	//if OU can be CH, than OU should broadcast to its neighbours..

}

//Hash address to be ID of node
func hashAddress(address string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(address))
	hashedAddress := h.Sum32()
	return hashedAddress
}


func estimateLocation() float64 {
	rand.Seed(time.Now().UTC().UnixNano())
	//num := (rand.Float64() * 495) + 5
	num := (rand.Float64() * 195) + 5
	return num
}


func randEstimateBattery() float64 {
	//float64
	rand.Seed(time.Now().UTC().UnixNano())
	num := rand.Float64()

	return num
}


/*func (ou *ObservationUnit) batteryConsumption() {
	battery := 3000
	//battery := 100

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	done := make(chan bool)
	go func() {
		time.Sleep(10 * time.Second)
		done <- true
	}()
	for {
		select {
		case <-done:
			fmt.Println("Done!")
			return
		case t := <-ticker.C:
			fmt.Println("Current time: ", t)
			battery -= 1
			fmt.Println(battery)
			ou.BatteryTime = battery

		}
	}
}*/

func (ou *ObservationUnit) batteryConsumption(seconds int) {
	ou.BatteryTime -= float64(seconds)
	fmt.Println(ou.BatteryTime)
}


func simulateSleep() {
	timer := time.NewTimer(time.Second * 2)
    <- timer.C
    fmt.Println("Timer expired")
}


func setMaxProcs() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
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
func retrieveAddresses(list []string, addr string) []string {

	if listContains(list, addr) {
		fmt.Printf("List contains %s.\n", addr)
		return list
	} else {
		fmt.Printf("List do not contain %s, need to append.\n", addr)
		list = append(list, addr)
		return list
	}
}


func weather_sensor() {
	weather := make([]string, 0)
	weather = append(weather,
    "Sunny",
    "Cloudy",
    "Rain",
    "Windy",
    "Snow")

	rand.Seed(time.Now().UTC().UnixNano())
    rand_weather := weather[rand.Intn(len(weather))]
	fmt.Printf("\nRandom weather is: %s, ", rand_weather)
	//ou.weather = rand_weather
}


func temperature_sensor() {
	rand_number := randomInt(-30, 20)
	//ou.temperature = rand_number
	fmt.Println(rand_number)
}


/*Return a random int describing which bandwidth-type for specific OU.
Use this value to determine if OU can be CH*/
func bandwidth() int {
	bw := make(map[string]int)
	bw["LoRa"] = 50
	bw["Cable"] = 90
	bw["Wifi"] = 30

    i := rand.Intn(len(bw))
	var k string
	for k = range bw {
	  if i == 0 {
	    break
	  }
	  i--
	}

	//fmt.Println(k, bw[k])
	//fmt.Println(bw[k])
	return bw[k]

}