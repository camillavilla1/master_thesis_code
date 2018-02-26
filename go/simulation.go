package main

import (
	"flag"
	"net/http"
	"log"
	"io"
	"fmt"
	"io/ioutil"
	"strings"
	//"sync/atomic"
	"encoding/json"
	"math"
	"time"
	"os"
	//"bytes"
)

var hostname string
var ouPort string

var numOU int64
var numCH int64
var clusterHead string
//var oldClusterHead string

var runningNodes []string
var numNodesRunning int


var runningCH []string
var oldRunningCH []string

var nodeRadius float64
var gridX int32
var gridY int32 

type ObservationUnit struct {
	Addr string
	Id uint32
	//Pid int
	Neighbours []string
	//LocationDistance float32
	Xcor float64
	Ycor float64
	//clusterHead string
	//temperature int
	//weather string
}

var runningOus []ObservationUnit


func main() {
	numNodesRunning = 0
	nodeRadius = 100.0
	gridX = 500
	gridY = 500

	clusterHead = ""
	hostname = "localhost"

	flag.StringVar(&ouPort, "Simport", ":8080", "Simulation port (prefix with colon)")
	numOU := flag.Int("numOU", 0, "Numbers of OUs running")
	numCH := flag.Int("numCH", 0, "Number of Cluster Heads")
	flag.Parse()

	fmt.Println(*numOU, *numCH)


	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/notifySimulation", reachableHostHandler)
	http.HandleFunc("/removeReachablehost", removeReachablehostHandler)

	log.Printf("Started simulation on %s%s\n", hostname, ouPort)

	err := http.ListenAndServe(ouPort, nil)

	if err != nil {
		log.Panic(err)
	}
}


func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// We don't use the body, but read it anyway
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	body := "Super Observation Unit running on " + hostname
	fmt.Fprintf(w, "<h1>%s</h1></br><p>Post segments to to /segment</p>", body)
}


/*Remove OUnodes that are dead/don't run anymore*/
func removeReachablehostHandler(w http.ResponseWriter, r *http.Request) {
	var addrString string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &addrString)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	if listContains(runningNodes, addrString) {
		runningNodes = removeElement(runningNodes, addrString)
		numNodesRunning -= 1
	}

	fmt.Printf("Running nodes are: %v\n", runningNodes)
	fmt.Println("Number of nodes running: ", numNodesRunning)


	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
}



/*Receive from OUnodes who's running and append them to a list.*/
func reachableHostHandler(w http.ResponseWriter, r *http.Request) {
    var ou ObservationUnit

    body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)

    //fmt.Printf(string(body))

	if err := json.Unmarshal(body, &ou); err != nil {
        panic(err)
    }
    //fmt.Println(ou)

    if !listContains(runningNodes, ou.Addr) {
		runningNodes = append(runningNodes, ou.Addr)
		runningOus = append(runningOus, ou)
		numNodesRunning += 1
	}

	fmt.Println("Number of nodes running: ", numNodesRunning)

	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	go findNearestneighbours(ou)

	fmt.Printf("\n")
}

/*Find nearest neighbour(s) that OUnode can contact.*/
func findNearestneighbours(ou ObservationUnit) {
	for _, startedOu := range runningOus {
		fmt.Printf("Running OU are: %+v\n", runningOus)
		fmt.Printf("-----------\n")
		if !(ou.Id == startedOu.Id) {
			distance := findDistance(ou.Xcor, ou.Ycor, startedOu.Xcor, startedOu.Ycor)
			//fmt.Println(distance)
			if distance < nodeRadius {
				fmt.Printf("Node are in range!\n")
				/*Need to sleep, or else it get connection refused.*/
				//time.Sleep(1000 * time.Millisecond)
				//tellNodeAboutNeighbour(ou, startedOu)
				ou.Neighbours = append(ou.Neighbours, startedOu.Addr)
			} else {
				fmt.Printf("Node is not in range..\n")
				//time.Sleep(1000 * time.Millisecond)
				//tellNodeAboutNeighbour(ou, startedOu)
				ou.Neighbours = append(ou.Neighbours, startedOu.Addr)
				fmt.Printf("Should have appended to OUs neighbours list...\n\n")
			}
		} else {
			fmt.Printf("Node is the same as in list..\n")
		}
	}
	printSlice(ou.Neighbours)
	time.Sleep(1000 * time.Millisecond)
	if len(ou.Neighbours) >= 1 {
		fmt.Println("hello", len(ou.Neighbours))
		go tellNodeAboutNeighbour(ou)
		
	}
}

func stringify(input []string) string {
	return strings.Join(input, ",")
}

/*Tell OU about other OUs that are reachable for this specific OU.*/
func tellNodeAboutNeighbour(ou ObservationUnit) {
	url := fmt.Sprintf("http://%s/neighbour", ou.Addr)
	fmt.Printf("Sending neighbour to url: %s\n", url)

	//neighbours := stringify(ou.Neighbours)
	//fmt.Printf(neighbours)

	b, err := json.Marshal(ou.Neighbours)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("\nwith data: ")
	fmt.Println(string(b))

	//addressBody := strings.NewReader(ou)
	addressBody := strings.NewReader(string(b))

	//_, err := http.Post(url, "string", addressBody)
	//errorMsg("Error posting to OU ", err)


	/*b, err := json.Marshal(ou)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("\nwith data: ")
	fmt.Println(string(b))

	//addressBody := strings.NewReader(string(b))
	addressBody2 := bytes.NewReader(b)
	*/
	res, err := http.Post(url, "string", addressBody)
	errorMsg("POST request to OU failed: ", err)
	io.Copy(os.Stdout, res.Body)
}


func errorMsg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

/*Find distance between the current node and the nearest neighbours by its "GPS" coordinates*/
func findDistance(startX float64, startY float64, stopX float64, stopY float64) float64 {
	xx := math.Pow((startX-stopX), 2)
	yy := math.Pow((startY-stopY), 2)

	res := math.Sqrt(xx+yy)
	return res
}

//Remove element in a slice
func removeElement(s []string, r string) []string {
	for i, v := range s {
		if len(s) == 1 {
			s = append(s[:i])
		}else {
			if v == r {
				return append(s[:i], s[i+1:]...)
			}
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


