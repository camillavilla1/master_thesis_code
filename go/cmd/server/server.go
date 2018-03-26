package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
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
	Addr                string   `json:"Addr"`
	ID                  uint32   `json:"Id"`
	Pid                 int      `json:"Pid"`
	ReachableNeighbours []string `json:"-"`
	Neighbours          []string `json:"-"`
	BatteryTime         int64    `json:"-"`
	Xcor                float64  `json:"Xcor"`
	Ycor                float64  `json:"Ycor"`
	ClusterHead         string   `json:"-"`
	IsClusterHead       bool     `json:"-"`
	ClusterHeadCount    int      `json:"-"`
	Bandwidth           int      `json:"-"`
	PathToCh            []string `json:"-"`
	CHpercentage        float64  `json:"-"`
	SensorData          `json:"-"`
}

type CHpkt struct {
	Path        []string
	Source      string
	Destination string
	ClusterHead string
}

type SensorData struct {
	ID          uint32
	Fingerprint uint32
	Data        []byte
	Source      string
	Destination string
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
	batteryStart = 1000
	secondInterval = 1
	hostaddress := ouHost + ouPort

	log.Printf("Starting Observation Unit on %s\n", hostaddress)

	ou := &ObservationUnit{
		Addr:                hostaddress,
		ID:                  hashAddress(hostaddress),
		Pid:                 os.Getpid(),
		BatteryTime:         batteryStart,
		ReachableNeighbours: []string{},
		Neighbours:          []string{},
		Xcor:                estimateLocation(),
		Ycor:                estimateLocation(),
		ClusterHead:         "",
		IsClusterHead:       false,
		ClusterHeadCount:    0,
		Bandwidth:           bandwidth(),
		PathToCh:            []string{},
		CHpercentage:        0,
		SensorData: SensorData{
			ID:          0,
			Fingerprint: 0,
			Data:        []byte{},
			Source:      "",
			Destination: ""}}

	//func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/shutdown", ou.shutdownHandler)
	http.HandleFunc("/reachableNeighbours", ou.reachableNeighboursHandler)
	http.HandleFunc("/clusterheadPercentage", ou.clusterheadPercentageHandler)
	http.HandleFunc("/newNeighbour", ou.newNeighboursHandler)
	http.HandleFunc("/noReachableNeighbours", ou.NoReachableNeighboursHandler)
	http.HandleFunc("/connectingOk", ou.connectingOkHandler)
	http.HandleFunc("/broadcastNewLeader", ou.broadcastNewLeaderHandler)
	http.HandleFunc("/notifyNeighboursGetData", ou.notifyNeighboursGetDataHandler)
	http.HandleFunc("/sendDataToLeader", ou.sendDataToLeaderHandler)

	go ou.batteryConsumption()
	go ou.tellSimulationUnit()
	go ou.measureSensorData()
	go ou.getData()

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
	//fmt.Printf("\n### ReachableNeighbours Handler ###\n")
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
	//fmt.Printf("\n### OU received no neighbour (Handler) ###\n")
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
	fmt.Printf("\n\n------------------------------------------\n###(%s): BROADCAST HANDLER.. Received a new leader. ###\n------------------------------------------\n", ou.Addr)
	var pkt CHpkt

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &pkt); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	fmt.Printf("(%s): Received packet from %s\n", ou.Addr, pkt.Source)

	if ou.ClusterHead == pkt.ClusterHead {
		//fmt.Printf("\n(%s) clusterhead: %s and pkt-ch %s is similar.. No need to do anything..\n", ou.Addr, ou.ClusterHead, pkt.ClusterHead)
	} else {
		//fmt.Printf("\n(%s) CH is %s, changes to pkt-ch: %s\n", ou.Addr, ou.ClusterHead, pkt.ClusterHead)
		ou.ClusterHead = pkt.ClusterHead
	}

	if len(ou.PathToCh) == 0 {
		ou.PathToCh = pkt.Path
	} else if len(ou.PathToCh) >= len(pkt.Path) {
		ou.PathToCh = pkt.Path
	}
	//fmt.Printf("\n######################\n(%s) path to CH is: %v\n######################\n", ou.Addr, ou.PathToCh)

	for _, addr := range ou.Neighbours {
		if addr != ou.ClusterHead && addr != pkt.Source {
			//fmt.Printf("\n(%s) have neighbours. Send to %s\n", ou.Addr, addr)
			//fmt.Printf("%s is not ch (%s)\n", addr, ou.ClusterHead)
			//fmt.Printf("(%s) appending own addr to path\n", ou.Addr)
			if !listContains(pkt.Path, ou.Addr) {
				pkt.Path = append(pkt.Path, ou.Addr)
			}

			go ou.broadcastNewLeader(pkt)

		} else {
			//fmt.Printf("\n(%s): %s is clusterhead or/and pkt-source %s. No need to send update\n", ou.Addr, addr, pkt.Source)
			continue
		}
	}
}

