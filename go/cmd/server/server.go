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

/*SimPort is port of simulator*/
var SimPort string

var reachableHosts []string

var batteryStart int64
var secondInterval int64

var wg sync.WaitGroup

var biggestAddress string

/*ObservationUnit is a struct of a OU*/
type ObservationUnit struct {
	Addr                string   `json:"Addr"`
	ID                  uint32   `json:"Id"`
	Pid                 int      `json:"Pid"`
	ReachableNeighbours []string `json:"-"`
	Neighbours          []string `json:"-"`
	BatteryTime         int64    `json:"-"`
	Xcor                float64  `json:"Xcor"`
	Ycor                float64  `json:"Ycor"`
	Sends               int      `json:"-"`
	SendsToLeader       int      `json:"-"`
	ClusterHeadCount    int      `json:"-"`
	Bandwidth           int      `json:"-"`
	CHpercentage        float64  `json:"-"`
	AccCount            int      `json:"-"`
	SensorData          `json:"-"`
	DataBaseStation     `json:"-"`
	LeaderElection      `json:"-"`
	LeaderCalculation   `json:"-"`
}

/*SensorData is data from "sensors" on the OU*/
type SensorData struct {
	ID          uint32
	Fingerprint uint32
	Data        []byte
	Source      string
	Destination string
	Accumulated bool
}

/*DataBaseStation is data sent to/gathered from the BS*/
type DataBaseStation struct {
	BSdatamap map[uint32][]byte
}

/*LeaderElection contains info about new leader election*/
type LeaderElection struct {
	ID         uint32
	Number     float64
	LeaderPath []string
	LeaderAddr string
	ViceLeader string
}

/*LeaderCalculation contains ID for new leader calculation*/
type LeaderCalculation struct {
	ID uint32
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

func addCommonFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&SimPort, "Simport", ":0", "Simulation (prefix with colon)")
	flagset.StringVar(&ouHost, "host", "localhost", "OU host")
	flagset.StringVar(&ouPort, "port", ":8081", "OU port (prefix with colon)")
}

func startServer() {
	/*1800 = 30 min, 3600 in 60 min*/
	batteryStart = 1800
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
		Sends:               0,
		SendsToLeader:       0,
		ClusterHeadCount:    0,
		Bandwidth:           bandwidth(),
		CHpercentage:        0,
		AccCount:            0,
		SensorData: SensorData{
			ID:          0,
			Fingerprint: 0,
			Data:        []byte{},
			Source:      "",
			Destination: "",
			Accumulated: false},
		DataBaseStation: DataBaseStation{
			BSdatamap: make(map[uint32][]byte)},
		LeaderElection: LeaderElection{
			ID:         hashAddress(hostaddress),
			Number:     0.0,
			LeaderPath: []string{},
			LeaderAddr: "",
			ViceLeader: ""},
		LeaderCalculation: LeaderCalculation{
			ID: 0}}

	//func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/shutdown", ou.shutdownHandler)
	http.HandleFunc("/reachableNeighbours", ou.reachableNeighboursHandler)
	http.HandleFunc("/clusterheadPercentage", ou.clusterheadPercentageHandler)
	http.HandleFunc("/newNeighbour", ou.newNeighboursHandler)
	http.HandleFunc("/noReachableNeighbours", ou.NoReachableNeighboursHandler)
	http.HandleFunc("/connectingOk", ou.connectingOkHandler)
	http.HandleFunc("/notifyNeighboursGetData", ou.notifyNeighboursGetDataHandler)
	http.HandleFunc("/sendDataToLeader", ou.sendDataToLeaderHandler)
	http.HandleFunc("/gossipLeaderElection", ou.gossipLeaderElectionHandler)
	http.HandleFunc("/gossipNewLeaderCalculation", ou.gossipNewLeaderCalculationHandler)

	//go ou.checkBatteryStatus()
	go ou.batteryConsumption()
	go ou.tellSimulationUnit()
	ou.LeaderElection.Number = randomFloat()
	//go ou.calculateLeaderThreshold()
	fmt.Printf("\n(%s): Random number %f\n", ou.Addr, ou.LeaderElection.Number)

	go ou.measureSensorData()

	//remove go
	go ou.getData()

	err := http.ListenAndServe(ouPort, nil)

	if err != nil {
		log.Panic(err)
	}
}

