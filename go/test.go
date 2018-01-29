package main

import(
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
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

func find_free_ip() {
	
}
/*


type Server struct {
	ListenHost string
	ListenPort string
	Logger *log.Logger
}

func NewHTTPServer(listenIP, httpPort string) *Server {
	res := Server{
		ListenHost: listenIP
		ListenPort: httpPort
		Logger: log.New(os.Stdout, "server> ", log.Ltime.|log.Lshortfile)}

	http.HandleFunc("/", res.HandleIndex)

	reutnr &res
}


func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	defer timeTrack(time.Now(), w, "HandeIndex")

	s.httpHeaders(w)
	io.WriteString(w, "hello, world")
}

*/

/*func spawn(port int) {
	port += 1
    fmt.Println(port)
	
	newPort := strconv.Itoa(port)
	//fmt.Println(newPort)

	address := "localhost:" + newPort

	ln, err := net.Listen("tcp", address)
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

    fmt.Printf("\nClosed connection!\n")


}

func main() {
	port := 8080
	for i := 1; i < 3; i++ {
		fmt.Printf("Spawing process\n")
		go spawn(port)
		fmt.Printf("Finished\n")

	}
}*/

func main() {
	fmt.Printf("Launching server...\n")
    port := 8080
    //Listen on all interfaces
    for i := 1; i < 3; i++ {
    	port += 1
    	fmt.Println(port)

    	newPort := strconv.Itoa(port)
    	//fmt.Println(newPort)

    	address := "localhost:" + newPort

    	ln, err := net.Listen("tcp", address)
    	error_msg("Listen: ", err)
    	defer ln.Close()

    	hostname, err = os.Hostname()
    	error_msg("Hostname: ", err)

    
	    hostaddress = strings.Split(hostname, ".")[0]
	    fmt.Printf("Hostname is %s.\n", hostname)
	    //Connection to port
	    conn, err := ln.Accept()
	    error_msg("Connection: ", err)
	    defer conn.Close()

	    //go handle(conn) to make it concurrent..
	    handle_conn(conn) //and then handler..

	    fmt.Printf("\nClosed connection!\n")


    }
}


/*func main() {
    fmt.Printf("Launching server...\n")
    var port int = 8080

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
}*/

func handle_conn(c net.Conn) {
	fmt.Printf("\nHandle connection")
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

