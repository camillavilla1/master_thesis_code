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
	ClusterHead         string   `json:"-"`
	IsClusterHead       bool     `json:"-"`
	ClusterHeadCount    int      `json:"-"`
	Bandwidth           int      `json:"-"`
	PathToCh            []string `json:"-"`
	CHpercentage        float64  `json:"-"`
	RandLeaderNumber    float64  `json:"-"`
	SensorData          `json:"-"`
	DataBaseStation     `json:"-"`
	ElectNewLeader      `json:"-"`
	LeaderElection      `json:"-"`
}

/*CHpkt is a struct containg info about path to CH*/
type CHpkt struct {
	Path        []string
	Source      string
	Destination string
	ClusterHead string
}

/*SensorData is data from "sensors" on the OU*/
type SensorData struct {
	ID          uint32
	Fingerprint uint32
	Data        []byte
	Source      string
	Destination string
}

/*DataBaseStation is data sent to/gathered from the BS*/
type DataBaseStation struct {
	BSdatamap map[uint32][]byte
}

/*ElectNewLeader contains info about new leader election*/
type ElectNewLeader struct {
	Source string
	ID     uint32
}

/*LeaderElection contains info about new leader election*/
type LeaderElection struct {
	ID         uint32
	Number     float64
	LeaderPath []string
	LeaderAddr string
	First      bool
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
	batteryStart = 120
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
		RandLeaderNumber:    0,
		SensorData: SensorData{
			ID:          0,
			Fingerprint: 0,
			Data:        []byte{},
			Source:      "",
			Destination: ""},
		DataBaseStation: DataBaseStation{
			BSdatamap: make(map[uint32][]byte)},
		ElectNewLeader: ElectNewLeader{
			Source: "",
			ID:     0},
		LeaderElection: LeaderElection{
			ID:         hashAddress(hostaddress),
			Number:     0.0,
			LeaderPath: []string{},
			LeaderAddr: "",
			First:      true}}

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
	http.HandleFunc("/broadcastElectNewLeader", ou.broadcastElectNewLeaderHandler)

	http.HandleFunc("/gossipLeader", ou.gossipLeaderHandler)

	go ou.checkBatteryStatus()
	go ou.batteryConsumption()
	go ou.tellSimulationUnit()
	//go ou.measureSensorData()
	//go ou.getData()

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

	//go ou.clusterHeadElection()
	go ou.clusterHeadCalculation()
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

	var data []string
	data = append(data, ou.Addr)
	data = append(data, newNeighbour)
	data = append(data, ou.ClusterHead)

	if !listContains(ou.Neighbours, newNeighbour) {
		ou.Neighbours = append(ou.Neighbours, newNeighbour)
	}

	time.Sleep(1000 * time.Millisecond)
	go ou.tellContactingOuOk(data)
}

/*Receive a new leader from a neighbour. Update clusterhead and clusterhead-status*/
func (ou *ObservationUnit) broadcastNewLeaderHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("\n\n------------------------------------------\n###(%s): BROADCAST HANDLER.. Received a new leader. ###\n------------------------------------------\n", ou.Addr)
	var pkt CHpkt

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &pkt); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	//fmt.Printf("(%s): Received packet from %s\n", ou.Addr, pkt.Source)

	if ou.ClusterHead != pkt.ClusterHead {
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

			if !listContains(pkt.Path, ou.Addr) {
				pkt.Path = append(pkt.Path, ou.Addr)
			}
			go ou.broadcastNewLeader(pkt)
		} else {
			continue
		}
	}
}

/*Receive a msg from CH about sending (accumulated) data to CH. */
func (ou *ObservationUnit) notifyNeighboursGetDataHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("\n\n------------------------------------------\n###(%s): NOTIFY NEIGHBOURS DATA HANDLER. ###\n------------------------------------------\n", ou.Addr)
	var sData SensorData

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &sData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	fmt.Printf("\n(%s): Received get-data-msg from %s\n", ou.Addr, sData.Source)

	if sData.ID != ou.SensorData.ID {
		//fmt.Printf("\n(%s): Have not received this msg before. Need to forward msg to neighbours..\n", ou.Addr)
		ou.SensorData.ID = sData.ID

		go ou.notifyNeighboursGetData(sData)

		//if ch is neighbour, no need for accumulate data..
		if len(ou.PathToCh) <= 1 {
			if ou.PathToCh[0] == ou.ClusterHead {
				fmt.Printf("\n(%s) Send data to CH \n", ou.Addr)
				go ou.sendDataToLeader(sData)
			}
		} else {
			//fmt.Printf("\n(%s) Accumulate data and send to CH (through path)\n", ou.Addr)
			//Need to accumulate data with neighbours
			go ou.accumulateSensorData(sData)
			//go ou.sendDataToLeader(sData)
		}
	} /*else {
		fmt.Printf("(%s): Data id and sensordata id similar..\n", ou.Addr)
	}*/
}