/*Get cluster percentage from Simulation.*/
func (ou *ObservationUnit) clusterheadPercentageHandler(w http.ResponseWriter, r *http.Request) {
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

/*NoReachableNeighboursHandler have no OUs in range of the OU..*/
func (ou *ObservationUnit) NoReachableNeighboursHandler(w http.ResponseWriter, r *http.Request) {
	var addrString string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &addrString)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	//go ou.calculateLeaderThreshold()
	//go ou.leaderElection(ou.LeaderElection)
}

/*Receive a new neighbour from OU that wants to connect to the cluster/a neighbour.*/
func (ou *ObservationUnit) newNeighboursHandler(w http.ResponseWriter, r *http.Request) {
	var newNeighbour string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &newNeighbour)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if !listContains(ou.ReachableNeighbours, newNeighbour) {
		ou.ReachableNeighbours = append(ou.ReachableNeighbours, newNeighbour)
	}

	time.Sleep(1000 * time.Millisecond)
	go ou.tellContactingOuOk(newNeighbour)
}

/*Receive a msg from leader about sending (accumulated) data to leader. */
func (ou *ObservationUnit) notifyNeighboursGetDataHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("\n\n------------------------------------------\n###(%s): NOTIFY NEIGHBOURS DATA HANDLER. ###\n------------------------------------------\n", ou.Addr)
	var sData SensorData

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &sData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	//fmt.Printf("\n(%s): Received get-data-msg from %s\n", ou.Addr, sData.Source)

	if sData.ID != ou.SensorData.ID {
		//fmt.Printf("\n(%s): Have not received this msg before. Need to forward msg to neighbours..\n", ou.Addr)
		ou.SensorData.ID = sData.ID

		go ou.notifyNeighboursGetData(sData)

		if ou.LeaderElection.LeaderPath[0] == ou.LeaderElection.LeaderAddr {
			fmt.Printf("(%s): First element is leader..Send data\n", ou.Addr)
			go ou.sendDataToLeader(sData)
		} else {
			//fmt.Printf("\n(%s) Accumulate data and send to leader (through path)\n", ou.Addr)
			//Need to accumulate data with neighbours
			go ou.accumulateSensorData(sData)
			//go ou.sendDataToLeader(sData)
		}
	} /*else {
		fmt.Printf("(%s): Data id and sensordata id similar..\n", ou.Addr)
	}*/
}

func (ou *ObservationUnit) gossipNewLeaderCalculationHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("(%s): Receive a new leader calculation handler\n", ou.Addr)
	var recLeaderCalc LeaderCalculation

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &recLeaderCalc); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	fmt.Printf("\n!!!!!!!!\n(%s): Random number is %f\n", ou.Addr, ou.LeaderElection.Number)
	if ou.LeaderCalculation.ID != recLeaderCalc.ID {
		fmt.Printf("(%s): Not received this msg before.. LeadID %d is not equal recID %d \nUpdate rand-num and ID.\n", ou.Addr, ou.LeaderCalculation.ID, recLeaderCalc.ID)
		ou.LeaderElection.Number = randomFloat()
		ou.LeaderCalculation.ID = recLeaderCalc.ID
		fmt.Printf("\n!!!!!!!!\n(%s): Random number is now %f\n", ou.Addr, ou.LeaderElection.Number)
		go ou.gossipNewLeaderCalculation()
	} else {
		fmt.Printf("(%s): Received this msg before..\n", ou.Addr)
		return
	}
}

