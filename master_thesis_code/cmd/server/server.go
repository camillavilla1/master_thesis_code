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
	_ "net/http/pprof"
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
	ReceivedDataPkt     int      `json:"-"`
	ClusterHeadCount    int      `json:"-"`
	Bandwidth           int      `json:"-"`
	CHpercentage        float64  `json:"-"`
	AccCount            int      `json:"-"`
	SensorData          `json:"-"`
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
var DataBaseStation struct {
	sync.Mutex
	BSdatamap map[uint32][]byte
}

/*LeaderElection contains info about new leader election*/
type LeaderElection struct {
	ID            uint32
	Number        float64
	LeaderPath    []string
	LeaderAddr    string
	ViceLeader    string
	TmpLeaderPath []string
	TmpLeaderAddr string
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
		//ret := setMaxProcs()
		//setMaxProcs()
		//fmt.Printf("\n\n----------------------------------------------------------------------------\n")
		//fmt.Println("Processes:", ret)
		//time.Sleep(time.Second * 1)
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
	/*ca 225 = 3.5 min, 450 = 7.5 min, 900 = 15 min*/
	batteryStart = 900
	secondInterval = 1
	hostaddress := ouHost + ouPort
	DataBaseStation.BSdatamap = make(map[uint32][]byte)

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
		ReceivedDataPkt:     0,
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
		LeaderElection: LeaderElection{
			ID:            hashAddress(hostaddress),
			Number:        0.0,
			LeaderPath:    []string{},
			LeaderAddr:    "",
			ViceLeader:    "",
			TmpLeaderPath: []string{},
			TmpLeaderAddr: ""},
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
	go ou.Experiments(os.Getpid())
	//go ou.Experiments2(os.Getpid())
	//go ReadCsv()
	//go ou.ConvertExperiments()

	go ou.batteryConsumption()
	go ou.tellSimulationUnit()
	ou.LeaderElection.Number = randomFloat()

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
	ErrorMsg("readall: ", err)

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
	ErrorMsg("readall: ", err)

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

	if !ListContains(ou.ReachableNeighbours, newNeighbour) {
		ou.ReachableNeighbours = append(ou.ReachableNeighbours, newNeighbour)
	}

	time.Sleep(1000 * time.Millisecond)
	go ou.tellContactingOuOk(newNeighbour)
}