func (ou *ObservationUnit) sendDataToLeaderHandler(w http.ResponseWriter, r *http.Request) {
	var sData SensorData
	//var lock sync.RWMutex

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &sData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	if ou.IsClusterHead == true {
		if len(ou.BSdatamap) == 0 {
			//lock.Lock()
			//defer lock.Unlock()
			//fmt.Printf("(%s): Locked for writing to map..\n", ou.Addr)

			ou.BSdatamap[sData.Fingerprint] = sData.Data
			//fmt.Printf("(%s): [0] Added data to BSdatamap\n", ou.Addr)
		} else {
			for key := range ou.BSdatamap {
				//fmt.Println("key:", key, "value:", []byte(value))
				if key == sData.Fingerprint {
					//fmt.Printf("(%s): [1] Key and Fingerprint is similar: %d\n", ou.Addr, key)
					//log.Printf("(%s):APPENDING %+v", ou.Addr, append(ou.BSdatamap[sData.Fingerprint][:], sData.Data[:]...))
				} else if key != sData.Fingerprint {
					ou.BSdatamap[sData.Fingerprint] = sData.Data
					//fmt.Printf("(%s): [2]  Added data to BSdatamap\n\n", ou.Addr)
				}
			}
		}
		fmt.Printf("\n------------\n(%s): MAP IS: %+v\n------------\n\n", ou.Addr, ou.BSdatamap)
	} else {
		//fmt.Printf("\n(%s): is not CH. Accumulate data and send to CH..\n", ou.Addr)
		go ou.accumulateSensorData(sData)
	}
}

/*Receive ok from CH that new OU can join.*/
func (ou *ObservationUnit) connectingOkHandler(w http.ResponseWriter, r *http.Request) {
	var data []string

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &data); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	neighOu := strings.Join(data[:1], "") //first element
	//newOu := strings.Join(data[1:2],"") //middle element, nr 2
	clusterHead := strings.Join(data[2:], "")

	if !listContains(ou.Neighbours, neighOu) {
		ou.Neighbours = append(ou.Neighbours, neighOu)
	}

	ou.ClusterHead = clusterHead
}

func (ou *ObservationUnit) broadcastElectNewLeaderHandler(w http.ResponseWriter, r *http.Request) {
	var electLeaderData ElectNewLeader

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &electLeaderData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	fmt.Printf("(%s): ElectLeaderData is : %+v\n", ou.Addr, electLeaderData)

	ou.ElectNewLeader.ID = electLeaderData.ID
	ou.ElectNewLeader.Source = electLeaderData.Source

	if electLeaderData.ID != ou.ElectNewLeader.ID {
		go ou.clusterHeadCalculation()
		if ou.IsClusterHead == false {
			go ou.broadcastElectNewLeader()
		}
	}
}

