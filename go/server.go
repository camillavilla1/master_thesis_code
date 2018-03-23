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


type SensorData struct {
	Weather []string
	Temperature []int
	DateTime []string
	Destination string
	Source string
	Path []string
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
		fmt.Printf("\n\n----------------------------------------------------------------------------\n")
		fmt.Println("Processes:", ret)
		go startServer()
		wg.Wait()

	default:
		log.Fatalf("Unknown mode %q\n", os.Args[1])
	}
}

/*func (ou *ObservationUnit) write_to_file() {
	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile("test.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write([]byte("appended some data\n")); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}*/


func addCommonFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&SimPort, "Simport", ":0", "Simulation (prefix with colon)")
	flagset.StringVar(&ouHost, "host", "localhost", "OU host")
	flagset.StringVar(&ouPort, "port", ":8081", "OU port (prefix with colon)")
}


func startServer() {
	/*1800 = 30 min, 3600 in 60 min*/
	batteryStart = 500
	secondInterval = 1
	hostaddress := ouHost + ouPort
	
	log.Printf("Starting Observation Unit on %s\n", hostaddress)

	sensorData := &SensorData {
		Weather:		[]string{},
		Temperature:	[]int{},
		DateTime:		[]string{},
		Destination:	"",
		Source:			"",
		Path:			[]string{}}

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
	//http.HandleFunc("/findPathToLeader", ou.findPathToLeaderHandler)
	http.HandleFunc("/foundPathToLeader", ou.foundPathToLeaderHandler)
	


	go ou.batteryConsumption()
	go ou.tellSimulationUnit()
	go sensorData.measureSensorData()
	//go ou.getData(sensorData)

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
	//fmt.Printf("\n\nOU IS %s\n", ou.Addr)
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
	//fmt.Printf("\n\nOU IS %s\n", ou.Addr)
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
	//fmt.Printf("OU IS %s\n", ou.Addr)
	//fmt.Printf("### OU received a new neighbour (Handler) ###\n")
	var newNeighbour string
	//var data []string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &newNeighbour)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	//fmt.Println("New neighbour is:", newNeighbour)

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
	//fmt.Println(ou)

}

/*Receive a new leader from a neighbour. Update clusterhead and clusterhead-status*/
func (ou *ObservationUnit) broadcastNewLeaderHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\nOU IS %s\n", ou.Addr)
	fmt.Printf("### Receive a broadcast about a new leader ###\n")
	var pkt CHpkt
	
	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &pkt); err != nil {
        panic(err)
    }

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if ou.ClusterHead == "" || ou.ClusterHead != pkt.ClusterHead {
		ou.ClusterHead = pkt.ClusterHead
		ou.IsClusterHead = false

		fmt.Printf("[%s new CH is: %s]\n", ou.Addr, ou.ClusterHead)

		fmt.Printf("Update %s path\n", ou.Addr)
		if len(ou.PathToCh) == 0 {
			ou.PathToCh = pkt.Path
		} else if len(ou.PathToCh) >= len(pkt.Path) {
			ou.PathToCh = pkt.Path
		}
		fmt.Printf("\n%s path to CH is now: %v", ou.Addr, ou.PathToCh)

		if len(ou.Neighbours) > 1 {
			go ou.broadcastNewLeader(pkt)
		/*	for _, addr := range ou.Neighbours {
				if  addr != ou.ClusterHead {
					fmt.Printf("%s have neighbours. Send to %s\n", ou.Addr, addr)
					fmt.Printf("%s is not ch (%s)\n", addr, ou.ClusterHead)
					
					pkt.Path = append(pkt.Path, ou.Addr)
					fmt.Println("Packet is: ", pkt)
					//if !listContains(pkt.Path, addr) {
					go ou.broadcastNewLeader(pkt)
					//} else {
					//	continue
					//}
				}
			}*/
		}

	}


}


