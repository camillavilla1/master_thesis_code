package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

func readContentOfFolder3(index int, limit int) []string {
	var rows []string

	folder := "./cmd/server/results"
	path := folder + "/experiments.csv"

	var line string

	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	errMsg("Error open file:", err)

	scanner := bufio.NewScanner(file)

	// Read line by line
	var lmt = 0
	for scanner.Scan() {
		// Skip first line
		if lmt == 0 {
			lmt++
			continue
		}
		readLine := strings.Split(scanner.Text(), "\t")
		fmt.Printf("(%d - %d) - %v\n", index, len(readLine), readLine)

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
	line = ""
	//}
	return rows
}

/*ReadMemory reads memory values*/
func readMemory3() {

	folder := "./cmd/server/results/result"
	path := folder + "/mem_result.log"

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	ErrorMsg("Open file log: ", err)
	defer f.Close()

	memSlice := []string{}
	writer := csv.NewWriter(f)
	writer.Comma = '\t'

	headerTime := readContentOfFolder3(0, 10)
	//rows := readContentOfFolder(2, 10)
	//fmt.Printf("rows: %+v\n", rows)
	//fmt.Printf("headerTime: %+v", headerTime)
	//fmt.Printf("headerTime: %s\n", reflect.TypeOf(headerTime))

	for _, v := range headerTime {
		splitted := strings.Split(v, "\t")
		fmt.Printf("splitted: %s", splitted)
		memSlice = append(memSlice, v)
	}

	//memSlice = append(memSlice[:1], memSlice[1:]...)

	AppendFile(path, writer, memSlice)
}

/*errMsg returns error if any*/
func errMsg(s string, err error) {
	if err != nil {
		log.Fatal(s, err)
	}
}

/*FixEverything etc*/
func FixEverything() {
	wg = sync.WaitGroup{}
	wg.Add(1)
	defer wg.Done()
	readMemory3()
}
