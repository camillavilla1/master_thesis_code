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
	"bytes"
)

var taken_port []int

//LEGGE INN FLAGS MED ANTALL OU OG ANTALL CH DET SKAL VÆRE I CLUSTERE(T/NE)
func main() {

	numThreads := runtime.NumCPU()
    runtime.GOMAXPROCS(numThreads)

	numServers := os.Args[1]
	fmt.Println(numServers)
	numServers2, err := strconv.Atoi(numServers)
	errorMsg("Str to int: ", err)

	port := 8081
	port2 := random(29170, 29998)
	fmt.Println("Random port number: ", port2)
	for i := 0; i < numServers2; i++ {
		//fmt.Println(i)

		port += 1
		s_port := strconv.Itoa(port)
		if !sliceContains(taken_port, port) {
			taken_port = append(taken_port, port)
			cmnd := exec.Command("go", "run", "server.go", "run", "-Simport=8080", "-host=localhost", "-port=:"+s_port)		
			errorMsg("Command error: ", err)
				
			fmt.Println(i)

			var out bytes.Buffer
			var stderr bytes.Buffer
			cmnd.Stdout = &out
			cmnd.Stderr = &stderr

			err := cmnd.Start()
		    errorMsg("Error starting process: ", err)
		    time.Sleep(1000 * time.Millisecond)
		    //err = cmnd.Wait()
		    //errorMsg("Wait to exit.. ", err)
		    
		    //err := cmnd.Run()
		    //errorMsg("Run process: ", err)
		    if err != nil {
			    fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			    return
			} else {
				fmt.Println("Result: " + out.String())
			}

			/*cmndIn, _ := cmnd.StdinPipe()
		    cmndOut, _ := cmnd.StdoutPipe()
		    fmt.Println(cmndIn)
		    fmt.Println(cmndOut)*/
		} else {
			fmt.Printf("Port is taken... Try again or something.\n")
		}
	}
	/*Need this so the program doesn't quit before the OU are done..*/
	time.Sleep(5000 * time.Millisecond)
	printSlice(taken_port)
}


func errorMsg(s string, err error) {
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
	rand.Seed(time.Now().UTC().UnixNano())
    return rand.Intn(max - min) + min
}


func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}