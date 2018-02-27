package main

import (
	"flag"
	"net/http"
	"log"
	"io"
	"fmt"
	"io/ioutil"
	"strings"
	"encoding/json"
	"math"
	"time"
	"os"
)

var hostname string
var ouPort string

var numOU int64
var numCH int64


var runningNodes []string
var numNodesRunning int

var nodeRadius float64
var gridX int32
var gridY int32 

type ObservationUnit struct {
	Addr string
	Id uint32
	Pid int
	Neighbours []string
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

	hostname = "localhost"

	flag.StringVar(&ouPort, "Simport", ":8080", "Simulation port (prefix with colon)")
	numOU := flag.Int("numOU", 0, "Numbers of OUs running")
	numCH := flag.Int("numCH", 0, "Number of Cluster Heads")
	flag.Parse()

	fmt.Println(*numOU, *numCH)


	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/notifySimulation", reachableOuHandler)
	http.HandleFunc("/removeReachableOu", removeReachableOuHandler)

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

	//body := "Simulation Unit running on " + hostname
	//fmt.Fprintf(w, "Simulation IndexHandler", body)
	fmt.Fprintf(w, "Index Handler\n")
}


/*Remove OUnodes that are dead/don't run anymore*/
func removeReachableOuHandler(w http.ResponseWriter, r *http.Request) {
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
func reachableOuHandler(w http.ResponseWriter, r *http.Request) {
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
	fmt.Printf("\n### Find Nearest Neighbours!!###\n")
	for _, startedOu := range runningOus {
		//fmt.Printf("Running OU are: %+v\n", runningOus)
		if !(ou.Id == startedOu.Id) {
			distance := findDistance(ou.Xcor, ou.Ycor, startedOu.Xcor, startedOu.Ycor)

			if distance < nodeRadius {
				fmt.Printf("\n--- OU IS IN RANGE!! ---\n")
				ou.Neighbours = append(ou.Neighbours, startedOu.Addr)
			} else {
				fmt.Printf("OU is NOT in range..\n")
				/*Should tell node that no OU is available..
				noNeighbours(ou)*/
				//ou.Neighbours = append(ou.Neighbours, startedOu.Addr)
				//go tellOuNoNeighbours(ou)
			}
		} else {
			fmt.Printf("Node is the same as in list..\n")
			//go tellOuNoNeighbours(ou)
		}
	}
	//printSlice(ou.Neighbours)

	/*Tell OU about no neighbours or neighbours..*/
	if len(ou.Neighbours) >= 1 {
		fmt.Println("# neighbours: ", len(ou.Neighbours))
		/*Need to sleep, or else it get connection refused.*/
		time.Sleep(1000 * time.Millisecond)
		go tellOuAboutNeighbour(ou)	
	} else {
		time.Sleep(1000 * time.Millisecond)
		go tellOuNoNeighbours(ou)
	}
}

func tellOuNoNeighbours(ou ObservationUnit) {
	fmt.Printf("\n### tell Ou NoNeighbours ###\n")
	url := fmt.Sprintf("http://%s/noNeighbours", ou.Addr)
	fmt.Printf("Sending no neighbours to url: %s\n", url)

	message := "OU have no neighbours!"
	addressBody := strings.NewReader(message)


	res, err := http.Post(url, "string", addressBody)
	errorMsg("Post request telling OU about no neighbours failed: ", err)
	io.Copy(os.Stdout, res.Body)
}

/*Tell OU about other OUs that are reachable for this specific OU.*/
func tellOuAboutNeighbour(ou ObservationUnit) {
	url := fmt.Sprintf("http://%s/neighbours", ou.Addr)
	fmt.Printf("Sending neighbour to url: %s\n", url)

	b, err := json.Marshal(ou.Neighbours)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("\nwith data: ")
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


