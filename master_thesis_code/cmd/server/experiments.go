package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

/*Experiments measures memory, cpu etc on the system*/
func Experiments() {
	tickChan := time.NewTicker(time.Second * 10).C
	folder := "./cmd/server/results"

	memSlice := []string{}
	cpuSlice := []string{}
	procSlice := []string{}

	fmem, err := os.OpenFile(folder+"/mem.log", os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("Memory log: ", err)
	defer fmem.Close()

	fcpu, err := os.OpenFile(folder+"/cpu.log", os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("CPU log: ", err)
	defer fcpu.Close()

	fproc, err := os.OpenFile(folder+"/proc.log", os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("PROC log: ", err)
	defer fproc.Close()

	memWriter := csv.NewWriter(fmem)
	cpuWriter := csv.NewWriter(fcpu)
	procWriter := csv.NewWriter(fproc)

	memSlice = append(memSlice, "UsedPercent")
	memSlice = append(memSlice, "UsedMemory")
	err = memWriter.Write(memSlice)
	ErrorMsg("Error write mem: ", err)
	memWriter.Flush()
	memSlice = []string{}

	cpuSlice = append(cpuSlice, "CPUPercentage")
	err = cpuWriter.Write(cpuSlice)
	ErrorMsg("Error write CPU: ", err)
	cpuWriter.Flush()
	cpuSlice = []string{}

	procSlice = append(procSlice, "NumProc")
	err = procWriter.Write(procSlice)
	ErrorMsg("Error write proc: ", err)
	procWriter.Flush()
	procSlice = []string{}

	doneChan := make(chan bool)
	go func() {
		time.Sleep(time.Second * time.Duration(batteryStart))
		doneChan <- true
	}()

	for {
		select {
		case <-tickChan:
			memWriter := csv.NewWriter(fmem)
			cpuWriter := csv.NewWriter(fcpu)

			//MEMORY!!!!
			mem, _ := mem.VirtualMemory()
			//fmt.Printf("Total: %v, Free:%v, UsedPercent:%f%%\n", mem.Total, mem.Free, mem.UsedPercent)

			memUsedPercentage := strconv.FormatFloat(mem.UsedPercent, 'g', -1, 64)
			memUsed := strconv.FormatUint(mem.Used, 10)
			//fmt.Fprintln(w, "%f\t.", mem.UsedPercent)
			memSlice = append(memSlice, memUsedPercentage)
			memSlice = append(memSlice, memUsed)

			err = memWriter.Write(memSlice)
			ErrorMsg("Error write mem: ", err)
			memWriter.Flush()
			memSlice = []string{}
			//MEMORY END

			//CPU!!
			oneCPUPercentage, _ := cpu.Percent(0, false)
			newOneCPUPercentage := strconv.FormatFloat(oneCPUPercentage[0], 'g', -1, 64)
			cpuSlice = append(cpuSlice, newOneCPUPercentage)

			err = cpuWriter.Write(cpuSlice)
			ErrorMsg("Error write CPU: ", err)
			cpuWriter.Flush()
			cpuSlice = []string{}
			//CPU END

			//DISK!!!
			//Do we need to measure disk-utilites??
			//DISK END

			//NET!!!
			//connections, _ := net.Connections()
			//NET END

			//PROCESS !!!
			//Number of processes running?
			procConnections, _ := process.Processes()
			fmt.Printf("procConnections: %d\n", len(procConnections))

			numProc := strconv.Itoa(len(procConnections))
			procSlice = append(procSlice, numProc)

			err = procWriter.Write(procSlice)
			ErrorMsg("Error write proc: ", err)
			procWriter.Flush()
			procSlice = []string{}
			//PROC END
		}
	}

	//DISK
	//diskStat, _ := disk.Usage("/")
	//diskCount, _ := disk.IOCounters()

	//diskRead, _ := disk.IOCountersStat()
	//diskWrite, _ := disk.IOCountersStat()

	//diskReadBytes, _ := disk.IOCountersStat.ReadBytes()
	//diskWriteBytes, _ := disk.IOCountersStat.WriteBytes()

	//fmt.Printf("DISKstat: %+v\n", diskStat)
	//fmt.Printf("DISK count: %+v\n", diskCount)
}