/*func (ou *ObservationUnit) findPathToLeaderHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\nOU IS %s\n", ou.Addr)
	fmt.Printf("### Find path to leader Handler ###\n")
	var pkt CHpkt

	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &pkt); err != nil {
        panic(err)
    }

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	for _, addr := range ou.Neighbours {
		fmt.Printf("OU CH is: %s while pkt-CH is: %s\n", ou.ClusterHead, pkt.ClusterHead)
		if ou.ClusterHead == addr {
			//fmt.Printf("%s neighbour is CH!\n", ou.Addr)
			if !listContains(pkt.Path, ou.Addr) {
				pkt.Path = append(pkt.Path, ou.Addr)
			}

			if !listContains(pkt.Path, addr) {
				pkt.Path = append(pkt.Path, addr)
			}
			//tell OU about path to CH..
			go ou.foundPathToLeader(pkt)
			break

		} else {
			fmt.Printf("%s neighbour is not CH. Neighbour-ch is %s.. Need to forward to next neighbours..\n", addr, ou.ClusterHead)
			fmt.Println("OU.path is: ", ou.PathToCh)

			if len(ou.PathToCh) == 0 {
				fmt.Printf("OU-path is not empty, so have path to CH..\n")
			} else {
				fmt.Printf("Path is empty.. Need to forward to next neighbour..\n")
				go ou.findPathToLeader(pkt)
			}
			//time.Sleep(20 * time.Second)
			//go ou.findPathToLeader(pkt)
		}
	}

	fmt.Println("Pkt-path: ", pkt.Path)
}*/

func (ou *ObservationUnit) foundPathToLeaderHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\nOU IS %s\n", ou.Addr)
	fmt.Printf("### Found path to leader Handler ###\n")
	var pkt CHpkt

	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &pkt); err != nil {
        panic(err)
    }

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if len(ou.PathToCh) == 0 {
		fmt.Printf("%s path to ch is empty.. Adding path", ou.Addr)
		ou.PathToCh = pkt.Path

	} else if len(ou.PathToCh) > len(pkt.Path) {
		fmt.Printf("OU path is longer than pkt-path, so change to shortest path\n")
		ou.PathToCh = pkt.Path
	} else {
		fmt.Printf("Path is not empty or longer than new path..\n")
	}

	fmt.Println("OUs path to CH is: ", ou.PathToCh)
}


/*Receive ok from CH that new OU can join.*/
func (ou *ObservationUnit) connectingOkHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\nOU IS %s\n", ou.Addr)
	fmt.Printf("### Received OK from neighbour. Connect OU to new neighbour!\n")

	var data []string
	
	body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)
  

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
    //fmt.Println(ou)


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
	//fmt.Printf("\nOU IS %s\n", ou.Addr)
	//fmt.Printf("### Tell contacting Neighbour that it's ok to connect ###\n")

	//recOu := strings.Join(data[:1],"") //first element
	newOu := strings.Join(data[1:2],"") //middle element, nr 2
	//clusterHead := strings.Join(data[2:],"") //all elements except the two first
 

	url := fmt.Sprintf("http://%s/connectingOk", newOu)
	//fmt.Printf("Sending to url: %s\n", url)

	b, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return
	}

	//fmt.Printf("\n")
	//fmt.Println(string(b))

	addressBody := strings.NewReader(string(b))


	//fmt.Printf("\n")
	_, err = http.Post(url, "string", addressBody)
	errorMsg("Error posting to neighbour about connection ok: ", err)
}

/*Contant neighbours in range with OUs address as body to tell that it wants to connect */
func (ou *ObservationUnit) contactNewNeighbour() {
	//fmt.Printf("\nOU IS %s\n", ou.Addr)
	//fmt.Printf("### Contacting neighbours.. ###\n")
	var i int
	for _, neighbour := range ou.ReachableNeighbours {
		i += 1
		url := fmt.Sprintf("http://%s/newNeighbour", neighbour)
		//fmt.Printf("\nContacting neighbour url: %s ", url)	
		//fmt.Printf(" with body: %s \n", ou.Addr)

		addressBody := strings.NewReader(ou.Addr)

		//fmt.Printf("\n")
		_, err := http.Post(url, "string", addressBody)
		if err != nil {
			fmt.Printf("(%s) Try to post broadcast %s\n", ou.Addr, err)
			continue
		}
	}

	if i == len(ou.ReachableNeighbours) {
		time.Sleep(2 * time.Second)
		go ou.clusterHeadElection()
	}
}