func (ou *ObservationUnit) sendDataToLeaderHandler(w http.ResponseWriter, r *http.Request) {
	var sData SensorData
	var lock sync.RWMutex

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &sData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if ou.LeaderElection.LeaderAddr == ou.Addr {
		if len(ou.BSdatamap) == 0 {
			//fmt.Printf("(%s): Locked for writing to map..\n", ou.Addr)
			lock.Lock()
			defer lock.Unlock()
			ou.BSdatamap[sData.Fingerprint] = sData.Data
			//fmt.Printf("(%s): [0] Added data to BSdatamap\n", ou.Addr)
		} else {
			for key := range ou.BSdatamap {
				//fmt.Println("key:", key, "value:", []byte(value))
				if key == sData.Fingerprint {
					//fmt.Printf("(%s): [1] Key and Fingerprint is similar: %d\n", ou.Addr, key)
					//log.Printf("(%s):APPENDING %+v", ou.Addr, append(ou.BSdatamap[sData.Fingerprint][:], sData.Data[:]...))
				} else if key != sData.Fingerprint {
					lock.Lock()
					defer lock.Unlock()
					ou.BSdatamap[sData.Fingerprint] = sData.Data
					//fmt.Printf("(%s): [2]  Added data to BSdatamap\n\n", ou.Addr)
				}
			}
		}
		fmt.Printf("\n------------\n(%s): MAP IS: %+v\n------------\n\n", ou.Addr, ou.BSdatamap)
		fmt.Printf("(%s): Accumulated data from other nodes\n", ou.Addr)
	} else {
		//fmt.Printf("\n(%s): is not leader. Accumulate data and send to leader..\n", ou.Addr)
		go ou.accumulateSensorData(sData)
	}
}

/*Receive OK that new OU can join.*/
func (ou *ObservationUnit) connectingOkHandler(w http.ResponseWriter, r *http.Request) {
	var newNeighbour string

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &newNeighbour); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if !listContains(ou.ReachableNeighbours, newNeighbour) {
		ou.ReachableNeighbours = append(ou.ReachableNeighbours, newNeighbour)
	}
}

func (ou *ObservationUnit) gossipLeaderElectionHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("\n(%s) Gossip leader handler!\n", ou.Addr)
	var recLeaderData LeaderElection

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &recLeaderData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	//fmt.Printf("(%s): Gossip Leader Handler received: %+v\n\n", ou.Addr, recLeaderData)

	time.Sleep(time.Second * 2)
	go ou.leaderElection(recLeaderData)

}

