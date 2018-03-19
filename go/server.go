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
	"math"
)

var ouPort string
var ouHost string

var SimPort string

var reachableHosts []string

var batteryStart int64
var secondInterval int64

var wg sync.WaitGroup

var biggestAddress string

type ObservationUnit struct {
	Addr string `json:"Addr"`
	ID uint32 `json:"Id"`
	Pid int `json:"Pid"`
	ReachableNeighbours []string `json:"-"`
	Neighbours []string `json:"-"`
	BatteryTime int64 `json:"-"`
	Xcor float64 `json:"Xcor"`
	Ycor float64 `json:"Ycor"`
	ClusterHead string `json:"-"`
	IsClusterHead bool `json:"-"`
	ClusterHeadCount int `json:"-"`
	Bandwidth int `json:"-"`
	PathToCh []string `json:"-"`
	CHpercentage float64 `json:"-"`
	//prevClusterHead string
	Temperature []int `json:"-"`
	Weather []string `json:"-"`
}

type CHpkt struct{
	Path []string
	Source string
	Destination string
	ClusterHead string
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
	/*1800 = 30 min, 3600 in 60 min*/
	batteryStart = 100
	secondInterval = 1
	hostaddress := ouHost + ouPort
	
	log.Printf("Starting Observation Unit on %s\n", hostaddress)


	ou := &ObservationUnit{
        Addr:					hostaddress,
        ID:						hashAddress(hostaddress),
        Pid:					os.Getpid(),
        BatteryTime:			batteryStart,
        ReachableNeighbours:	[]string{},
        Neighbours:				[]string{},
        Xcor:					estimateLocation(),
        Ycor:					estimateLocation(),
    	ClusterHead:			"", 
		IsClusterHead:			false,
		ClusterHeadCount:		0,
		Bandwidth:				bandwidth(),
		PathToCh:				[]string{},
		CHpercentage:			0,
		Temperature:			[]int{},
		Weather:				[]string{}}


	//func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/shutdown", ou.shutdownHandler)
	http.HandleFunc("/reachableNeighbours", ou.reachableNeighboursHandler)
	http.HandleFunc("/clusterheadPercentage", ou.clusterheadPercentageHandler)
	http.HandleFunc("/newNeighbour", ou.newNeighboursHandler)
	http.HandleFunc("/noReachableNeighbours", ou.NoReachableNeighboursHandler)
	http.HandleFunc("/connectingOk", ou.connectingOkHandler)
	http.HandleFunc("/broadcastNewLeader", ou.broadcastNewLeaderHandler)


	go ou.batteryConsumption()
	go ou.tellSimulationUnit()
	go ou.measureData()

	err := http.ListenAndServe(ouPort, nil)
	
	if err != nil {
		log.Panic(err)
	}
}

/*Get cluster percentage from Simulation.*/
func (ou *ObservationUnit) clusterheadPercentageHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("\n### Receiving cluster head percentage from Simulator ###\n")
	var chPercentage float64

	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &chPercentage); err != nil {
        panic(err)
    }

    io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	ou.CHpercentage = chPercentage
}

/*Receive neighbours from simulation. Contact neighbours to say "Hi, Here I am"*/
func (ou *ObservationUnit) reachableNeighboursHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n### ReachableNeighbours Handler ###\n")
	var tmpNeighbour []string

	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &tmpNeighbour); err != nil {
        panic(err)
    }

    io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

    ou.ReachableNeighbours = tmpNeighbour

	go ou.contactNewNeighbour()
}


/*There are no OUs in range of the OU..*/
func (ou *ObservationUnit) NoReachableNeighboursHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n### OU received no neighbour (Handler) ###\n")
	var addrString string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &addrString)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	go ou.clusterHeadElection()
}



