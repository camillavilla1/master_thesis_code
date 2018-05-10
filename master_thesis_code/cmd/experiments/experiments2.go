package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

/*CsvLine from csv*/
type CsvLine struct {
	Time          string
	Pid           string
	MemoryPercent string
}

/*ErrorMsg returns error if any*/
func ErrorMsg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

/*AppendFile2 appends to a file*/
func AppendFile2(path string, writer *csv.Writer, slice []string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("Append to log: ", err)
	defer f.Close()

	err = writer.Write(slice)
	ErrorMsg("Error write to log: ", err)
	writer.Flush()
}

//ListContains check if a value is in a list/slice and return true or false
func ListContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

/*ReadCsv reads csv*/
func ReadCsv() {
	time.Sleep(time.Second * 5)
	//Time	Pid	Memory percent	CPU percent	Process running	Process connections	Sends	Sends to leader	Cluster Head count	Received data pkt

	fmt.Printf("READING CSV/LOG..\n")
	record := []string{}
	timeSlice := []string{}
	pidSlice := []string{}
	memSlice := []string{}

	headerSlice := []string{}
	rowSlice := []string{}
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

		if !ListContains(timeSlice, record[0]) {
			timeSlice = append(timeSlice, record[0])
		}

		headerSlice = timeSlice

		fmt.Printf("HEADERSLICE: %+v\n", headerSlice)

		//pidSlice = append(pidSlice, record[1])
		if !ListContains(pidSlice, record[1]) {
			pidSlice = append(pidSlice, record[1])
		}
		//fmt.Printf("PIDSLICE: %+v\n", pidSlice)

		memSlice = append(memSlice, record[2])
		//fmt.Printf("MEMSLICE: %+v\n", memSlice)

		lineCount++

		fmt.Printf("Rowslice %+v\n", rowSlice)

		//if rowSlice[0] == record[1] {
		//	rowSlice = append(rowSlice, record[2])
		//	fmt.Printf("Appended to rowslice %s\n\n", record[2])
		//}

		for i := 0; i < len(record); i++ {
			//fmt.Println(" ", record)

			if !ListContains(rowSlice, record[1]) {
				rowSlice = append(rowSlice, record[1])

				if rowSlice[0] == record[1] {
					rowSlice = append(rowSlice, record[2])
					fmt.Printf("Appended to rowslice %s\n\n", record[2])
				}
			}

		}

		fmt.Printf("ROWSLICE: %s\n", rowSlice)
	}
}

//ListContains check if a value is in a list/slice and return true or false
func d2ListContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

/*func main() {
	ReadCsv()
}*/
