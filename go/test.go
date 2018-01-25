package main

import(
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

var startedObservationUnits []string
var hostname string
var hostaddress string


func error_msg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}


func main() {
    fmt.Printf("Launching server...\n")

    //Listen on all interfaces
    ln, err := net.Listen("tcp", "localhost:8081")
    error_msg("Listen: ", err)
    defer ln.Close()

    hostname, err = os.Hostname()
    error_msg("Hostname: ", err)

    hostaddress = strings.Split(hostname, ".")[0]
    fmt.Printf("Hostname is %s", hostname)
    //Connection to port
    conn, err := ln.Accept()
    error_msg("Connection: ", err)
    defer conn.Close()

    //go handle(conn) to make it concurrent..
    handle_conn(conn) //and then handler..

    //conn.Close()
    //ln.Close()
    fmt.Printf("\nClosed connection!\n")
}

func handle_conn(c net.Conn) {
	fmt.Printf("Handle connection")
	//packet := new(Pack)
	//w := make(chan *Pack, 100)
	buf := make([]byte, 1024)
	_, err := c.Read(buf)
	error_msg("Request: ", err)

	c.Write([]byte("Message received.\n"))

}

/*
func handler() {
	fmt.Printf("Handler..")
	//func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
	//http.HandleFunc("/xxx", xxxHandler)
}

func xxxHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("One handler..")
}
*/