/*Broadcast new CH message to neighbours*/
func (ou *ObservationUnit) broadcastNewLeader(pkt CHpkt) {
	fmt.Printf("\nOU IS %s\n", ou.Addr)
	fmt.Printf("### Broadcast new leader to neighbours ###\n")
    //var pkt CHpkt
	if !listContains(pkt.Path, ou.Addr) {
		fmt.Printf("\nAppending (%s) to the pkt-path\n", ou.Addr)
		pkt.Path = append(pkt.Path, ou.Addr)
	}

	fmt.Println("%s neighbours: %v", ou.Addr, ou.Neighbours)
	for _, addr := range ou.Neighbours {
		if !listContains(pkt.Path, addr) {
			fmt.Printf("(%s)Contact %s because it's not in pkt-path %v..\n", ou.Addr, addr, pkt.Path)
			url := fmt.Sprintf("http://%s/broadcastNewLeader", addr)
			fmt.Printf("\nContacting neighbour url: %s ", url)

			pkt.Source = ou.Addr
			pkt.Destination = addr

			fmt.Printf("\n")

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
				fmt.Printf("(%s) Try to post broadcast %s\n", ou.Addr, err)
				continue
			}
		} else {
			fmt.Printf("Address %s is already in pkt-path. Do not need to contact..\n", addr)
			continue
		}
	}

}


/*func (ou *ObservationUnit) findPathToLeader(pkt CHpkt) {
	fmt.Printf("\nOU IS %s\n", ou.Addr)
	fmt.Printf("### Find path to leader! ### \n")
	fmt.Printf("[path], source, destination, clusterhead\n")
	fmt.Println(pkt)

	for _, addr := range ou.Neighbours {

		if ou.ClusterHead == addr {
			fmt.Printf("Neighbour is CH, so no need for contact..\n")
			if !listContains(ou.PathToCh, addr) {
				ou.PathToCh = append(ou.PathToCh, addr)
			}
			fmt.Println("Path to CH is: ", ou.PathToCh)
			break
		} else {
			url := fmt.Sprintf("http://%s/findPathToLeader", addr)
			fmt.Printf("\nContacting neighbour url: %s ", url)

			pkt.ClusterHead = ou.ClusterHead
			pkt.Source = ou.Addr
			pkt.Destination = addr

			b, err := json.Marshal(pkt)
			if err != nil {
				fmt.Println(err)
				return
			}

			addressBody := strings.NewReader(string(b))

			_, err = http.Post(url, "string", addressBody)
			//errorMsg("Error posting to neighbour ", err)
			if err != nil {
				continue
			}
		}

	}
}*/


func (ou *ObservationUnit) foundPathToLeader(pkt CHpkt) {
	fmt.Printf("\nOU IS %s\n", ou.Addr)
	fmt.Printf("PKT SOURCE: %s", pkt.Source)
	url := fmt.Sprintf("http://%s/foundPathToLeader", pkt.Source)
	fmt.Printf("\nContacting neighbour url: %s ", url)

	b, err := json.Marshal(pkt)
	if err != nil {
		fmt.Println(err)
		return
	}

	addressBody := strings.NewReader(string(b))

	http.Post(url, "string", addressBody)
}

/*How to broadcast to neighbours with/without list of path... and how to receive??*/
func (ou *ObservationUnit) NotifyNeighbours(sensorData *SensorData) {
	fmt.Printf("\nOU IS %s\n", ou.Addr)

	for _, addr := range ou.Neighbours {
		if !listContains(sensorData.Path, addr) {
			url := fmt.Sprintf("http://%s/NotifyNeighbours", addr)
			fmt.Printf("\nContacting neighbour url: %s ", url)

			if !listContains(sensorData.Path, ou.Addr) {
				sensorData.Path = append(sensorData.Path, ou.Addr)
			}
			sensorData.Source = ou.Addr
			sensorData.Destination = addr

			message := "Can I get some data from you OUs.."
			addressBody := strings.NewReader(message)
			//fmt.Println("\nAddressbody: ", addressBody)

			_, err := http.Post(url, "string", addressBody)
			//errorMsg("Error posting to neighbour ", err)
			if err != nil {
				continue
			}
		}
	}
}


