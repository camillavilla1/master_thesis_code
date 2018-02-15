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
)

var hostname string
var ouPort string
var hostaddress string

var runningNodes []string
var partitionScheme int32

func main() {
	hostname = "localhost"
	flag.StringVar(&ouPort, "OUport", ":8080", "Super Observation Unit port (prefix with colon)")
	flag.Parse()

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

/*func broadcastReachablehosts() {
	fmt.Printf("\nBroadcasting reachable hosts\n")
	var nodeString string
	nodeString = ""
	hostaddress = hostname + ouPort

	nodeString = stringify(runningNodes)
	for _, addr := range runningNodes {
		url := fmt.Sprintf("http://%s/broadcastReachablehost", addr)
		fmt.Printf("Broadcast to: %s", url)
		if addr != hostaddress {
			fmt.Printf("\nwith: %s.\n", nodeString)
			addressBody := strings.NewReader(nodeString)
			http.Post(url, "string", addressBody)
		}
	}
}
*/


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
	}

	fmt.Printf("Running nodes are: %v\n", runningNodes)

	//broadcastReachablehosts()

	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
}

/*func reachableHosts() []string {
	if !listContains(runningNodes, addrString) {
		runningNodes = append(runningNodes, addrString)
	}
	return runningNodes
}*/

/*Receive from OUnodes who's running and append them to a list.*/
func reachableHostsHandler(w http.ResponseWriter, r *http.Request) {
	// We don't use the body, but read it anyway
	var addrString string

	pc, rateErr := fmt.Fscanf(r.Body, "%s", &addrString)
	if pc != 1 || rateErr != nil {
		log.Printf("Error parsing Post request: (%d items): %s", pc, rateErr)
	}

	if !listContains(runningNodes, addrString) {
		runningNodes = append(runningNodes, addrString)
	}

	//broadcastReachablehosts()

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


