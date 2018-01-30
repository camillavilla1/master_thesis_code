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

var port_taken []int

var available_port []int

/*func main() {

	numServers := os.Args[1]
	fmt.Println(numServers)
	numServers2, err := strconv.Atoi(numServers)
	error_msg("Str to int: ", err)

	for i := 0; i < numServers2; i++ {
		fmt.Println(i)
		port := random_port()
		s_port := strconv.Itoa(port)


		number := make(chan int)
		go random_port2(number)
		result := <-number //receive from number

		fmt.Println(result)
		printSlice(port_taken)


		cmnd := exec.Command("go", "run", "server.go", "run", "-SOUport=8080 -localhost=localhost -port=:"+s_port)		

		//fmt.Println(cmnd)

	    err := cmnd.Start()
	    error_msg("Start process: ", err)
	}
}

func random_port2(c chan int) {
	number := random(8081, 9090)
	//fmt.Println(number)

	if !sliceContains(port_taken, number) {
		port_taken = append(port_taken, number)
		//printSlice(port_taken)	
	} else {
		number = random_port()
	}
	c <- number //send number to c
}*/

func main() {

	numThreads := runtime.NumCPU()
    runtime.GOMAXPROCS(numThreads)

	numServers := os.Args[1]
	fmt.Println(numServers)
	numServers2, err := strconv.Atoi(numServers)
	error_msg("Str to int: ", err)

	numPort := make(chan int)
	go find_port(numServers2, numPort)
	result := <- numPort //receive from numport
	fmt.Println(result)
	printSlice(available_port)

	for i := 0; i < numServers2; i++ {
		//fmt.Println(i)
		port := random_port()
		s_port := strconv.Itoa(port)
		//port := 8081
		//port += 1
		//s_port := strconv.Itoa(port)
		//add port to taken_ports..

		cmnd := exec.Command("go", "run", "server.go", "run", "-SOUport=8080 -localhost=localhost -port=:"+s_port)		

		//fmt.Println(cmnd)

	    err := cmnd.Start()
	    error_msg("Start process: ", err)
	}
	//printSlice(port_taken)
}


func find_port(num int, c chan int) {
	number := 0
	for i := 0; i < num; i++ {
		number = random2(8081, 9090) 
		available_port = append(available_port, number)
	}

	c <- number
}


func random2(min, max int) int {
    rand.Seed(time.Now().Unix())
    return rand.Intn(max - min) + min
}
/*
func main() {

	numServers := os.Args[1]
	fmt.Println(numServers)
	numServers2, err := strconv.Atoi(numServers)
	error_msg("Str to int: ", err)

	for i := 0; i < numServers2; i++ {
		//fmt.Println(i)
		port := random_port()
		s_port := strconv.Itoa(port)

		cmnd := exec.Command("go", "run", "server.go", "run", "-SOUport=8080 -localhost=localhost -port=:"+s_port)		

		//fmt.Println(cmnd)

	    err := cmnd.Start()
	    error_msg("Start process: ", err)
	}
	printSlice(port_taken)
}*/

/*func main() {
	port := random_port()
	s_port := strconv.Itoa(port)
	cmnd := exec.Command("go", "run", "server.go", "run", "-SOUport=8080 -localhost=localhost -port=:"+s_port)

	//fmt.Println(cmnd)

    err := cmnd.Start()
    error_msg("Start process: ", err)
	
	//cmndIn, _ := cmnd.StdinPipe()
    //cmndOut, _ := cmnd.StdoutPipe()
    //fmt.Println(cmndIn)
    //fmt.Println(cmndOut)
}*/

func error_msg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
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

func random_port() (number int) {
	number = random(8081, 9090)
	//fmt.Println(number)

	if !sliceContains(port_taken, number) {
		port_taken = append(port_taken, number)
	} else {
		number = random_port()
	}

	return
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}