func (ou *ObservationUnit) gossipLeaderHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n(%s) Gossip leader handler!\n", ou.Addr)
	var recLeaderData LeaderElection

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &recLeaderData); err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()

	//ou.LeaderElection.Number = recLeaderData.Number

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
func (ou *ObservationUnit) tellContactingOuOk(data []string) {
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

	addressBody := strings.NewReader(string(b))

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

		_, err := http.Post(url, "string", addressBody)
		if err != nil {
			fmt.Printf("(%s) Try to post broadcast %s\n", ou.Addr, err)
			continue
		}
	}

	if i == len(ou.ReachableNeighbours) {
		time.Sleep(2 * time.Second)
		go ou.clusterHeadCalculation()
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
			//fmt.Printf("\n(%s): Contacting neighbour url: %s ", ou.Addr, url)

			pkt.Source = ou.Addr
			pkt.Destination = addr

			b, err := json.Marshal(pkt)
			if err != nil {
				fmt.Println(err)
				return
			}

			addressBody := strings.NewReader(string(b))
			//fmt.Println("\nAddressbody: ", addressBody)

			_, err = http.Post(url, "string", addressBody)
			errorMsg("Error posting to neighbour ", err)
			/*if err != nil {
				fmt.Printf("(%s): Try to post broadcast %s\n", ou.Addr, err)
				continue
			}*/
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
			fmt.Printf("\n(%s): Contacting neighbour url: %s\n", ou.Addr, url)

			/*	if !listContains(sensorData.Path, ou.Addr) {
				sensorData.Path = append(sensorData.Path, ou.Addr)
			}*/
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
	fmt.Printf("\n(%s): Send data to last node in path: %s ..\n\n", ou.Addr, lastElem)
	//removing last element
	//newPath := ou.PathToCh[:len(ou.PathToCh)-1]
	//fmt.Printf("\n(%s): New path looks like: %s\n", ou.Addr, newPath)

	url := fmt.Sprintf("http://%s/sendDataToLeader", lastElem)
	fmt.Printf("\n(%s): Contacting neighbour url: %s ", ou.Addr, url)

	//sensorData.Source = ou.Addr

	sensorData.Source = ou.Addr
	sensorData.Destination = lastElem
	sensorData.Fingerprint = hashByte(ou.SensorData.Data)
	sensorData.Data = ou.SensorData.Data

	//fmt.Printf("\n(%s): sending: %+v\n", ou.Addr, sensorData)

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
	//fmt.Printf("(%s): Sending to url: %s.\n", ou.Addr, url)

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

func (ou *ObservationUnit) broadcastElectNewLeader() {
	for _, addr := range ou.Neighbours {
		if ou.ElectNewLeader.Source != addr {
			url := fmt.Sprintf("http://%s/broadcastElectNewLeader", addr)
			fmt.Printf("(%s):broadcast elect new leader to url: %s \n", ou.Addr, url)

			ou.ElectNewLeader.ID = hashAddress(ou.Addr)
			ou.ElectNewLeader.Source = ou.Addr

			b, err := json.Marshal(ou.ElectNewLeader)
			if err != nil {
				fmt.Println(err)
				return
			}

			addressBody := strings.NewReader(string(b))

			_, err = http.Post(url, "string", addressBody)
			errorMsg("Post request broadcasting election of new leader failed: ", err)
			//io.Copy(os.Stdout, res.Body)
		}
	}
}

func (ou *ObservationUnit) gossipLeader() {
	fmt.Printf("(%s): Gossip leader to neighbours..\n", ou.Addr)
	//time.Sleep(time.Second * 2)
	for _, addr := range ou.Neighbours {
		url := fmt.Sprintf("http://%s/gossipLeader", addr)
		fmt.Printf("(%s): Gossip new leader to url: %s \n", ou.Addr, url)

		b, err := json.Marshal(ou.LeaderElection)
		if err != nil {
			fmt.Println(err)
			return
		}

		addressBody := strings.NewReader(string(b))

		_, err = http.Post(url, "string", addressBody)
		errorMsg("Post request broadcasting election of new leader failed: ", err)
		//io.Copy(os.Stdout, res.Body)
	}
}

func (ou *ObservationUnit) leaderElection(recLeaderData LeaderElection) {
	fmt.Printf("\n(%s): LEADER ELECTION!!\n", ou.Addr)

	//No leader
	if ou.ClusterHead == "" {
		fmt.Printf("(%s): Have no leader\n", ou.Addr)

		ou.updateLeader(recLeaderData)
		go ou.gossipLeader()
		ou.LeaderElection.First = false
		return
	}

	//If the one sending out the leader election is the one we have here
	if ou.LeaderElection.ID == recLeaderData.ID {
		fmt.Printf("(%s): OU ID (%d) and received ID (%d) are similar.\n", ou.Addr, ou.LeaderElection.ID, recLeaderData.ID)
		//If the random number generated is higher than we already registered
		if ou.LeaderElection.Number > recLeaderData.Number {
			fmt.Printf("(%s): ID similar:  This OU rand-number (%f) is greater than received rand-number (%f)\n", ou.Addr, ou.RandLeaderNumber, recLeaderData.Number)
			//ou.updateLeader(recLeaderData)
			ou.gossipLeader()
		}

		//Random numbers are equal
		if ou.LeaderElection.Number == recLeaderData.Number {
			fmt.Printf("(%s): ID similar: This OU rand-number (%f) is equal to received rand-number (%f)\n", ou.Addr, ou.RandLeaderNumber, recLeaderData.Number)
			fmt.Printf("(%s): Don't do anyting except updating to shorter path to leader if possible..", ou.Addr)
			//should we switch to a shorter path to leader?
			if len(ou.PathToCh) > len(recLeaderData.LeaderPath) {
				fmt.Printf("(%s): Updating to shorter path to leader", ou.Addr)
				ou.updateLeader(recLeaderData)
			}

			if ou.LeaderElection.First == true {
				ou.gossipLeader()
			}

		}
	} else {
		//Different leader
		fmt.Printf("(%s): Different leader\n", ou.Addr)

		//Received rand-num is higher than the one we have.. Update leader..
		if recLeaderData.Number > ou.LeaderElection.Number {
			fmt.Printf("(%s): Received rand-num (%f) is greater than what we have (%f)\n", ou.Addr, recLeaderData.Number, ou.LeaderElection.Number)
			ou.updateLeader(recLeaderData)
			ou.gossipLeader()
		} else if recLeaderData.Number == ou.LeaderElection.Number {
			fmt.Printf("(%s): Received rand-num (%f) is equal to what we have (%f)\n", ou.Addr, recLeaderData.Number, ou.LeaderElection.Number)
			//Same random number, but different leader
			//Check if shorter path
			if len(ou.PathToCh) > len(recLeaderData.LeaderPath) {
				fmt.Printf("(%s): Received path to leader is shorter than what we have. Update leader\n", ou.Addr)
				ou.updateLeader(recLeaderData)
			} else {
				fmt.Printf("(%s): Received path to leader is longer than what we have. Don't update leader.\n", ou.Addr)
				//Longer or equal path to CH so don't do anything..
			}
		} else {
			//Different leader, lower rand-number..
			fmt.Printf("(%s): Received rand-num (%f) is lower than what we have (%f)\n", ou.Addr, recLeaderData.Number, ou.LeaderElection.Number)
			//insert self to path..
			if !listContains(ou.LeaderElection.LeaderPath, ou.Addr) {
				ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
			}
			ou.gossipLeader()
		}

	}
}

func (ou *ObservationUnit) updateLeader(recLeaderData LeaderElection) {
	fmt.Printf("(%s): Update Leader information\n", ou.Addr)
	ou.ClusterHead = recLeaderData.LeaderAddr
	//Switch to shortest path to CH
	if !listContains(ou.PathToCh, recLeaderData.LeaderAddr) {
		ou.PathToCh = recLeaderData.LeaderPath
	}
	ou.LeaderElection = recLeaderData
	fmt.Printf("(%s): Leader information looks like this now: %+v\n", ou.Addr, ou.LeaderElection)

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

func (ou *ObservationUnit) clusterHeadElection() {
	//fmt.Printf("\n### Cluster Head Election ###\n")
	//fmt.Printf("OU IS %s\n", ou.Addr)

	var pkt CHpkt

	if ou.Addr == "localhost:8084" && len(ou.ReachableNeighbours) != 0 {
		fmt.Printf("\n\nLOCALHOST:8084 IS CH!!!\n\n")
		ou.ClusterHeadCount++
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
	//log.Printf("(%s):sensordata was %+v -> APPENDING and now %+v", ou.Addr, ou.SensorData.Data, append(ou.SensorData.Data[:], sData.Data[:]...))
	//fmt.Printf("\n(%s):sensordata was %+v", ou.Addr, ou.SensorData.Data)
	ou.SensorData.Data = append(ou.SensorData.Data[:], sData.Data[:]...)
	//fmt.Printf("\n(%s) sensordata is now %+v\n", ou.Addr, ou.SensorData.Data)
	go ou.sendDataToLeader(ou.SensorData)
	//ou.SensorData.Data = append(ou.SensorData.Data, i)
}

func (ou *ObservationUnit) clusterHeadCalculation() {
	ou.RandLeaderNumber = randomFloat()
	ou.LeaderElection.Number = ou.RandLeaderNumber
	ou.LeaderElection.LeaderPath = append(ou.LeaderElection.LeaderPath, ou.Addr)
	ou.LeaderElection.LeaderAddr = ou.Addr
	fmt.Printf("\n(%s): Leader election is : %+v\n", ou.Addr, ou.LeaderElection)
	go ou.leaderElection(ou.LeaderElection)

	/*var pkt CHpkt

	//if battery is under 20% cannot OU be CH
	if float64(ou.BatteryTime) < (float64(batteryStart) * 0.20) {
		fmt.Printf("(%s): cannot be CH because of low battery\n", ou.Addr)
	} else {
		randNum := randomFloat()
		threshold := ou.threshold()

		if randNum < threshold {
			//if randNum > threshold {
			fmt.Printf("\n---------------------\n(%s): CAN BE CH!!!! BROADCAST NEW LEADER TO NEIGHBOURS\n---------------------\n", ou.Addr)
			ou.ClusterHeadCount++
			ou.ClusterHead = ou.Addr
			ou.IsClusterHead = true
			pkt.ClusterHead = ou.Addr
			go ou.broadcastLeaderPath(pkt)
			//go ou.broadcastNewLeader(pkt)
		}
	}*/
}

func (ou *ObservationUnit) getData() {
	var num uint32
	num = 0
	tickChan := time.NewTicker(time.Second * 30).C

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
				num++
				ou.SensorData.ID = ou.ID + num
				tmp := ou.SensorData.Data
				ou.SensorData.Data = []byte{}
				fmt.Printf("(%s): SENSORDATA ID: %d\n", ou.Addr, ou.SensorData.ID)
				go ou.notifyNeighboursGetData(ou.SensorData)
				ou.SensorData.Data = tmp
			}

		case <-doneChan:
			return
		}
	}
}