/*Receive a msg from leader about sending (accumulated) data to leader. */
func (ou *ObservationUnit) notifyNeighboursGetDataHandler(w http.ResponseWriter, r *http.Request) {
	var sData SensorData

	body, err := ioutil.ReadAll(r.Body)
	ErrorMsg("readall: ", err)

	if err := json.Unmarshal(body, &sData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if sData.ID != ou.SensorData.ID {
		//fmt.Printf("\n(%s): Have not received this msg before. Need to forward msg to neighbours..\n", ou.Addr)
		ou.SensorData.ID = sData.ID

		go ou.notifyNeighboursGetData(sData)

		if len(ou.LeaderElection.LeaderPath) == 0 {
			return
		} else {
			go ou.sendDataToLeader(sData)
		}

		/*if ou.LeaderElection.LeaderPath[0] == ou.LeaderElection.LeaderAddr {
			fmt.Printf("(%s): FIRST ELEMENT IN PATH IS LEADER. SEND DATA TO LEADER\n", ou.Addr)
			go ou.sendDataToLeader(sData)
		} else {
			//Need to accumulate data with neighbours
			fmt.Printf("(%s): FIRST ELEMENT IS NOT LEADER. ACCUMULATE DATA\n", ou.Addr)
			go ou.accumulateSensorData(sData)
		}*/
	}
}

/*gossipNewLeaderCalculationHandler calculates a new score for leader election*/
func (ou *ObservationUnit) gossipNewLeaderCalculationHandler(w http.ResponseWriter, r *http.Request) {
	var recLeaderCalc LeaderCalculation

	body, err := ioutil.ReadAll(r.Body)
	ErrorMsg("readall: ", err)

	if err := json.Unmarshal(body, &recLeaderCalc); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if ou.LeaderCalculation.ID != recLeaderCalc.ID {
		ou.LeaderElection.Number = randomFloat()
		ou.LeaderCalculation.ID = recLeaderCalc.ID
		ou.LeaderElection.TmpLeaderPath = ou.LeaderElection.LeaderPath
		ou.LeaderElection.LeaderPath = []string{}
		ou.LeaderElection.TmpLeaderAddr = ou.LeaderElection.LeaderAddr
		ou.LeaderElection.LeaderAddr = ""

		go ou.gossipNewLeaderCalculation()
	} else {
		return
	}
}

/*sendDataToLeaderHandler handles/accumulates data as ch or as regular node */
func (ou *ObservationUnit) sendDataToLeaderHandler(w http.ResponseWriter, r *http.Request) {
	var sData SensorData

	DataBaseStation.Lock()
	defer DataBaseStation.Unlock()

	body, err := ioutil.ReadAll(r.Body)
	ErrorMsg("readall: ", err)

	if err := json.Unmarshal(body, &sData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if ou.LeaderElection.LeaderAddr == ou.Addr {
		ou.ReceivedDataPkt++
		if len(DataBaseStation.BSdatamap) == 0 {
			DataBaseStation.BSdatamap[sData.Fingerprint] = sData.Data
		} else {
			for key := range DataBaseStation.BSdatamap {
				if key != sData.Fingerprint {
					DataBaseStation.BSdatamap[sData.Fingerprint] = sData.Data
				}
			}
		}
		fmt.Printf("(%s): Accumulated data from other nodes\n", ou.Addr)
	} else {
		go ou.accumulateSensorData(sData)
	}
}

/*connectingOkHandler receive OK that new OU can join.*/
func (ou *ObservationUnit) connectingOkHandler(w http.ResponseWriter, r *http.Request) {
	var newNeighbour string

	body, err := ioutil.ReadAll(r.Body)
	ErrorMsg("readall: ", err)

	if err := json.Unmarshal(body, &newNeighbour); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if !ListContains(ou.ReachableNeighbours, newNeighbour) {
		ou.ReachableNeighbours = append(ou.ReachableNeighbours, newNeighbour)
	}
}

/*gossipLeaderElectionHandler receives a new leader election and starts a new leader election*/
func (ou *ObservationUnit) gossipLeaderElectionHandler(w http.ResponseWriter, r *http.Request) {
	var recLeaderData LeaderElection

	body, err := ioutil.ReadAll(r.Body)
	ErrorMsg("readall: ", err)

	if err := json.Unmarshal(body, &recLeaderData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	time.Sleep(time.Millisecond * 2000)
	go ou.leaderElection(recLeaderData)

}

/*IndexHandler doesn't do anything*/
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Don't use the request body. Consume it anyway.
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

/*tellContactingOuOk tell contaction OU that it is ok to join the cluster*/
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
	ErrorMsg("Error posting to neighbour about connection ok: ", err)
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
		time.Sleep(time.Millisecond * 8000)
		go ou.leaderElection(ou.LeaderElection)
	}
}

/*gossipNewLeaderCalculation gossip new leader calculation to its neighours*/
func (ou *ObservationUnit) gossipNewLeaderCalculation() {
	for _, addr := range ou.ReachableNeighbours {
		url := fmt.Sprintf("http://%s/gossipNewLeaderCalculation", addr)
		//fmt.Printf("\n(%s): Contacting neighbour url: %s\n", ou.Addr, url)

		b, err := json.Marshal(ou.LeaderCalculation)
		if err != nil {
			fmt.Println(err)
			return
		}

		addressBody := strings.NewReader(string(b))
		ou.Sends++
		_, err = http.Post(url, "string", addressBody)
		if err != nil {
			fmt.Printf("(%s): Try to post notify neighbours to calculate new number %s\n", ou.Addr, err)
			continue
		}
	}
}

/*notifyNeighboursGetData notify neighbours to send data to leader*/
func (ou *ObservationUnit) notifyNeighboursGetData(sensorData SensorData) {
	for _, addr := range ou.ReachableNeighbours {
		if addr != sensorData.Source {
			url := fmt.Sprintf("http://%s/notifyNeighboursGetData", addr)
			//fmt.Printf("\n(%s): Contacting neighbour url: %s\n", ou.Addr, url)

			ou.SensorData.Source = ou.Addr
			ou.SensorData.Destination = addr

			b, err := json.Marshal(ou.SensorData)
			if err != nil {
				fmt.Println(err)
				return
			}

			addressBody := strings.NewReader(string(b))
			ou.Sends++

			_, err = http.Post(url, "string", addressBody)
			if err != nil {
				fmt.Printf("(%s): Try to post notify neighbours to get data %s\n", ou.Addr, err)
				continue
			}
		}

	}
}

/*sendDataToLeader sends data to the next neighbour in the path to leader*/
func (ou *ObservationUnit) sendDataToLeader(sensorData SensorData) {
	var lastElem string

	if len(ou.LeaderElection.LeaderPath) > 1 {
		lastElem = ou.LeaderElection.LeaderPath[len(ou.LeaderElection.LeaderPath)-1]
		if lastElem == ou.Addr {
			//removing last element (self)
			ou.LeaderElection.LeaderPath = ou.LeaderElection.LeaderPath[:len(ou.LeaderElection.LeaderPath)-1]
			lastElem = ou.LeaderElection.LeaderPath[len(ou.LeaderElection.LeaderPath)-1]
		}
	} else {
		lastElem = ou.LeaderElection.LeaderPath[0]
		if lastElem == ou.Addr {
			return
		}
	}

	url := fmt.Sprintf("http://%s/sendDataToLeader", lastElem)
	//fmt.Printf("\n(%s): Contacting neighbour url: %s\n", ou.Addr, url)

	sensorData.Source = ou.Addr
	sensorData.Destination = lastElem
	sensorData.Fingerprint = hashByte(ou.SensorData.Data)
	sensorData.Data = ou.SensorData.Data

	b, err := json.Marshal(sensorData)
	if err != nil {
		fmt.Println(err)
		return
	}

	addressBody := strings.NewReader(string(b))

	ou.Sends++
	ou.SendsToLeader++
	_, err = http.Post(url, "string", addressBody)
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
	ErrorMsg("POST request to Simulation failed: ", err)
	io.Copy(os.Stdout, res.Body)
}

/*Tell SOU that you're dead */
func (ou *ObservationUnit) tellSimulationUnitDead() {
	url := fmt.Sprintf("http://localhost:%s/removeReachableOu", SimPort)
	//fmt.Printf("(%s): Sending 'I'm dead..' to url: %s \n", ou.Addr, url)

	b, err := json.Marshal(ou)
	if err != nil {
		fmt.Println(err)
		return
	}

	addressBody := strings.NewReader(string(b))
	ou.Sends++
	_, err = http.Post(url, "string", addressBody)
	ErrorMsg("Post request dead OU: ", err)
}

func (ou *ObservationUnit) shutdownOu() {
	fmt.Printf("Low battery, shutting down..\n")
	ou.tellSimulationUnitDead()
	os.Exit(0)
}

/*gossipLeaderElection gossip leader election to neighbours*/
func (ou *ObservationUnit) gossipLeaderElection() {
	for _, addr := range ou.ReachableNeighbours {
		url := fmt.Sprintf("http://%s/gossipLeaderElection", addr)
		//fmt.Printf("\n(%s): Gossip new leader to url: %s \n", ou.Addr, url)

		if !ListContains(ou.LeaderElection.LeaderPath, ou.Addr) {
			ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
		}

		b, err := json.Marshal(ou.LeaderElection)
		if err != nil {
			fmt.Println(err)
			return
		}

		addressBody := strings.NewReader(string(b))
		ou.Sends++
		_, err = http.Post(url, "string", addressBody)
		ErrorMsg("Post request broadcasting election of new leader failed: ", err)
	}
}

/*leaderElection to find out which node should be leader*/
func (ou *ObservationUnit) leaderElection(recLeaderData LeaderElection) {
	//No leader
	if ou.LeaderElection.LeaderAddr == "" {
		//Update (randnum), id ,addr, path!!
		ou.LeaderElection.LeaderAddr = ou.Addr
		ou.LeaderElection.ID = ou.ID
		if !ListContains(ou.LeaderElection.LeaderPath, ou.Addr) {
			ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
		}

		go ou.gossipLeaderElection()
		return
	}

	//If the one sending out the leader election is the one we have here
	if ou.LeaderElection.ID == recLeaderData.ID {
		//If the random number generated is higher than we already registered
		if ou.LeaderElection.Number < recLeaderData.Number {
			ou.LeaderElection.LeaderPath = recLeaderData.LeaderPath
			if !ListContains(ou.LeaderElection.LeaderPath, ou.Addr) {
				ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
			}
			//Update randnum!!!
			ou.LeaderElection.Number = recLeaderData.Number
			ou.LeaderElection.LeaderAddr = recLeaderData.LeaderAddr
			go ou.gossipLeaderElection()
		} else {
			//Random numbers are equal
			if ou.LeaderElection.Number == recLeaderData.Number {
				//should we switch to a shorter path to leader?
				if len(ou.LeaderElection.LeaderPath) > len(recLeaderData.LeaderPath) {
					ou.LeaderElection.LeaderPath = recLeaderData.LeaderPath
				}
			} else {
				go ou.gossipLeaderElection()
			}
		}
	} else {
		//Different msg
		//Received rand-num is higher than the one we have.. Update leader..
		if recLeaderData.Number > ou.LeaderElection.Number {
			//Update randnum, id, addr, path!!!
			ou.LeaderElection = recLeaderData
			if !ListContains(ou.LeaderElection.LeaderPath, ou.Addr) {
				ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
			}

			go ou.gossipLeaderElection()
		} else if recLeaderData.Number == ou.LeaderElection.Number {
			//Same random number, but different leader
			//Check if shorter path
			if len(ou.LeaderElection.LeaderPath) > len(recLeaderData.LeaderPath) {
				//Update id, addr, path!!!
				ou.LeaderElection.ID = recLeaderData.ID
				ou.LeaderElection.LeaderAddr = recLeaderData.LeaderAddr
				ou.LeaderElection.LeaderPath = recLeaderData.LeaderPath
				//append self
				if !ListContains(ou.LeaderElection.LeaderPath, ou.Addr) {
					ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
				}

				go ou.gossipLeaderElection()

			} else {
				//Longer or equal path to leader so don't do anything..
			}
		} else {
			//Different leader, lower rand-number..
			//insert self to path..
			if !ListContains(ou.LeaderElection.LeaderPath, ou.Addr) {
				ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
			}

			go ou.gossipLeaderElection()
			return
		}
	}
}

/*Chose if node is the biggest..*/
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

/*accumulateSensorData accumulates data if not accumualted before and send to leader*/
func (ou *ObservationUnit) accumulateSensorData(sData SensorData) {
	var lock sync.RWMutex

	lock.Lock()
	defer lock.Unlock()

	if ou.SensorData.Accumulated == false {
		ou.SensorData.Data = append(ou.SensorData.Data[:], sData.Data[:]...)
		ou.SensorData.Accumulated = true
	}

	go ou.sendDataToLeader(ou.SensorData)
}

/*calculateLeaderThreshold algorithm for calculate if a node can be ch..*/
func (ou *ObservationUnit) calculateLeaderThreshold() {
	time.Sleep(time.Second * 1)
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

/*getData gathers data in intervals from nodes in the clusters*/
func (ou *ObservationUnit) getData() {
	var num uint32
	var count uint32
	num = 0
	count = 0
	//randAccCount := randomInt(1, 6)
	tickChan := time.NewTicker(time.Second * 100).C

	doneChan := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(batteryStart))
		doneChan <- true
	}()

	for {
		time.Sleep(time.Second * 40)
		if ou.LeaderElection.LeaderAddr == ou.Addr {
			select {
			case <-tickChan:
				fmt.Printf("\n\n(%s): Leader gathering data..\n", ou.Addr)
				num++
				ou.SensorData.ID = ou.ID + num
				tmp := ou.SensorData.Data
				ou.SensorData.Data = []byte{}

				go ou.notifyNeighboursGetData(ou.SensorData)

				ou.SensorData.Data = tmp
				ou.AccCount++
				time.Sleep(time.Second * 60)
				//if ou.AccCount == randAccCount {
				if ou.AccCount == 6 {
					ou.ClusterHeadCount++
					//go ChExperiments(os.Getpid(), ou.ClusterHeadCount, ou.ReceivedDataPkt)
					//Accumulated data x times, elect a new leader..
					fmt.Printf("\n\n------------\n(%s): LEADER SENT DATA 2 TIMES!!!\n------------\n", ou.Addr)

					ou.LeaderElection.Number = randomFloat()
					ou.LeaderCalculation.ID = hashAddress(ou.Addr)
					count++
					ou.LeaderCalculation.ID = ou.LeaderCalculation.ID + count
					ou.AccCount = 0

					time.Sleep(time.Second * 60)
					fmt.Printf("(%s): Leader gossip new leader calculation \n", ou.Addr)
					go ou.gossipNewLeaderCalculation()

					time.Sleep(time.Second * 100)
					fmt.Printf("(%s): Leader gossip new leader election \n", ou.Addr)
					go ou.gossipLeaderElection()
				}
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
	num := (rand.Float64() * 495) + 5
	//num := (rand.Float64() * 200) + 5
	//num := (rand.Float64() * 50) + 5
	//num := (rand.Float64() * 350) + 5
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
	}
}

/*batteryConsumption simulates a nodes battery lifetime..*/
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
				ou.BatteryTime -= (secondInterval * 4)
			} else {
				ou.BatteryTime -= secondInterval
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

/*ErrorMsg returns error if any*/
func ErrorMsg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

func printSlice(s []string) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}

//ListContains check if a value is in a list/slice and return true or false
func ListContains(s []string, e string) bool {
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