/*Tell Simulation that node is up and running*/
func (ou *ObservationUnit) tellSimulationUnit() {
	fmt.Printf("\nOU IS %s\n", ou.Addr)
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
	//fmt.Printf("\n### Cluster Head Election ###\n")
	//fmt.Printf("OU IS %s\n", ou.Addr)

	var pkt CHpkt

	if ou.Addr == "localhost:8084" && len(ou.ReachableNeighbours) != 0 {
		fmt.Printf("\n\nLOCALHOST:8084 IS CH!!!\n\n")
		ou.ClusterHeadCount += 1
		ou.ClusterHead = ou.Addr
		ou.IsClusterHead = true
		pkt.ClusterHead = ou.Addr
		//go ou.broadcastNewLeader(pkt)
		go ou.broadcastLeaderPath(pkt)
	}

	/*if len(ou.ReachableNeighbours) == 0 {
		fmt.Printf("No Reachable Neighbours.. Be your own CH!\n")
		ou.IsClusterHead = true
		ou.ClusterHead = ou.Addr
		ou.ClusterHeadCount += 1
	} else {
		//go ou.clusterHeadCalculation()
		go ou.findPathToLeader(pkt)
	}*/
	//go ou.clusterHeadCalculation()
}

func (ou ObservationUnit) broadcastLeaderPath(pkt CHpkt) {
	tickChan := time.NewTicker(time.Second * 10).C

	doneChan := make(chan bool)
    go func() {
        time.Sleep(time.Second * time.Duration(batteryStart))
        doneChan <- true
    }()
    
    for {
        select {
        case <- tickChan:
        	fmt.Printf("\n-------\n(%s) BROADCASTING NEW LEADER!!!!\n-------\n", ou.Addr)
        	go ou.broadcastNewLeader(pkt)

        case <- doneChan:
            return
      }
    }
}

func (ou *ObservationUnit) clusterHeadCalculation() {
	var pkt CHpkt

	//if battery us under 20% cannot OU be CH
	if float64(ou.BatteryTime) < (float64(batteryStart)*0.20) {
		fmt.Printf("OU cannot be CH because of low battery\n")
	} else {
		randNum := randomFloat()
		threshold := ou.threshold()

		if randNum < threshold {
		//if randNum > threshold {
			fmt.Printf("\n---------------------\nOU CAN BE CH!!!! BROADCAST TO NEIGHBOURS\n---------------------\n")
			ou.ClusterHeadCount += 1
			ou.ClusterHead = ou.Addr
			ou.IsClusterHead = true
			pkt.ClusterHead = ou.Addr
			go ou.broadcastNewLeader(pkt)
		} else {
			fmt.Printf("OU can not be CH because of threshold.. Wait for a path to leader..\n")
		}
		//time.Sleep(5 * time.Second)
	}
}

func (ou *ObservationUnit) getData(sensorData *SensorData) {
	tickChan := time.NewTicker(time.Second * 3).C

	doneChan := make(chan bool)
    go func() {
        time.Sleep(time.Second * time.Duration(batteryStart))
        doneChan <- true
    }()
    
    for {
        select {
        case <- tickChan:
        	go ou.NotifyNeighbours(sensorData)

        case <- doneChan:
            return
      }
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
	num := (rand.Float64() * 45) + 5
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


func (sd *SensorData) measureSensorData() {
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
        	//fmt.Println(start)
        	temp := temperatureSensor()
        	weather := weatherSensor()
        	//fmt.Println(temp, weather)

        	//Add values to sensorData
        	sd.Weather = append(sd.Weather, weather)
        	sd.Temperature = append(sd.Temperature, temp)
        	sd.DateTime = append(sd.DateTime, start)


        case <- doneChan:
            return
      }
    }
}


func weatherSensor() string {
	weather := make([]string, 0)
	weather = append(weather,
    "Sunny",
    "Cloudy",
    "Rain",
    "Windy",
    "Snow")

	rand.Seed(time.Now().UTC().UnixNano())
    rand_weather := weather[rand.Intn(len(weather))]
	//ou.Weather = append(ou.Weather, rand_weather)
	//fmt.Println("Weather: ", ou.Weather)
	return rand_weather
}


func temperatureSensor() int {
	rand_number := randomInt(-30, 20)
	//ou.Temperature = append(ou.Temperature, rand_number)
	//fmt.Println(rand_number)
	//fmt.Println("Temperature: ", ou.Temperature)
	return rand_number
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