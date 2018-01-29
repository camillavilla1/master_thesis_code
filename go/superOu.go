package main

import (
	"flag"
	"net/http"
	"log"
	"io"
	"fmt"
	"io/ioutil"
	//"sync/atomic"
)

var hostname string
var ouPort string

var runningNodes []string
var partitionScheme int32

func main() {
	hostname = "localhost"
	flag.StringVar(&ouPort, "OUport", ":8080", "Super Observation Unit port (prefix with colon)")
	flag.Parse()

	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/reachablehosts", reachableHostsHandler)

	log.Printf("Started Super Observation Unit on %s%s\n", hostname, ouPort)

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


func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// We don't use the body, but read it anyway
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	body := "Super Observation Unit running on " + hostname
	fmt.Fprintf(w, "<h1>%s</h1></br><p>Post segments to to /segment</p>", body)
}


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
	fmt.Printf("%v\n", runningNodes)
	//Check if node is list already..

	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
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