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

/*AppendToFile appends to a file*/
func AppendToFile(path string, slice []string) {
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
	//Time	Pid	Memorypercent	CPUpercent	ProcessRunning	ProcessConnections	Sends	Sendstoleader	ClusterHeadcount Received data pkt

	fmt.Printf("READING CSV/LOG..\n")

	record := []string{}
	pidSlice := []string{}

	headerTimeSlice := []string{}

	memMap := make(map[string][]string)
	cpuMap := make(map[string][]string)
	numConnMap := make(map[string][]string)
	numSendsMap := make(map[string][]string)
	numSendsChMap := make(map[string][]string)
	chCountMap := make(map[string][]string)

	file, err := os.Open("experiments.log")
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
		//fmt.Println("Record", lineCount, "is", record, "and has", len(record), "fields")

		if !ListContains2(headerTimeSlice, record[0]) {
			headerTimeSlice = append(headerTimeSlice, record[0])
		}

		if !ListContains2(pidSlice, record[1]) {
			pidSlice = append(pidSlice, record[1])
		}

		lineCount++

		memMap[record[1]] = append(memMap[record[1]], record[2])
		cpuMap[record[1]] = append(cpuMap[record[1]], record[3])
		numConnMap[record[1]] = append(numConnMap[record[1]], record[5])

		numSendsMap[record[1]] = append(numSendsMap[record[1]], record[6])
		numSendsChMap[record[1]] = append(numSendsChMap[record[1]], record[7])
		chCountMap[record[1]] = append(chCountMap[record[1]], record[8])
	}

	/*for k, v := range memMap {
		fmt.Printf("KEY: %s, Value: %s\n", k, v)
	}*/

	memSlice := converteMap(memMap)
	cpuSlice := converteMap(cpuMap)
	numConnSlice := converteMap(numConnMap)
	numSendsSlice := converteMap(numSendsMap)
	numSendsChSlice := converteMap(numSendsChMap)
	chCountSlice := converteMap(chCountMap)

	//fmt.Printf("MemSlice: %+v\n", memSlice)

	//retAverage := average(cpuSlice)
	//fmt.Printf("AVERAGE: %f\n", retAverage)

	//getDataFromFile(memMap, pidSlice, record, 1, 2)

	AppendToFile("memResults.log", memSlice)
	AppendToFile("CPUResults.log", cpuSlice)
	AppendToFile("numConnResults.log", numConnSlice)
	AppendToFile("numSendsResults.log", numSendsSlice)
	AppendToFile("numSendsChResults.log", numSendsChSlice)
	AppendToFile("chCountResults.log", chCountSlice)
}

func main() {
	ReadCsv2()
}

func average(xs []string) float64 {
	total := 0.0
	var val1 float64
	fmt.Printf("XS/CPUSlice: %+v\n", xs)
	for _, v := range xs {
		for _, val := range v {
			val1 = float64(val)
		}

		total += val1
	}
	return total / float64(len(xs))
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
