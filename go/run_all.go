package main 

import (
	"os/exec"
	"fmt"
	"math/rand"
	"time"
	"strconv"
	"log"
)

var port_taken []int

func main() {
	port := random_port()
	s_port := strconv.Itoa(port)
	cmnd := exec.Command("go", "run", "script.go", "-localhost=localhost -port="+s_port)

	//fmt.Println(cmnd)

    err := cmnd.Start()
    error_msg("Start process: ", err)
	
	//cmndIn, _ := cmnd.StdinPipe()
    //cmndOut, _ := cmnd.StdoutPipe()
    //fmt.Println(cmndIn)
    //fmt.Println(cmndOut)
}

func error_msg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}


func random(min, max int) int {
    rand.Seed(time.Now().Unix())
    return rand.Intn(max - min) + min
}

func random_port() (number int) {
	number = random(8081, 9090)
	fmt.Println(number)
	port_taken = append(port_taken, number)
	printSlice(port_taken)
	return
}

func printSlice(s []int) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}