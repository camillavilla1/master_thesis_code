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

type ObservationUnit struct {
	Addr string
	Id uint32
	Pid int
	Neighbors []string
	LocationDistance float32
	//clusterHead string
	//temperature int
	//weather string
}



func main() {
	numNodesRunning = 0
	clusterHead = ""
	hostname = "localhost"
	flag.StringVar(&ouPort, "OUport", ":8080", "Super Observation Unit port (prefix with colon)")
	numOU := flag.Int("numOU", 0, "Numbers of OUs running")
	numCH := flag.Int("numCH", 0, "Number of Cluster Heads")
	flag.Parse()

	fmt.Println(*numOU, *numCH)


	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/reachablehosts", reachableHostsHandler)
	http.HandleFunc("/removeReachablehost", removeReachablehostHandler)
	http.HandleFunc("/fetchReachablehosts", fetchReachableHostsHandler)
	

	log.Printf("Started Base Station 2 on %s%s\n", hostname, ouPort)

	err := http.ListenAndServe(ouPort, nil)

	if err != nil {
		log.Panic(err)
	}
}


func errorMsg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

func stringify(input []string) string {
	return strings.Join(input, ", ")
}


func fetchReachableHostsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n### FetchReachablehosts ###\n")
	// We don't use the body, but read it anyway
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	fmt.Printf("Running nodes are: ")
	printSlice(runningNodes)
	
	for _, host := range runningNodes {
		fmt.Fprintln(w, host)
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
func reachableHostsHandler(w http.ResponseWriter, r *http.Request) {

    var ou ObservationUnit

    body, err := ioutil.ReadAll(r.Body)
    errorMsg("readall: ", err)

	if err := json.Unmarshal(body, &ou); err != nil {
        panic(err)
    }
    fmt.Println(ou)

    if !listContains(runningNodes, ou.Addr) {
		runningNodes = append(runningNodes, ou.Addr)
		numNodesRunning += 1
	}

	fmt.Println("Number of nodes running: ", numNodesRunning)

	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

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