/*IndexHandler doesn't do anything*/
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
func (ou *ObservationUnit) tellContactingOuOk(newNeighbour string) {

	url := fmt.Sprintf("http://%s/connectingOk", newNeighbour)
	//fmt.Printf("Sending to url: %s\n", url)

	b, err := json.Marshal(ou.Addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	addressBody := strings.NewReader(string(b))
	ou.Sends++
	_, err = http.Post(url, "string", addressBody)
	errorMsg("Error posting to neighbour about connection ok: ", err)
}

/*Contant neighbours in range with OUs address as body to tell that it wants to connect */
func (ou *ObservationUnit) contactNewNeighbour() {
	var i int
	for _, neighbour := range ou.ReachableNeighbours {
		i++
		url := fmt.Sprintf("http://%s/newNeighbour", neighbour)
		//fmt.Printf("\nContacting neighbour url: %s ", url)

		addressBody := strings.NewReader(ou.Addr)
		ou.Sends++
		_, err := http.Post(url, "string", addressBody)
		if err != nil {
			fmt.Printf("(%s) Try to post broadcast %s\n", ou.Addr, err)
			continue
		}
	}

	if i == len(ou.ReachableNeighbours) {
		time.Sleep(10 * time.Second)
		go ou.leaderElection(ou.LeaderElection)
	}
}

func (ou *ObservationUnit) gossipNewLeaderCalculation() {
	//fmt.Printf("(%s): Gossip new leader calculation..\n", ou.Addr)
	for _, addr := range ou.ReachableNeighbours {
		url := fmt.Sprintf("http://%s/gossipNewLeaderCalculation", addr)
		fmt.Printf("\n(%s): Contacting neighbour url: %s\n", ou.Addr, url)

		b, err := json.Marshal(ou.LeaderCalculation)
		if err != nil {
			fmt.Println(err)
			return
		}

		addressBody := strings.NewReader(string(b))
		ou.Sends++
		_, err = http.Post(url, "string", addressBody)
		//errorMsg("Error posting to neighbour ", err)
		if err != nil {
			fmt.Printf("(%s): Try to post notify neighbours to calculate new number %s\n", ou.Addr, err)
			continue
		}
	}
}

/*How to broadcast to neighbours with/without list of path... and how to receive??*/
func (ou *ObservationUnit) notifyNeighboursGetData(sensorData SensorData) {
	//fmt.Printf("\n(%s): Tell neighbours to send data to leader\n", ou.Addr)
	for _, addr := range ou.ReachableNeighbours {
		if addr != sensorData.Source {
			//if !listContains(sensorData.Path, addr) {
			url := fmt.Sprintf("http://%s/notifyNeighboursGetData", addr)
			//fmt.Printf("\n(%s): Contacting neighbour url: %s\n", ou.Addr, url)

			ou.SensorData.Source = ou.Addr
			ou.SensorData.Destination = addr
			//ou.SensorData.Data = []byte{}

			b, err := json.Marshal(ou.SensorData)
			if err != nil {
				fmt.Println(err)
				return
			}

			//fmt.Printf("(%s): Sending this get-data msg: %+v\n\n", ou.Addr, ou.SensorData)

			addressBody := strings.NewReader(string(b))
			//fmt.Println("\nAddressbody: ", addressBody)
			ou.Sends++
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
	//lastElem := ou.PathToCh[len(ou.PathToCh)-1]
	var lastElem string
	fmt.Printf("\n\n(%s): Leaderpath is : %+v\n", ou.Addr, ou.LeaderElection.LeaderPath)

	if len(ou.LeaderElection.LeaderPath) > 1 {
		lastElem = ou.LeaderElection.LeaderPath[len(ou.LeaderElection.LeaderPath)-1]
		if lastElem == ou.Addr {
			//removing last element
			//fmt.Printf("(%s): Remove last elem which is self\n", ou.Addr)
			ou.LeaderElection.LeaderPath = ou.LeaderElection.LeaderPath[:len(ou.LeaderElection.LeaderPath)-1]
			lastElem = ou.LeaderElection.LeaderPath[len(ou.LeaderElection.LeaderPath)-1]
		}
	} else {
		lastElem = ou.LeaderElection.LeaderPath[0]

		if lastElem == ou.Addr {
			fmt.Printf("(%s): Last element is ou-addr..\n", ou.Addr)
			return
		}
	}

	fmt.Printf("(%s): Last element is: %s\n", ou.Addr, lastElem)

	url := fmt.Sprintf("http://%s/sendDataToLeader", lastElem)
	fmt.Printf("\n(%s): Contacting neighbour url: %s\n", ou.Addr, url)

	sensorData.Source = ou.Addr
	sensorData.Destination = lastElem
	sensorData.Fingerprint = hashByte(ou.SensorData.Data)
	sensorData.Data = ou.SensorData.Data
	//sensorData.Accumulated = true

	//fmt.Printf("\n(%s): sending: %+v\n", ou.Addr, sensorData)

	b, err := json.Marshal(sensorData)
	if err != nil {
		fmt.Println(err)
		return
	}

	addressBody := strings.NewReader(string(b))
	//fmt.Println("\nAddressbody: ", addressBody)

	ou.Sends++
	ou.SendsToLeader++
	_, err = http.Post(url, "string", addressBody)
	//errorMsg("Error posting to neighbour ", err)
	if err != nil {
		fmt.Printf("(%s): Try to post new data to %s: %s\n", ou.Addr, lastElem, err)
	}

}

/*Tell Simulation that node is up and running*/
func (ou *ObservationUnit) tellSimulationUnit() {
	url := fmt.Sprintf("http://localhost:%s/notifySimulation", SimPort)
	//fmt.Printf("(%s): Sending to url: %s.\n", ou.Addr, url)

	b, err := json.Marshal(ou)
	if err != nil {
		fmt.Println(err)
		return
	}

	addressBody := strings.NewReader(string(b))
	ou.Sends++
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
	ou.Sends++
	_, err = http.Post(url, "string", addressBody)
	errorMsg("Post request dead OU: ", err)
}

func (ou *ObservationUnit) shutdownOu() {
	fmt.Printf("Low battery, shutting down..\n")
	ou.tellSimulationUnitDead()
	os.Exit(0)
}

func (ou *ObservationUnit) gossipLeaderElection() {
	//fmt.Printf("(%s): Gossip leader to neighbours if any..\n", ou.Addr)
	//time.Sleep(time.Second * 2)
	//fmt.Printf("(%s): Reachable Neighbours are: %+v\n", ou.Addr, ou.ReachableNeighbours)
	for _, addr := range ou.ReachableNeighbours {
		url := fmt.Sprintf("http://%s/gossipLeaderElection", addr)
		fmt.Printf("\n(%s): Gossip new leader to url: %s \n", ou.Addr, url)

		if !listContains(ou.LeaderElection.LeaderPath, ou.Addr) {
			ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
		}

		b, err := json.Marshal(ou.LeaderElection)
		if err != nil {
			fmt.Println(err)
			return
		}

		//fmt.Printf("(%s): Gossiping data: %+v\n\n", ou.Addr, ou.LeaderElection)
		addressBody := strings.NewReader(string(b))
		ou.Sends++
		_, err = http.Post(url, "string", addressBody)
		errorMsg("Post request broadcasting election of new leader failed: ", err)
		//io.Copy(os.Stdout, res.Body)
	}
}

func (ou *ObservationUnit) leaderElection(recLeaderData LeaderElection) {
	fmt.Printf("\n(%s): LEADER ELECTION!!\n", ou.Addr)

	//No leader
	if ou.LeaderElection.LeaderAddr == "" {
		//fmt.Printf("(%s): Have no leader\n", ou.Addr)

		//Update (randnum), id ,addr, path!!
		ou.LeaderElection.LeaderAddr = ou.Addr
		ou.LeaderElection.ID = ou.ID
		if !listContains(ou.LeaderElection.LeaderPath, ou.Addr) {
			ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
		}

		//fmt.Printf("\n\n--------------------\n(%s): OU-LEADERELECTION IS: %+v\n--------------------\n\n", ou.Addr, ou.LeaderElection)

		//ou.updateLeader(recLeaderData)
		go ou.gossipLeaderElection()
		return
	}
	fmt.Printf("(%s): Received data is: %+v\n\n", ou.Addr, recLeaderData)
	//If the one sending out the leader election is the one we have here
	if ou.LeaderElection.ID == recLeaderData.ID {
		fmt.Printf("(%s): OU ID (%d) and received ID (%d) are similar.\n", ou.Addr, ou.LeaderElection.ID, recLeaderData.ID)
		//If the random number generated is higher than we already registered
		if ou.LeaderElection.Number < recLeaderData.Number {
			fmt.Printf("(%s): ID similar:  This OU rand-number (%f) is not greater than received rand-number (%f)\n", ou.Addr, ou.LeaderElection.Number, recLeaderData.Number)
			//ou.updateLeader(recLeaderData)
			if len(ou.LeaderElection.LeaderPath) > len(recLeaderData.LeaderPath) {
				//Update path!
				fmt.Printf("(%s): Updating (%+v) to shorter path to leader\n", ou.Addr, ou.LeaderElection.LeaderPath)
				ou.LeaderElection.LeaderPath = recLeaderData.LeaderPath
				//fmt.Printf("(%s): Updated (%+v) to shorter path to leader\n", ou.Addr, ou.LeaderElection.LeaderPath)
			} else if len(ou.LeaderElection.LeaderPath) == len(recLeaderData.LeaderPath) {
				fmt.Printf("(%s): Path to leader: %+v is equal to: %+v\n", ou.Addr, ou.LeaderElection.LeaderPath, recLeaderData.LeaderPath)
				//if ou.LeaderElection.LeaderPath[0] == recLeaderData.LeaderAddr {
				ou.LeaderElection.LeaderPath = recLeaderData.LeaderPath
				//}
			}
			//Update randnum!!!
			//fmt.Printf("\n###################\n(%s): Update rand-num from %f to %f!!!\n###################\n", ou.Addr, ou.LeaderElection.Number, recLeaderData.Number)
			ou.LeaderElection.Number = recLeaderData.Number
			ou.LeaderElection.LeaderAddr = recLeaderData.LeaderAddr
			go ou.gossipLeaderElection()
		} else {
			//Random numbers are equal
			if ou.LeaderElection.Number == recLeaderData.Number {
				fmt.Printf("(%s): ID similar: This OU rand-number (%f) is equal to received rand-number (%f)\n", ou.Addr, ou.LeaderElection.Number, recLeaderData.Number)
				fmt.Printf("(%s): Don't do anyting except updating to shorter path to leader if possible..\n", ou.Addr)
				//should we switch to a shorter path to leader?
				if len(ou.LeaderElection.LeaderPath) > len(recLeaderData.LeaderPath) {
					if ou.LeaderElection.LeaderPath[0] == recLeaderData.LeaderPath[0] {
						fmt.Printf("(%s): Updating (%+v) to shorter path to leader %+v\n", ou.Addr, ou.LeaderElection.LeaderPath, recLeaderData.LeaderPath)
						ou.LeaderElection.LeaderPath = recLeaderData.LeaderPath
					} else {
						fmt.Printf("(%s): Paths leader[0] is not similar..\n", ou.Addr)
					}
					//fmt.Printf("(%s): Updating (%+v) to shorter path to leader\n", ou.Addr, ou.LeaderElection.LeaderPath)

					//ou.updateLeader(recLeaderData)
				} else if len(ou.LeaderElection.LeaderPath) == len(recLeaderData.LeaderPath) {
					fmt.Printf("(%s): Path (%+v) is equal to (%+v)..\n", ou.Addr, ou.LeaderElection.LeaderPath, recLeaderData.LeaderPath)
					if ou.LeaderElection.LeaderPath[0] == recLeaderData.LeaderPath[0] {
						fmt.Printf("(%s): EQUAL[0] %s and %s..\n", ou.Addr, ou.LeaderElection.LeaderPath[0], recLeaderData.LeaderPath[0])

					}
				}
			} else {
				fmt.Printf("\n(%s): ID similar: This OU rand-number (%f) is greater than received rand-number (%f)\n", ou.Addr, ou.LeaderElection.Number, recLeaderData.Number)
				if ou.Addr != ou.LeaderElection.LeaderAddr {
					ou.LeaderElection.LeaderPath = []string{}
					ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
					ou.LeaderElection.LeaderAddr = ou.Addr
				}

				//ou.LeaderElection = recLeaderData
				/*if len(ou.LeaderElection.LeaderPath) > len(recLeaderData.LeaderPath) {
					//Update path!!
					//fmt.Printf("(%s): Updating (%+v) to shorter path to leader\n", ou.Addr, ou.LeaderElection.LeaderPath)
					ou.LeaderElection.LeaderPath = recLeaderData.LeaderPath
					//fmt.Printf("(%s): Updated (%+v) to shorter path to leader\n", ou.Addr, ou.LeaderElection.LeaderPath)
				}*/
				/*if !listContains(ou.LeaderElection.LeaderPath, ou.Addr) {
					ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
				}*/
				//fmt.Printf("(%s):NEW LEADERELECTION IS: %+v \n\n\n", ou.Addr, ou.LeaderElection)
				go ou.gossipLeaderElection()
			}
		}
	} else {
		//Different leader
		fmt.Printf("(%s): OU ID (%d) and received ID (%d) are NOT similar. DIFFERENT LEADER.\n", ou.Addr, ou.LeaderElection.ID, recLeaderData.ID)
		//Received rand-num is higher than the one we have.. Update leader..
		if recLeaderData.Number > ou.LeaderElection.Number {
			//Update randnum, id, addr, path!!!
			fmt.Printf("(%s): Received rand-num (%f) is greater than what we have (%f)\n", ou.Addr, recLeaderData.Number, ou.LeaderElection.Number)
			ou.LeaderElection = recLeaderData
			if !listContains(ou.LeaderElection.LeaderPath, ou.Addr) {
				ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
				//fmt.Printf("(%s): Added self to leaderpath..\n", ou.Addr)
			}

			//fmt.Printf("(%s): Updated leader election with data from rec-election\n", ou.Addr)
			go ou.gossipLeaderElection()
		} else if recLeaderData.Number == ou.LeaderElection.Number {
			fmt.Printf("(%s): Received rand-num (%f) is equal to what we have (%f)\n", ou.Addr, recLeaderData.Number, ou.LeaderElection.Number)
			//Same random number, but different leader
			//Check if shorter path
			if len(ou.LeaderElection.LeaderPath) > len(recLeaderData.LeaderPath) {
				//Update id, addr, path!!!
				fmt.Printf("(%s): Received path to leader is shorter than what we have. Update leader\n", ou.Addr)
				ou.LeaderElection.ID = recLeaderData.ID
				ou.LeaderElection.LeaderAddr = recLeaderData.LeaderAddr
				ou.LeaderElection.LeaderPath = recLeaderData.LeaderPath
			} else {
				//Longer or equal path to leader so don't do anything..
				fmt.Printf("(%s): Received path to leader is longer than what we have. Don't update leader.\n", ou.Addr)
			}
		} else {
			//Different leader, lower rand-number..
			fmt.Printf("(%s): Received rand-num (%f) is lower than what we have (%f). Don't update LeaderElection..\n", ou.Addr, recLeaderData.Number, ou.LeaderElection.Number)
			//insert self to path..
			if !listContains(ou.LeaderElection.LeaderPath, ou.Addr) {
				//fmt.Printf("(%s): Insert self into (%+v)\n", ou.Addr, ou.LeaderElection.LeaderPath)
				ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
				//fmt.Printf("(%s): Inserted self into (%+v)\n\n", ou.Addr, ou.LeaderElection.LeaderPath)
			}

			go ou.gossipLeaderElection()
			return
		}
	}
	fmt.Printf("\n--------------------\n(%s): LEADERELECTION RESULT %+v\n--------------------\n", ou.Addr, ou.LeaderElection)
}

/*Chose if node is the biggest and become chief..*/
func (ou *ObservationUnit) biggestID() bool {
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
	}

	return false
}

func (ou *ObservationUnit) accumulateSensorData(sData SensorData) {
	var lock sync.RWMutex
	//fmt.Printf("\n(%s): Accumulate data with data from %s\n", ou.Addr, sData.Source)
	//log.Printf("(%s):sensordata was %+v -> APPENDING and now %+v", ou.Addr, ou.SensorData.Data, append(ou.SensorData.Data[:], sData.Data[:]...))
	//fmt.Printf("\n(%s):sensordata was %+v", ou.Addr, ou.SensorData.Data)
	lock.Lock()
	defer lock.Unlock()
	if ou.SensorData.Accumulated == false {
		//fmt.Printf("(%s): Accumulated data is false..\n", ou.Addr)
		ou.SensorData.Data = append(ou.SensorData.Data[:], sData.Data[:]...)
		ou.SensorData.Accumulated = true
	}
	//ou.SensorData.Data = append(ou.SensorData.Data[:], sData.Data[:]...)
	//ou.SensorData.Accumulated = true
	//fmt.Printf("\n(%s): Sensordata is now %+v\n", ou.Addr, ou.SensorData.Data)

	go ou.sendDataToLeader(ou.SensorData)
	//ou.SensorData.Data = append(ou.SensorData.Data, i)
}

func (ou *ObservationUnit) calculateLeaderThreshold() {
	time.Sleep(time.Second * 1)
	//have battery, bw, #sends(?) #communication with neighbours.., #neighbours, len(path to leader), Leaderpercentage
	fmt.Printf("(%s): Bandwidth: %d\n", ou.Addr, ou.Bandwidth)
	fmt.Printf("(%s): # neighbours: %d\n", ou.Addr, len(ou.ReachableNeighbours))
	fmt.Printf("(%s): Length of leaderpath: %d\n", ou.Addr, len(ou.LeaderElection.LeaderPath))
	fmt.Printf("(%s): Batterytime: %d\n", ou.Addr, ou.BatteryTime)
	fmt.Printf("(%s): # Sends: %d\n", ou.Addr, ou.Sends)
	fmt.Printf("(%s): # Sends to leader: %d\n", ou.Addr, ou.SendsToLeader)

	nodeCap := 100.0
	res := (float64(len(ou.ReachableNeighbours)) / nodeCap)
	res = toFixed(res, 3)
	fmt.Printf("(%s): Result is: %f\n", ou.Addr, res)

	time.Sleep(time.Second * 60)
}

func (ou *ObservationUnit) getData() {
	var num uint32
	var count uint32
	num = 0
	count = 0
	tickChan := time.NewTicker(time.Second * 180).C

	doneChan := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(batteryStart))
		doneChan <- true
	}()

	for {
		time.Sleep(time.Second * 50)
		if ou.LeaderElection.LeaderAddr == ou.Addr {
			select {
			case <-tickChan:
				//if ou.LeaderElection.LeaderAddr == ou.Addr {
				fmt.Printf("\n\n####################################\n(%s): IS LEADER.. SHOULD ASK FOR DATA\n####################################\n", ou.Addr)
				num++
				ou.SensorData.ID = ou.ID + num
				tmp := ou.SensorData.Data
				ou.SensorData.Data = []byte{}
				fmt.Printf("(%s): SENSORDATA ID: %d\n", ou.Addr, ou.SensorData.ID)
				go ou.notifyNeighboursGetData(ou.SensorData)
				ou.SensorData.Data = tmp
				ou.AccCount++
				time.Sleep(time.Second * 40)
				if ou.AccCount == 2 {
					//Accumulated data x times, elect a new leader..
					fmt.Printf("\n\n------------\n(%s): LEADER SENT DATA 2 TIMES!!!\n------------\n", ou.Addr)
					fmt.Printf("\n!!!!!!!!!!!\n(%s): Old random number is: %f\n", ou.Addr, ou.LeaderElection.Number)
					ou.LeaderElection.Number = randomFloat()
					fmt.Printf("(%s): New random number is: %f\n!!!!!!!!!!!\n", ou.Addr, ou.LeaderElection.Number)
					ou.LeaderCalculation.ID = hashAddress(ou.Addr)
					count++
					ou.LeaderCalculation.ID = ou.LeaderCalculation.ID + count
					ou.AccCount = 0
					go ou.gossipNewLeaderCalculation()
					time.Sleep(time.Second * 60)
					//go ou.leaderElection(ou.LeaderElection)
					go ou.gossipLeaderElection()

				}
				//}
			case <-doneChan:
				return
			}
		}
	}
}