/*Receive a new neighbour from OU that wants to connect to the cluster/a neighbour.*/
func (ou *ObservationUnit) newNeighboursHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n### OU received a new neighbour (Handler) ###\n")
	var newNeighbour string
	//var data []string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &newNeighbour)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	fmt.Println("New neighbour is:", newNeighbour)

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	var data []string

	data = append(data, ou.Addr)
	data = append(data, newNeighbour)
	data = append(data, ou.ClusterHead)

	//fmt.Printf("\n[ou.addr, newNeighbour, clusterhead]\n")

	if !listContains(ou.Neighbours, newNeighbour) {
		ou.Neighbours = append(ou.Neighbours, newNeighbour)
	}

	time.Sleep(1000 * time.Millisecond)
	go ou.tellContactingOuOk(data)
	fmt.Println(ou)

}

/*Receive a new leader from a neighbour. Update clusterhead and clusterhead-status*/
func (ou *ObservationUnit) broadcastNewLeaderHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n### Receive a broadcast about a new leader ###\n")
	var pkt CHpkt
	
	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &pkt); err != nil {
        panic(err)
    }

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if !listContains(pkt.Path, ou.Addr) {
		pkt.Path = append(pkt.Path, ou.Addr)
	}
	ou.ClusterHead = pkt.ClusterHead
	ou.IsClusterHead = false

	fmt.Println("OU: ", ou)
	//fmt.Println("\nPKT: ", pkt)

	if len(ou.Neighbours) > 1 {
		go ou.broadcastNewLeader(pkt)
		
	}

}

/*Receive ok from CH that new OU can join.*/
func (ou *ObservationUnit) connectingOkHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n### Received OK from neighbour. Connect OU to new neighbour!\n")

	var data []string
	
	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)
  
  	//fmt.Printf("\n[ou.neighbour, yourself, clusterhead]\n")
    //fmt.Printf(string(body))  
    //fmt.Printf("\n")

	if err := json.Unmarshal(body, &data); err != nil {
        panic(err)
    }

    neighOu := strings.Join(data[:1],"") //first element
	//fmt.Println(url2)
	//newOu := strings.Join(data[1:2],"") //middle element, nr 2
	clusterHead := strings.Join(data[2:],"")

	if !listContains(ou.Neighbours, neighOu) {
    	ou.Neighbours = append(ou.Neighbours, neighOu)
	}

	ou.ClusterHead = clusterHead
    fmt.Println(ou)


	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

}


func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Don't use the request body. But we should consume it anyway.
	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	fmt.Fprintf(w, "Index Handler\n")
}


func (ou *ObservationUnit) shutdownHandler(w http.ResponseWriter, r *http.Request) {
	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	ou.tellSimulationUnitDead()

	// Shut down
	log.Printf("Received shutdown command, committing suicide.")
	os.Exit(0)
}


/*Tell contaction OU that it is ok to join the cluster*/
func (ou *ObservationUnit) tellContactingOuOk(data []string) {
	fmt.Printf("\n### Tell contacting Neighbour that it's ok to connect ###\n")

	//recOu := strings.Join(data[:1],"") //first element
	newOu := strings.Join(data[1:2],"") //middle element, nr 2
	//clusterHead := strings.Join(data[2:],"") //all elements except the two first
 

	url := fmt.Sprintf("http://%s/connectingOk", newOu)
	fmt.Printf("Sending to url: %s", url)

	b, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Printf("\n")
	//fmt.Println(string(b))

	addressBody := strings.NewReader(string(b))


	fmt.Printf("\n")
	_, err = http.Post(url, "string", addressBody)
	errorMsg("Error posting to neighbour about connection ok: ", err)
}

