package main

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

/*Experiments measures memory, cpu etc on the system*/
func Experiments(pid int) {
	tickChan := time.NewTicker(time.Millisecond * 100).C
	folder := "./cmd/server/results"

	path := folder + "/experience.csv"

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

			totMem := strconv.FormatUint(mem.Total, 10)

			memUsed := strconv.FormatUint(mem.Used, 10)
			//fmt.Fprintln(w, "%f\t.", mem.UsedPercent)
			infoSlice = append(infoSlice, memUsedPercentage)
			infoSlice = append(infoSlice, totMem)
			infoSlice = append(infoSlice, memUsed)
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

			//infoSlice = append(infoSlice, strconv.Itoa(sends))
			//infoSlice = append(infoSlice, strconv.Itoa(sendsToLeader))

			appendFile(path, writer, infoSlice)
			infoSlice = []string{}

		}
	}
}

func appendFile(path string, writer *csv.Writer, slice []string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("Append to log: ", err)
	defer f.Close()

	err = writer.Write(slice)
	ErrorMsg("Error write to log: ", err)
	writer.Flush()
}
