package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

/*Experiments2 measures memory, cpu etc on the system*/
func (ou *ObservationUnit) Experiments2(pid int) {
	tickChan := time.NewTicker(time.Millisecond * 500).C
	folder := "./cmd/server/results"

	path := folder + "/experiments2.csv"

	infoSlice := []string{}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("Open file log: ", err)
	defer f.Close()

	writer := csv.NewWriter(f)
	writer.Comma = '\t'

	doneChan := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(batteryStart))
		doneChan <- true
	}()

	for {
		select {
		case <-tickChan:
			//TIME!!!
			t := strconv.FormatInt(time.Now().Unix(), 10)
			infoSlice = append(infoSlice, t)
			//TIME END

			//PID!!
			infoSlice = append(infoSlice, strconv.Itoa(pid))
			//PID END

			//MEMORY!!!!
			mem, _ := mem.VirtualMemory()
			//fmt.Printf("Total: %v, Free:%v, UsedPercent:%f%%\n", mem.Total, mem.Free, mem.UsedPercent)

			memUsedPercentage2 := toFixed(mem.UsedPercent, 3)
			memUsedPercentage := strconv.FormatFloat(memUsedPercentage2, 'g', -1, 64)

			//totMem := strconv.FormatUint(mem.Total, 10)

			//memUsed := strconv.FormatUint(mem.Used, 10)
			//fmt.Fprintln(w, "%f\t.", mem.UsedPercent)
			infoSlice = append(infoSlice, memUsedPercentage)
			//infoSlice = append(infoSlice, totMem)
			//infoSlice = append(infoSlice, memUsed)
			//MEMORY END

			//CPU!!
			oneCPUPercentage, _ := cpu.Percent(0, false)
			oneCPUPercentage[0] = toFixed(oneCPUPercentage[0], 3)
			newOneCPUPercentage := strconv.FormatFloat(oneCPUPercentage[0], 'g', -1, 64)

			infoSlice = append(infoSlice, newOneCPUPercentage)
			//CPU END

			//PROCEESS!!!
			//Number of processes running?
			procConnections, _ := process.Processes()
			//func (*Process) CPUPercent
			//func (*Process) MemoryPercent

			numProc := strconv.Itoa(len(procConnections))
			infoSlice = append(infoSlice, numProc)
			//PROC END

			//NET!!!
			connections, err := net.ConnectionsPid("all", int32(pid))
			ErrorMsg("con: ", err)
			//fmt.Printf("\nNET Connections: %v\n", connections)
			//fmt.Printf("LEN CONN: %d\n", len(connections))
			infoSlice = append(infoSlice, strconv.Itoa(len(connections)))
			//NET END

			infoSlice = append(infoSlice, strconv.Itoa(ou.Sends))
			infoSlice = append(infoSlice, strconv.Itoa(ou.SendsToLeader))

			infoSlice = append(infoSlice, strconv.Itoa(ou.ClusterHeadCount))

			infoSlice = append(infoSlice, strconv.Itoa(ou.ReceivedDataPkt))

			AppendFile2(path, writer, infoSlice)
			infoSlice = []string{}

		}
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

func readContentOfFolder(index int, limit int) []string {
	var rows []string

	folder := "./cmd/server/results"
	path := folder + "/experiments2.csv"

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
func readMemory() {

	folder := "./cmd/server/results/result"
	path := folder + "/mem_result2.log"

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	ErrorMsg("Open file log: ", err)
	defer f.Close()

	memSlice := []string{}
	writer := csv.NewWriter(f)
	writer.Comma = '\t'

	headerTime := readContentOfFolder(0, 10)
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