/*Contant neighbours in range with OUs address as body to tell that it wants to connect */
func (ou *ObservationUnit) contactNewNeighbour() {
	fmt.Printf("\n### Contacting neighbours.. ###\n")
	var i int
	for _, neighbour := range ou.ReachableNeighbours {
		i += 1
		url := fmt.Sprintf("http://%s/newNeighbour", neighbour)
		fmt.Printf("\nContacting neighbour url: %s ", url)	
		fmt.Printf(" with body: %s \n", ou.Addr)

		addressBody := strings.NewReader(ou.Addr)

		//fmt.Printf("\n")
		_, err := http.Post(url, "string", addressBody)
		//http.Post(url, "string", addressBody)
		//errorMsg("Error posting to neighbour ", err)
		if err != nil {
			continue
		}
	}
	//fmt.Printf("\nContacted all neighbours.. Some may be unreachable due to death or sleep..\n")

	if i == len(ou.ReachableNeighbours) {
		fmt.Printf("Have contacted all neighbours, either sucessfully or unsucessfully.. Check if node can be cluster head.\n")
		time.Sleep(2 * time.Second)
		go ou.clusterHeadElection()
	}
}

/*Broadcast new CH message to neighbours*/
func (ou *ObservationUnit) broadcastNewLeader(pkt CHpkt) {
	fmt.Printf("### Broadcast new leader to neighbours ###\n")
    //var pkt CHpkt

	for _, addr := range ou.Neighbours {
		if !listContains(pkt.Path, addr) {
			url := fmt.Sprintf("http://%s/broadcastNewLeader", addr)
			fmt.Printf("\nContacting neighbour url: %s ", url)

			if !listContains(pkt.Path, ou.Addr) {
				pkt.Path = append(pkt.Path, ou.Addr)
			}
			pkt.Source = ou.Addr
			pkt.Destination = addr
			//but what if you're a cluster head from before??
			/*if ou.IsClusterHead == true {
				pkt.ClusterHead = ou.ClusterHead
			}*/

			b, err := json.Marshal(pkt)
			if err != nil {
				fmt.Println(err)
				return
			}

			addressBody := strings.NewReader(string(b))
			//fmt.Println("\nAddressbody: ", addressBody)

			_, err = http.Post(url, "string", addressBody)
			//errorMsg("Error posting to neighbour ", err)
			if err != nil {
				continue
			}
		}
	}
}


/*Tell Simulation that node is up and running*/
func (ou *ObservationUnit) tellSimulationUnit() {
	//fmt.Println("Battery is ", ou.BatteryTime)

	url := fmt.Sprintf("http://localhost:%s/notifySimulation", SimPort)
	fmt.Printf("Sending to url: %s with info about OU.\n", url)

	b, err := json.Marshal(ou)
	if err != nil {
		fmt.Println(err)
		return
	}

	addressBody := strings.NewReader(string(b))

	res, err := http.Post(url, "bytes", addressBody)
	errorMsg("POST request to Simulation failed: ", err)
	io.Copy(os.Stdout, res.Body)
}


/*Tell SOU that you're dead */
func (ou *ObservationUnit) tellSimulationUnitDead() {
	url := fmt.Sprintf("http://localhost:%s/removeReachableOu", SimPort)
	fmt.Printf("Sending 'I'm dead..' to url: %s \n", url)
	
	b, err := json.Marshal(ou)
	if err != nil {
		fmt.Println(err)
		return
	}

	addressBody := strings.NewReader(string(b))

	_, err = http.Post(url, "string", addressBody)
	errorMsg("Post request dead OU: ", err)
}


func saveBatterytime() {
	fmt.Printf("Sleeping...\n")
	time.Sleep(5 * time.Second)
}

func (ou *ObservationUnit) shutdownOu() {
	fmt.Printf("Low battery, shutting down..\n")
	ou.tellSimulationUnitDead()
	os.Exit(0)
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

	if biggest == ou.ID {
		return true
	} else {
		return false
	}
}


func (ou *ObservationUnit) clusterHeadElection() {
	fmt.Printf("\n### Cluster Head Election ###\n")
	if len(ou.ReachableNeighbours) == 0 {
		fmt.Printf("No Reachable Neighbours.. Be your own CH!\n")
		ou.IsClusterHead = true
		ou.ClusterHead = ou.Addr
		ou.ClusterHeadCount += 1
	} else {
		//if ou.ClusterHead 
		go ou.clusterHeadCalculation()
	}
}

