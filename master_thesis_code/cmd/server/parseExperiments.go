package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

func readContentOfExperiments(index int, limit int) []string {
	var rows []string

	folder := "./cmd/server/results"
	path := folder + "/experiments.log"

	var line string

	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	errMsg("Error open file:", err)

	scanner := bufio.NewScanner(file)

	// Read line by line
	var lmt = 0
	for scanner.Scan() {
		// Skip first line
		//if lmt == 0 {
		//	lmt++
		//	continue
		//	}
		readLine := strings.Split(scanner.Text(), "\t")
		//fmt.Printf("(%d - %d) - %v\n", index, len(readLine), readLine)

		line += fmt.Sprintf("%s\t", readLine[index])
		//line = append(line, readLine[index])
		lmt++
		// Only do to limit
		if lmt == limit {
			break
		}
	}
	err = scanner.Err()
	if err != nil {
		panic(err)
	}
	file.Close()
	line += fmt.Sprintf("\n")
	// Append to the rows
	rows = append(rows, line)
	//line = ""
	//}
	return rows
}

/*ReadMemory reads memory values*/
func readMemory3() {
	splitted := []string{}
	folder := "./cmd/server/results/result"
	path := folder + "/mem_result.log"

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	ErrorMsg("Open file log: ", err)
	defer f.Close()

	memSlice := []string{}
	writer := csv.NewWriter(f)
	writer.Comma = '\t'

	headerTime := readContentOfExperiments(0, 10)
	//rows := readContentOfFolder(2, 10)

	//Gets list of 1, need to split on tab..
	for _, v := range headerTime {
		splitted = strings.Split(v, "\t")
	}

	for _, val := range splitted {
		//fmt.Printf("SPLITTED V= %s\n", val)

		if !ListContains(memSlice, "Time/Pid") {
			memSlice = append(memSlice, "Time/Pid")
			//memSlice = append([]string{"Time/Pid"}, memSlice...)
		}

		if !ListContains(memSlice, val) {
			memSlice = append(memSlice, val)
		}
	}

	//remove last elemet since its ""..
	if len(memSlice) > 0 {
		memSlice = memSlice[:len(memSlice)-1]
	}

	/*for _, val := range memSlice {
		fmt.Printf("MMMELSIcE V= %s\n", val)
	}
	*/
	AppendFile(path, writer, memSlice)
}

/*errMsg returns error if any*/
func errMsg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

/*ConvertExperiments etc*/
func (ou *ObservationUnit) ConvertExperiments() {
	time.Sleep(time.Second * 5)
	wg = sync.WaitGroup{}
	wg.Add(1)
	defer wg.Done()
	readMemory3()

}
