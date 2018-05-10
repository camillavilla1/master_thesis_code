package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

/*ErrMsg returns error if any*/
func ErrMsg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

/*AppendFile3 appends to a file*/
func AppendFile3(path string, slice []string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	ErrMsg("Append to log: ", err)
	defer f.Close()

	//err = writer.Write(slice)
	//ErrMsg("Error write to log: ", err)
	//writer.Flush()
	line := strings.Join(slice, "\t")
	line = strings.Replace(line, "\n\t", "\n", -1)
	test(f, []string{line})
}

func test(file *os.File, data []string) {

	for _, v := range data {
		if _, err := fmt.Fprintf(file, v); err != nil {
			ErrMsg("Error: ", err)
		}

	}
}

//ListContains2 check if a value is in a list/slice and return true or false
func ListContains2(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

/*ReadCsv2 reads csv*/
func ReadCsv2() {
	//Time	Pid	Memory percent	CPU percent	Process running	Process connections	Sends	Sends to leader	Cluster Head count	Received data pkt

	fmt.Printf("READING CSV/LOG..\n")

	record := []string{}
	timeSlice := []string{}
	pidSlice := []string{}
	//memSlice := []string{}

	//headerSlice := []string{}

	memMap := make(map[string][]string)

	file, err := os.Open("experiments2.log")
	//file, err := os.Open("./cmd/server/results/experiments2.log")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	lineCount := 0

	for {
		// read just one record, but we could ReadAll() as well
		record, err = reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Error:", err)
			return
		}
		// record is an array of string so is directly printable
		fmt.Println("Record", lineCount, "is", record, "and has", len(record), "fields")
		// and we can iterate on top of that

		/*for i := 0; i < len(record); i++ {
			fmt.Println(" ", record[i])
		}
		fmt.Println()*/

		if !ListContains2(timeSlice, record[0]) {
			timeSlice = append(timeSlice, record[0])
		}

		//headerSlice = timeSlice

		//	fmt.Printf("HEADERSLICE: %+v\n", headerSlice)

		//pidSlice = append(pidSlice, record[1])
		if !ListContains2(pidSlice, record[1]) {
			pidSlice = append(pidSlice, record[1])
		}
		lineCount++

		fmt.Printf("MemMap %+v\n", memMap)

		memMap[record[1]] = append(memMap[record[1]], record[2])
	}

	for k, v := range memMap {
		fmt.Printf("KEY: %s, Value: %s\n", k, v)
	}

	memSlice := converteMap(memMap)

	fmt.Printf("MemSlice: %+v\n", memSlice)

	measureMemory(memSlice)

	//	fmt.Printf("ROWSLICE: %s\n", rowSlice)
}

func main() {
	ReadCsv2()
}

func converteMap(m map[string][]string) []string {
	pairs := []string{}
	for key, val := range m {
		pairs = append(pairs, key)
		for _, v := range val {
			pairs = append(pairs, v)

		}
		pairs = append(pairs, "\n")

	}

	return pairs
}

func measureMemory(s []string) {
	path := "experiments3new.log"
	//f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	//ErrMsg("Open file log: ", err)
	//defer f.Close()

	AppendFile3(path, s)
}
