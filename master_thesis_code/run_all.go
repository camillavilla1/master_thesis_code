package main

import (
	"bytes"
	"fmt"
	"go/build"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"time"
)

var takenPort []int

//LEGGE INN FLAGS MED ANTALL OU OG ANTALL CH DET SKAL VÆRE I CLUSTERE(T/NE)
func main() {
	// capture ctrl+c and stop CPU profiler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Printf("\nCaptured %v. Exiting..\n", sig)
			os.Exit(1)
		}
	}()

	numThreads := runtime.NumCPU()
	runtime.GOMAXPROCS(numThreads)

	numServers := os.Args[1]
	fmt.Println(numServers)
	numServers2, err := strconv.Atoi(numServers)
	errorMsg("Str to int: ", err)

	port := 8081
	//port2 := random(29170, 29998)
	//fmt.Println("Random port number: ", port2)
	for i := 0; i < numServers2; i++ {
		//fmt.Println(i)

		port++
		if !sliceContains(takenPort, port) {
			fmt.Println(i)
			takenPort = append(takenPort, port)

			dir := fmt.Sprintf("%s/bin/server", GetGoPath())
			args := []string{"run", "-Simport=8080", "-host=localhost", "-port=:" + strconv.Itoa(port)}

			cmnd := exec.Command(dir, args...)

			//cmnd := exec.Command("go", "run", "cmd/server/server.go", "run", "-Simport=8080", "-host=localhost", "-port=:"+sPort)
			errorMsg("Command error: ", err)

			//var out bytes.Buffer
			var stderr bytes.Buffer
			cmnd.Stdout = os.Stdout
			cmnd.Stderr = os.Stderr
			cmnd.Stdin = os.Stdin

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
			} /*else {
				fmt.Println("Result: " + out.String())
			}*/
		} else {
			fmt.Printf("Port is taken... Try again or something.\n")
		}
	}
	/*Need this so the program doesn't quit before the OU are done..*/
	time.Sleep(1800 * time.Second)
	printSlice(takenPort)
}

/*GetGoPath gets gopath*/
func GetGoPath() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	return gopath
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
	return rand.Intn(max-min) + min
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}
