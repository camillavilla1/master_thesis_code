package main

import (
	"encoding/csv"
	"fmt"
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

	path := folder + "/experience.log"

	//Delete files if exists
	//deleteFile(memPath)

	//Create new files if not exist
	//createFile(memPath)

	infoSlice := []string{}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("Open file log: ", err)
	defer f.Close()

	writer := csv.NewWriter(f)
	writer.Comma = '\t'

	//infoSlice = append(infoSlice, "HostAddress")
	//infoSlice = append(infoSlice, "UsedPercent")
	//infoSlice = append(infoSlice, "UsedMemory")
	//appendFile(path, writer, infoSlice)
	//infoSlice = []string{}

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

			//MEMORY!!!!
			mem, _ := mem.VirtualMemory()
			//fmt.Printf("Total: %v, Free:%v, UsedPercent:%f%%\n", mem.Total, mem.Free, mem.UsedPercent)

			memUsedPercentage := strconv.FormatFloat(mem.UsedPercent, 'g', -1, 64)
			memUsed := strconv.FormatUint(mem.Used, 10)
			//fmt.Fprintln(w, "%f\t.", mem.UsedPercent)
			infoSlice = append(infoSlice, strconv.Itoa(pid))
			infoSlice = append(infoSlice, memUsedPercentage)
			infoSlice = append(infoSlice, memUsed)

			//appendFile(path, writer, infoSlice)
			//infoSlice = []string{}
			//MEMORY END

			//CPU!!
			oneCPUPercentage, _ := cpu.Percent(0, false)
			newOneCPUPercentage := strconv.FormatFloat(oneCPUPercentage[0], 'g', -1, 64)
			infoSlice = append(infoSlice, newOneCPUPercentage)

			//appendFile(path, writer, infoSlice)
			//infoSlice = []string{}
			//CPU END

			connections, err := net.ConnectionsPid("all", int32(pid))
			ErrorMsg("con: ", err)
			//fmt.Printf("\nNET Connections: %v\n", connections)
			fmt.Printf("LEN CONN: %d\n", len(connections))
			for k, v := range connections {
				fmt.Printf("CONN: %d = %s\n", k, v)
			}
			//NET END

			//PROCESS !!!
			//Number of processes running?
			procConnections, _ := process.Processes()
			//fmt.Printf("procConnections: %d\n", len(procConnections))

			numProc := strconv.Itoa(len(procConnections))
			infoSlice = append(infoSlice, numProc)

			appendFile(path, writer, infoSlice)
			infoSlice = []string{}
			//PROC END
		}
	}
}

func createFile(path string) {
	// detect if file exists
	var _, err = os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		ErrorMsg("Create file: ", err)
		defer file.Close()
	}

	fmt.Println("==> Done creating file", path)
}

func deleteFile(path string) {
	var _, err = os.Stat(path)

	// create file if not exists
	if !os.IsNotExist(err) {
		// delete file
		var err = os.Remove(path)
		ErrorMsg("Delete file: ", err)
	}
	fmt.Println("==> Done deleting file")
}

func appendFile(path string, writer *csv.Writer, slice []string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("Append to log: ", err)
	defer f.Close()

	err = writer.Write(slice)
	ErrorMsg("Error write to log: ", err)
	writer.Flush()
	//slice = []string{}
}
