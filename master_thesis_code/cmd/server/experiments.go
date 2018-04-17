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
func Experiments(pid int, hostAddress string) {
	tickChan := time.NewTicker(time.Millisecond * 100).C
	folder := "./cmd/server/results"

	memPath := folder + "/mem.log"
	cpuPath := folder + "/cpu.log"
	procPath := folder + "/proc.log"
	//netPath := folder + "/net.log"

	//Delete files if exists
	//deleteFile(memPath)
	//deleteFile(cpuPath)
	//deleteFile(procPath)

	//Create new files if not exist
	//createFile(memPath)
	//createFile(cpuPath)
	//createFile(procPath)

	memSlice := []string{}
	cpuSlice := []string{}
	procSlice := []string{}

	fmem, err := os.OpenFile(memPath, os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("Memory log: ", err)
	defer fmem.Close()

	fcpu, err := os.OpenFile(cpuPath, os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("CPU log: ", err)
	defer fcpu.Close()

	fproc, err := os.OpenFile(procPath, os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("PROC log: ", err)
	defer fproc.Close()

	memWriter := csv.NewWriter(fmem)
	memWriter.Comma = '\t'
	cpuWriter := csv.NewWriter(fcpu)
	cpuWriter.Comma = '\t'
	procWriter := csv.NewWriter(fproc)
	procWriter.Comma = '\t'

	memSlice = append(memSlice, "HostAddress")
	memSlice = append(memSlice, "UsedPercent")
	memSlice = append(memSlice, "UsedMemory")
	appendFile(memPath, memWriter, memSlice)
	memSlice = []string{}

	cpuSlice = append(cpuSlice, "CPUPercentage")
	appendFile(cpuPath, cpuWriter, cpuSlice)
	cpuSlice = []string{}

	procSlice = append(procSlice, "NumProc")
	//err = procWriter.Write(procSlice)
	//ErrorMsg("Error write proc: ", err)
	//procWriter.Flush()
	appendFile(procPath, procWriter, procSlice)
	procSlice = []string{}

	doneChan := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(batteryStart))
		doneChan <- true
	}()

	for {
		select {
		case <-tickChan:
			//MEMORY!!!!
			mem, _ := mem.VirtualMemory()
			//fmt.Printf("Total: %v, Free:%v, UsedPercent:%f%%\n", mem.Total, mem.Free, mem.UsedPercent)

			memUsedPercentage := strconv.FormatFloat(mem.UsedPercent, 'g', -1, 64)
			memUsed := strconv.FormatUint(mem.Used, 10)
			//fmt.Fprintln(w, "%f\t.", mem.UsedPercent)
			memSlice = append(memSlice, hostAddress)
			memSlice = append(memSlice, memUsedPercentage)
			memSlice = append(memSlice, memUsed)

			appendFile(memPath, memWriter, memSlice)
			memSlice = []string{}
			//MEMORY END

			//CPU!!
			oneCPUPercentage, _ := cpu.Percent(0, false)
			newOneCPUPercentage := strconv.FormatFloat(oneCPUPercentage[0], 'g', -1, 64)
			cpuSlice = append(cpuSlice, newOneCPUPercentage)

			appendFile(cpuPath, cpuWriter, cpuSlice)
			cpuSlice = []string{}
			//CPU END

			//DISK!!!
			//Do we need to measure disk-utilites??
			//DISK END

			//NET!!!
			// inet, inet4, inet6, tcp, tcp4, tcp6, udp, udp4, udp6, unix, all
			//fmt.Printf("PID: %d\n", int32(pid))
			//connection, err := net.Connections("all")
			//ErrorMsg("connection: ", err)
			/*for k, v := range connection {
				fmt.Printf("Connection: %d: %+v\n", k, v)
				if v.Pid == int32(pid) {
					fmt.Printf("\n\n SIMILAR!!!\n\n\n")
					time.Sleep(time.Second * 5)
				}
			}*/

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
			procSlice = append(procSlice, numProc)

			appendFile(procPath, procWriter, procSlice)
			procSlice = []string{}
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