/*randomFloat returns a random number in [0.0,1.0]*/
func randomFloat() float64 {
	rand.Seed(time.Now().UTC().UnixNano())
	num := rand.Float64()
	num = toFixed(num, 3)
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
	return bs
}

func estimateLocation() float64 {
	rand.Seed(time.Now().UTC().UnixNano())
	//num := (rand.Float64() * 495) + 5
	num := (rand.Float64() * 100) + 5
	//num := (rand.Float64() * 50) + 5
	//num := (rand.Float64() * 200) + 5
	return num
}

func (ou *ObservationUnit) checkBatteryStatus() {
	for {
		if float64(ou.BatteryTime) <= (float64(batteryStart) * 0.20) {
			fmt.Printf("\n\n(%s): BATTERY IS UNDER 20\n", ou.Addr)
			break
		}
	}

	if ou.LeaderElection.LeaderAddr == ou.Addr {
		fmt.Printf("\n(%s): have low battery: %d.. Need to chose a new leader\n", ou.Addr, ou.BatteryTime)
		//go ou.broadcastElectNewLeader()
	}
}

func (ou *ObservationUnit) batteryConsumption() {
	tickChan := time.NewTicker(time.Second * 1).C

	doneChan := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(batteryStart))
		doneChan <- true
	}()

	for {
		select {
		case <-tickChan:
			if ou.LeaderElection.LeaderAddr == ou.Addr {
				ou.BatteryTime -= (secondInterval * 2)
			} else {
				ou.BatteryTime -= secondInterval
			}
			/*fmt.Printf("(%s): batterytime: %d\n", ou.Addr, ou.BatteryTime)*/
		case <-doneChan:
			ou.BatteryTime = 0
			fmt.Println("Done. OU is dead..")
			go ou.shutdownOu()
			return
		}
	}
}

func setMaxProcs() int {
	//GOMAXPROCS sets the maximum number of CPUs that can be executing simultaneously and returns the previous setting. If n < 1, it does not change the current setting. The number of logical CPUs on the local machine can be queried with NumCPU.
	maxProcs := runtime.GOMAXPROCS(0)
	//NumCPU returns the number of logical CPUs usable by the current process.
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

/*Round float number to x decimals */
func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

func randomInt64(max int64) int64 {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Int63n(max)
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
		case <-tickChan:
			go ou.byteSensor()
		case <-doneChan:
			return
		}
	}
}

func (ou *ObservationUnit) byteSensor() {
	rand.Seed(time.Now().UTC().UnixNano())
	ou.SensorData.Data = make([]byte, 4)
	ou.SensorData.Accumulated = false
	rand.Read(ou.SensorData.Data)
}

/*Return a random int describing which bandwidth-type for specific OU.
Use this value to determine if OU can be leader*/
func bandwidth() int {
	bw := make(map[string]int)
	bw["LoRa"] = 50
	bw["Cable"] = 90
	bw["Wifi"] = 30

	rand.Seed(time.Now().UTC().UnixNano())
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