func (ou *ObservationUnit) clusterHeadCalculation() {
	fmt.Printf("\n### CLUSTER HEAD CALCULATION ####\n")
	var pkt CHpkt

	//if battery us under 20% cannot OU be CH
	if float64(ou.BatteryTime) < (float64(batteryStart)*0.20) {
		fmt.Printf("OU cannot be CH because of low battery\n")
	} else {
		fmt.Printf("Check if OU can be CH..\n")
		//batPercent := calcPercentage(ou.BatteryTime, batteryStart)
		randNum := randomFloat()
		threshold := ou.threshold()

		//if randNum < threshold
		if randNum > threshold {
			fmt.Printf("OU can be CH because of threshold..\n")
			ou.ClusterHeadCount += 1
			ou.ClusterHead = ou.Addr
			ou.IsClusterHead = true
			pkt.ClusterHead = ou.Addr
			go ou.broadcastNewLeader(pkt)
		} else {
			fmt.Printf("OU can not be CH because of threshold..\n")
		}
		time.Sleep(5 * time.Second)
	}
}


func randomFloat() float64 {
	num := rand.Float64()
	return num
}


func (ou ObservationUnit) threshold() float64 {
	threshold := ou.CHpercentage/1-(ou.CHpercentage*(math.Mod(float64(ou.ClusterHeadCount), 1/float64(ou.ClusterHeadCount))))
	return threshold
}


/*Calculate battery percentage on OU*/
func calcPercentage(batteryTime int64, maxBattery int64) float64 {
	ret := (float64(batteryTime)/float64(maxBattery))*100
	return ret
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


func (ou *ObservationUnit) batteryConsumption() {
	//timeChan := time.NewTimer(time.Second).C
	tickChan := time.NewTicker(time.Millisecond * 1000).C

	doneChan := make(chan bool)
    go func() {
        time.Sleep(time.Second * time.Duration(batteryStart))
        doneChan <- true
    }()
    
    for {
        select {
        //case <- timeChan:
        //    fmt.Println("Timer expired.\n")
        case <- tickChan:
            //fmt.Println("Ticker ticked")
            //fmt.Println(secondInterval)
		    ou.BatteryTime -= secondInterval
		    //fmt.Println(ou.BatteryTime)

		    if ou.BatteryTime == 0 {
		    	//fmt.Printf("Batterytime is 0..\n")
		    } else if float64(ou.BatteryTime) <= (float64(batteryStart)*0.20) {
		    	fmt.Printf("OU have low battery.. Need to sleep to save power\n")
		    	//saveBatterytime()
		    }
        case <- doneChan:
        	ou.BatteryTime = 0
            fmt.Println("Done. OU is dead..\n")
            go ou.shutdownOu()
            return
      }
    }
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


func (ou *ObservationUnit) measureData() {
	tickChan := time.NewTicker(time.Second * 3).C

	doneChan := make(chan bool)
    go func() {
        time.Sleep(time.Second * time.Duration(batteryStart))
        doneChan <- true
    }()
    
    for {
        select {
        //case <- timeChan:
        //    fmt.Println("Timer expired.\n")
        case <- tickChan:
        	start := time.Now().Format("2006-01-02 15:04:05")//.Format(time.RFC850)
        	fmt.Println(start)
        	ou.temperatureSensor()
        	ou.weatherSensor()

        case <- doneChan:
            return
      }
    }
}


func (ou *ObservationUnit) weatherSensor() {
	weather := make([]string, 0)
	weather = append(weather,
    "Sunny",
    "Cloudy",
    "Rain",
    "Windy",
    "Snow")

	rand.Seed(time.Now().UTC().UnixNano())
    rand_weather := weather[rand.Intn(len(weather))]
	ou.Weather = append(ou.Weather, rand_weather)
	fmt.Println("Weather: ", ou.Weather)
}


func (ou *ObservationUnit) temperatureSensor() {
	rand_number := randomInt(-30, 20)
	ou.Temperature = append(ou.Temperature, rand_number)
	//fmt.Println(rand_number)
	fmt.Println("Temperature: ", ou.Temperature)
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