/*Receive a msg from CH about sending (accumulated) data to CH. */
func (ou *ObservationUnit) notifyNeighboursGetDataHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n\n------------------------------------------\n###(%s): NOTIFY NEIGHBOURS DATA HANDLER. ###\n------------------------------------------\n", ou.Addr)
	var sData SensorData

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &sData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	fmt.Printf("\n(%s): Received msg from %s\n", ou.Addr, sData.Source)

	if sData.ID == ou.SensorData.ID {
		fmt.Printf("\n(%s): Have received this msg before.. \n", ou.Addr)
	} else {
		fmt.Printf("\n(%s): Have not received this msg before. Need to forward msg to neighbours..\n", ou.Addr)
		go ou.notifyNeighboursGetData(sData)

		//go ou.sendDataToLeader(sData)

		//if ch is neighbour, no need for accumulate data..
		if len(ou.PathToCh) <= 1 {
			if ou.PathToCh[0] == ou.ClusterHead {
				fmt.Printf("\n(%s) Send data to CH \n", ou.Addr)
				sData.Source = ou.Addr
				sData.Fingerprint = hashByte(ou.SensorData.Data)
				go ou.sendDataToLeader(sData)
			}
		} else {
			fmt.Printf("\n(%s) Accumulate data and send to CH (through path)\n", ou.Addr)
			//Need to accumulate data with neighbours
			go ou.accumulateSensorData(sData)
		}
	}
}

func (ou *ObservationUnit) sendDataToLeaderHandler(w http.ResponseWriter, r *http.Request) {
	var sData SensorData

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &sData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if ou.IsClusterHead == true {
		fmt.Printf("\n(%s): Received data from %s. Need to accumulate that data\n", ou.Addr, sData.Source)
		fmt.Printf("\n(%s): sensordata looks like this: %x, with fingerprint: %b", ou.Addr, sData.Data, sData.Fingerprint)
	}

}

/*Receive ok from CH that new OU can join.*/
func (ou *ObservationUnit) connectingOkHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("\n###(%s):  Received OK from neighbour. Connect OU to new neighbour!\n", ou.Addr)

	var data []string

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &data); err != nil {
		panic(err)
	}

	neighOu := strings.Join(data[:1], "") //first element
	//fmt.Println(url2)
	//newOu := strings.Join(data[1:2],"") //middle element, nr 2
	clusterHead := strings.Join(data[2:], "")

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
	newOu := strings.Join(data[1:2], "") //middle element, nr 2
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
	//fmt.Printf("\n###(%s):  Broadcast new leader to neighbours ###\n", ou.Addr)
	//var pkt CHpkt
	if !listContains(pkt.Path, ou.Addr) {
		//fmt.Printf("\n(%s): Appending (%s) to the pkt-path\n", ou.Addr, ou.Addr)
		pkt.Path = append(pkt.Path, ou.Addr)
	}

	//fmt.Printf("(%s) neighbours: %v", ou.Addr, ou.Neighbours)
	for _, addr := range ou.Neighbours {
		if !listContains(pkt.Path, addr) {
			//fmt.Printf("\n(%s): Contact %s because it's not in pkt-path %v..\n", ou.Addr, addr, pkt.Path)
			url := fmt.Sprintf("http://%s/broadcastNewLeader", addr)
			fmt.Printf("\n(%s): Contacting neighbour url: %s ", ou.Addr, url)

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
				fmt.Printf("(%s): Try to post broadcast %s\n", ou.Addr, err)
				continue
			}
		} else {
			//fmt.Printf("\n(%s): Address %s is already in pkt-path. Do not need to contact..\n", ou.Addr, addr)
			continue
		}
	}

}