/*randomFloat returns a random number in [0.0,1.0)*/
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
	return bs
}

func estimateLocation() float64 {
	rand.Seed(time.Now().UTC().UnixNano())
	//num := (rand.Float64() * 495) + 5
	num := (rand.Float64() * 150) + 5
	return num
}

/*func (ou *ObservationUnit) batteryConsumption2() {

	tmp := randomInt64(batteryStart)
	fmt.Printf("(%s): Tmp battery is %d\n", ou.Addr, tmp)

	if float64(tmp) < (float64(batteryStart) * 0.40) {
		fmt.Printf("(%s): Not much battery left..\n", ou.Addr)
		if ou.IsClusterHead == true {
			ou.ElectNewLeader.ID = hashAddress(ou.Addr)
			ou.ElectNewLeader.Source = ou.Addr
			fmt.Printf("(%s): Is CH, broadcast to neighbours about electing a new leader..\n", ou.Addr)
			go ou.broadcastElectNewLeader()
		}

	}
}*/

func (ou *ObservationUnit) checkBatteryStatus() {
	for {
		if float64(ou.BatteryTime) <= (float64(batteryStart) * 0.50) {
			fmt.Printf("\n\n(%s): BATTERY IS UNDER 20\n", ou.Addr)
			break
		}
	}

	if ou.IsClusterHead == true {
		fmt.Printf("\n(%s): have low battery: %d.. Need to chose a new CH\n", ou.Addr, ou.BatteryTime)
		ou.ElectNewLeader.ID = hashAddress(ou.Addr)
		ou.ElectNewLeader.Source = ou.Addr
		go ou.broadcastElectNewLeader()
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

			if ou.IsClusterHead == true {
				ou.BatteryTime -= (secondInterval * 2)
			} else {
				ou.BatteryTime -= secondInterval
			}

			/*fmt.Printf("(%s): batterytime: %d\n", ou.Addr, ou.BatteryTime)

			if float64(ou.BatteryTime) <= (float64(batteryStart) * 0.50) {
				fmt.Printf("\n(%s): have low battery.. Need to chose a new CH\n", ou.Addr)
				ou.ElectNewLeader.ID = hashAddress(ou.Addr)
				ou.ElectNewLeader.Source = ou.Addr
				go ou.broadcastElectNewLeader()
			}*/
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
	rand.Read(ou.SensorData.Data)
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

	return bw[k]

}
