package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

var hostname string
var ouPort string

var numOU int64

//var numCH int64
var num int64

var runningNodes []string
var numNodesRunning int

var nodeRadius float64
var gridX int32
var gridY int32

/*ObservationUnit is struct of an OU*/
type ObservationUnit struct {
	Addr                string
	ID                  uint32
	Pid                 int
	ReachableNeighbours []string
	Xcor                float64
	Ycor                float64
	CHpercentage        float64
}

var runningOus []ObservationUnit

var numCH = flag.Int("numCH", 0, "Number of Cluster Heads")

func main() {
	numNodesRunning = 0
	nodeRadius = 80.0
	gridX = 200
	gridY = 200

	hostname = "localhost"

	flag.StringVar(&ouPort, "Simport", ":8080", "Simulation port (prefix with colon)")

	flag.Parse()

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/notifySimulation", reachableOuHandler)
	http.HandleFunc("/removeReachableOu", removeReachableOuHandler)

	log.Printf("\nStarted simulation on %s%s\n", hostname, ouPort)

	err := http.ListenAndServe(ouPort, nil)

	if err != nil {
		log.Panic(err)
	}
}

/*IndexHandler is not used*/
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// We don't use the body, but read it anyway
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	fmt.Fprintf(w, "Index Handler\n")
}

/*Remove OUnodes that are dead/don't run anymore*/
func removeReachableOuHandler(w http.ResponseWriter, r *http.Request) {
	//var addrString string
	var ou ObservationUnit

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &ou); err != nil {
		panic(err)
	}

	if listContains(runningNodes, ou.Addr) {
		runningNodes = removeElement(runningNodes, ou.Addr)
		runningOus = removeObservationUnit(runningOus, ou.Addr)
		numNodesRunning--
	}

	fmt.Println("Number of nodes running: ", numNodesRunning)
	//fmt.Printf("Running nodes are: %v\n", runningNodes)

	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
}

/*Receive from OUnodes who's running and append them to a list.*/
func reachableOuHandler(w http.ResponseWriter, r *http.Request) {
	var ou ObservationUnit

	body, err := ioutil.ReadAll(r.Body)
	errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &ou); err != nil {
		panic(err)
	}
	//fmt.Println(ou)

	if !listContains(runningNodes, ou.Addr) {
		runningNodes = append(runningNodes, ou.Addr)
		runningOus = append(runningOus, ou)
		numNodesRunning++
	}

	fmt.Println("Number of nodes running: ", numNodesRunning)

	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	go getClusterheadPercentage(ou)
	go findNearestneighbours(ou)

	//fmt.Printf("\n")
}

func getClusterheadPercentage(ou ObservationUnit) {
	ou.CHpercentage = float64(*numCH) / 100

	//fmt.Printf("\n### Tell OU percentage of Cluster Heads.. ###\n")
	url := fmt.Sprintf("http://%s/clusterheadPercentage", ou.Addr)
	//fmt.Printf("SendingCH percentage to url: %s ", url)

	b, err := json.Marshal(ou.CHpercentage)
	if err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Printf(" with data: %s.\n", string(b))

	addressBody := strings.NewReader(string(b))

	res, err := http.Post(url, "string", addressBody)
	errorMsg("Post request telling OU about CH percentage failed: ", err)
	io.Copy(os.Stdout, res.Body)

}

/*Find nearest neighbour(s) that OUnode can contact.*/
func findNearestneighbours(ou ObservationUnit) {
	//fmt.Printf("\n### Find Nearest Neighbours!!###\n")
	for _, startedOu := range runningOus {
		//fmt.Printf("Running OU are: %+v\n", runningOus)
		if !(ou.ID == startedOu.ID) {
			distance := findDistance(ou.Xcor, ou.Ycor, startedOu.Xcor, startedOu.Ycor)

			if distance < nodeRadius {
				ou.ReachableNeighbours = append(ou.ReachableNeighbours, startedOu.Addr)
			}
		}
	}

	/*Tell OU about no neighbours or neighbours..*/
	if len(ou.ReachableNeighbours) >= 1 {
		//fmt.Println("# OU neighbours: ", len(ou.ReachableNeighbours))
		/*Need to sleep, or else it get connection refused.*/
		time.Sleep(1000 * time.Millisecond)
		go tellOuAboutReachableNeighbour(ou)
	} else {
		//fmt.Printf("OU have no neighbours..\n")
		time.Sleep(1000 * time.Millisecond)
		go tellOuNoReachableNeighbours(ou)
	}
}

func tellOuNoReachableNeighbours(ou ObservationUnit) {
	//fmt.Printf("\n### tell Ou NoReachableNeighbours ###\n")
	url := fmt.Sprintf("http://%s/noReachableNeighbours", ou.Addr)
	fmt.Printf("Sending no reachable neighbours to url: %s\n", url)

	message := "OU have no reachable neighbours!"
	addressBody := strings.NewReader(message)

	res, err := http.Post(url, "string", addressBody)
	errorMsg("Post request telling OU about no neighbours failed: ", err)
	io.Copy(os.Stdout, res.Body)
}

/*Tell OU about other OUs that are reachable for this specific OU.*/
func tellOuAboutReachableNeighbour(ou ObservationUnit) {
	url := fmt.Sprintf("http://%s/reachableNeighbours", ou.Addr)
	fmt.Printf("Sending neighbour to url: %s ", url)

	b, err := json.Marshal(ou.ReachableNeighbours)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("with data: ")
	fmt.Println(string(b))

	addressBody := strings.NewReader(string(b))

	res, err := http.Post(url, "string", addressBody)
	errorMsg("Post request telling OU about neighbours failed: ", err)
	io.Copy(os.Stdout, res.Body)
}

func errorMsg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

/*Find distance between the current node and the nearest neighbours by its "GPS" coordinates - middle of the radio-range for the OU.*/
func findDistance(startX float64, startY float64, stopX float64, stopY float64) float64 {
	xx := math.Pow((startX - stopX), 2)
	yy := math.Pow((startY - stopY), 2)

	res := math.Sqrt(xx + yy)
	return res
}

//Remove element in a slice
func removeElement(s []string, r string) []string {
	for i, v := range s {
		if len(s) == 1 {
			s = append(s[:i])
		} else {
			if v == r {
				return append(s[:i], s[i+1:]...)
			}
		}
	}
	return s
}

//Remove OU in a slice
func removeObservationUnit(s []ObservationUnit, r string) []ObservationUnit {
	for i := 0; i < len(s); i++ {
		if s[i].Addr == r {
			s = append(s[:i], s[i+1:]...)
		}
	}
	return s
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

func printSlice(s []string) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