/*How to broadcast to neighbours with/without list of path... and how to receive??*/
func (ou *ObservationUnit) notifyNeighboursGetData(sensorData SensorData) {

	for _, addr := range ou.Neighbours {

		if addr != sensorData.Source {
			//if !listContains(sensorData.Path, addr) {
			url := fmt.Sprintf("http://%s/notifyNeighboursGetData", addr)
			fmt.Printf("\n(%s): Contacting neighbour url: %s ", ou.Addr, url)

			/*	if !listContains(sensorData.Path, ou.Addr) {
				sensorData.Path = append(sensorData.Path, ou.Addr)
			}*/
			ou.SensorData.Source = ou.Addr
			ou.SensorData.Destination = addr

			b, err := json.Marshal(ou.SensorData)
			if err != nil {
				fmt.Println(err)
				return
			}

			addressBody := strings.NewReader(string(b))
			//fmt.Println("\nAddressbody: ", addressBody)

			_, err = http.Post(url, "string", addressBody)
			//errorMsg("Error posting to neighbour ", err)
			if err != nil {
				fmt.Printf("(%s): Try to post notify neighbours to get data %s\n", ou.Addr, err)
				continue
			}
			//}
		}

	}
}

func (ou *ObservationUnit) sendDataToLeader(sensorData SensorData) {
	lastElem := ou.PathToCh[len(ou.PathToCh)-1]

	//removing last element
	//newPath := ou.PathToCh[:len(ou.PathToCh)-1]
	//fmt.Printf("\n(%s): New path looks like: %s\n", ou.Addr, newPath)

	url := fmt.Sprintf("http://%s/sendDataToLeader", lastElem)
	fmt.Printf("\n(%s): Contacting neighbour url: %s ", ou.Addr, url)

	sensorData.Source = ou.Addr

	b, err := json.Marshal(sensorData)
	if err != nil {
		fmt.Println(err)
		return
	}

	addressBody := strings.NewReader(string(b))
	//fmt.Println("\nAddressbody: ", addressBody)

	_, err = http.Post(url, "string", addressBody)
	//errorMsg("Error posting to neighbour ", err)
	if err != nil {
		fmt.Printf("(%s): Try to post new data to %s: %s\n", ou.Addr, lastElem, err)
	}

}

/*Tell Simulation that node is up and running*/
func (ou *ObservationUnit) tellSimulationUnit() {
	//fmt.Printf("\nOU IS %s\n", ou.Addr)
	//fmt.Println("Battery is ", ou.BatteryTime)

	url := fmt.Sprintf("http://localhost:%s/notifySimulation", SimPort)
	fmt.Printf("(%s): Sending to url: %s.\n", ou.Addr, url)

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
	fmt.Printf("(%s): Sending 'I'm dead..' to url: %s \n", ou.Addr, url)

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

	for i, neighbour := range ou.ReachableNeighbours {
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

	if ou.Addr == "localhost:8085" && len(ou.ReachableNeighbours) != 0 {
		fmt.Printf("\n\nLOCALHOST:8085 IS CH!!!\n\n")
		ou.ClusterHeadCount += 1
		ou.ClusterHead = ou.Addr
		ou.IsClusterHead = true
		pkt.ClusterHead = ou.Addr
		go ou.broadcastNewLeader(pkt)
		//go ou.broadcastLeaderPath(pkt)
	}

	//go ou.clusterHeadCalculation()
}

func (ou ObservationUnit) broadcastLeaderPath(pkt CHpkt) {
	tickChan := time.NewTicker(time.Second * 20).C

	doneChan := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(batteryStart))
		doneChan <- true
	}()

	for {
		select {
		case <-tickChan:
			go ou.broadcastNewLeader(pkt)

		case <-doneChan:
			return
		}
	}
}

func (ou *ObservationUnit) accumulateSensorData(sData SensorData) {
	fmt.Printf("\n(%s): Accumulate data with data from %s\n", ou.Addr, sData.Source)

}

func (ou *ObservationUnit) clusterHeadCalculation() {
	var pkt CHpkt

	//if battery is under 20% cannot OU be CH
	if float64(ou.BatteryTime) < (float64(batteryStart) * 0.20) {
		fmt.Printf("(%s): cannot be CH because of low battery\n", ou.Addr)
	} else {
		randNum := randomFloat()
		threshold := ou.threshold()

		if randNum < threshold {
			//if randNum > threshold {
			fmt.Printf("\n---------------------\n(%s): CAN BE CH!!!! BROADCAST NEW LEADER TO NEIGHBOURS\n---------------------\n", ou.Addr)
			ou.ClusterHeadCount += 1
			ou.ClusterHead = ou.Addr
			ou.IsClusterHead = true
			pkt.ClusterHead = ou.Addr
			go ou.broadcastLeaderPath(pkt)
			//go ou.broadcastNewLeader(pkt)
		} /* else {
			fmt.Printf("\n(%s): can not be CH because of threshold.. Wait for a path to leader..\n", ou.Addr)
		}*/
		//time.Sleep(5 * time.Second)
	}
}

