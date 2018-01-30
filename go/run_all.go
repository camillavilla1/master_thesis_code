package main 

import (
	"os/exec"
	"fmt"
	"math/rand"
	"time"
	"strconv"
	"log"
	"os"
	"runtime"
)

var taken_port []int


func main() {

	numThreads := runtime.NumCPU()
    runtime.GOMAXPROCS(numThreads)

	numServers := os.Args[1]
	fmt.Println(numServers)
	numServers2, err := strconv.Atoi(numServers)
	error_msg("Str to int: ", err)

	port := 8081
	for i := 0; i < numServers2; i++ {
		//fmt.Println(i)

		port += 1
		s_port := strconv.Itoa(port)
		if !sliceContains(taken_port, port) {
			taken_port = append(taken_port, port)
			cmnd := exec.Command("go", "run", "server.go", "run", "-SOUport=8080 -localhost=localhost -port=:"+s_port)		

			//fmt.Println(cmnd)

		    err := cmnd.Start()
		    error_msg("Start process: ", err)

			/*cmndIn, _ := cmnd.StdinPipe()
		    cmndOut, _ := cmnd.StdoutPipe()
		    fmt.Println(cmndIn)
		    fmt.Println(cmndOut)*/
		} else {
			fmt.Printf("Port is taken... Try again or something.\n")
		}
	}
	printSlice(taken_port)
}


func error_msg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

//Check if a value is in a list/slice and return true or false
func sliceContains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}


func random(min, max int) int {
    rand.Seed(time.Now().Unix())
    return rand.Intn(max - min) + min
}


func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}