func (ou *ObservationUnit) getData() {
	var num uint32
	num = 0
	tickChan := time.NewTicker(time.Second * 60).C

	doneChan := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(batteryStart))
		doneChan <- true
	}()

	for {
		select {
		case <-tickChan:
			if ou.IsClusterHead == true {
				fmt.Printf("\n(%s): IS CLUSTER HEAD.. SHOULD ASK FOR DATA\n", ou.Addr)
				num += 1
				ou.SensorData.ID = ou.ID + num
				fmt.Printf("(%s): SENSORDATA ID: %d\n", ou.Addr, ou.SensorData.ID)
				go ou.notifyNeighboursGetData(ou.SensorData)
			}

		case <-doneChan:
			return
		}
	}
}

func randomFloat() float64 {
	num := rand.Float64()
	return num
}

func (ou ObservationUnit) threshold() float64 {
	threshold := ou.CHpercentage/1 - (ou.CHpercentage * (math.Mod(float64(ou.ClusterHeadCount), 1/float64(ou.ClusterHeadCount))))
	return threshold
}

/*Calculate battery percentage on OU*/
func calcPercentage(batteryTime int64, maxBattery int64) float64 {
	ret := (float64(batteryTime) / float64(maxBattery)) * 100
	return ret
}

//Hash address to be ID of node
func hashAddress(address string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(address))
	hashedAddress := h.Sum32()
	return hashedAddress
}

func hashByte(data []byte) uint32 {
	h := fnv.New32a()
	h.Write([]byte(data))
	bs := h.Sum32()
	fmt.Printf("%x\n", bs)
	return bs
}

func estimateLocation() float64 {
	rand.Seed(time.Now().UTC().UnixNano())
	//num := (rand.Float64() * 495) + 5
	//num := (rand.Float64() * 145) + 5
	num := (rand.Float64() * 75) + 5
	return num
}

func (ou *ObservationUnit) batteryConsumption() {
	tickChan := time.NewTicker(time.Millisecond * 1000).C

	doneChan := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(batteryStart))
		doneChan <- true
	}()

	for {
		select {
		case <-tickChan:
			ou.BatteryTime -= secondInterval

			if float64(ou.BatteryTime) <= (float64(batteryStart) * 0.20) {
				fmt.Printf("\n(%s): have low battery.. Need to sleep to save power and chose a new CH\n", ou.Addr)
			}
		case <-doneChan:
			ou.BatteryTime = 0
			fmt.Println("Done. OU is dead..")
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
	return rand.Intn(max-min) + min
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

func (ou *ObservationUnit) measureSensorData() {
	tickChan := time.NewTicker(time.Second * 10).C

	doneChan := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(batteryStart))
		doneChan <- true
	}()

	for {
		select {
		//case <- timeChan:
		//    fmt.Println("Timer expired.\n")
		case <-tickChan:
			//start := time.Now().Format("2006-01-02 15:04:05") //.Format(time.RFC850)
			//fmt.Println(start)
			/*temp := temperatureSensor()
			weather := weatherSensor()
			//fmt.Println(temp, weather)

			//Add values to sensorData
			sd.Weather = append(sd.Weather, weather)
			sd.Temperature = append(sd.Temperature, temp)
			sd.DateTime = append(sd.DateTime, start)
			*/
			go ou.byteSensor()
			//fmt.Printf("\n(%s): data is: %b\n", ou.Addr, ou.SensorData.Data)

		case <-doneChan:
			return
		}
	}
}

func (ou *ObservationUnit) byteSensor() {
	rand.Seed(time.Now().UTC().UnixNano())
	token := make([]byte, 4)
	rand.Read(token)
	fmt.Println(token)
	num := randomInt(1, 5000)
	ou.SensorData.Data = append(ou.SensorData.Data, byte(num))
}

/*func weatherSensor() string {
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
}*/

/*func temperatureSensor() int {
	rand_number := randomInt(-30, 20)
	//ou.Temperature = append(ou.Temperature, rand_number)
	//fmt.Println(rand_number)
	//fmt.Println("Temperature: ", ou.Temperature)
	return rand_number
}*/

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

	return bw[k]

}
