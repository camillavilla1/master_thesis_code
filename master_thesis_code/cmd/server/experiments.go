package main

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

/*Experiments measures memory, cpu etc on the system*/
func Experiments() {
	tickChan := time.NewTicker(time.Second * 10).C
	folder := "./cmd/server/results"

	memSlice := []string{}
	cpuSlice := []string{}

	fmem, err := os.OpenFile(folder+"/mem.log", os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("Memory log: ", err)
	defer fmem.Close()

	fcpu, err := os.OpenFile(folder+"/cpu.log", os.O_APPEND|os.O_WRONLY, 0600)
	ErrorMsg("Memory log: ", err)
	defer fcpu.Close()

	memWriter := csv.NewWriter(fmem)
	cpuWriter := csv.NewWriter(fcpu)

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

			/*cpuPercentage, _ := cpu.Percent(0, true)
			for i := range cpuPercentage {
				newCPUPercentage = append(newCPUPercentage, string(i))
			}
			err = cpuWriter.Write(newCPUPercentage)
			ErrorMsg("Error write mem: ", err)
			cpuWriter.Flush()
			*/
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

	//NET

	//PROCESS
	//Number of processes running?

}

func runesToString(runes []rune) (outString string) {
	// don't need index so _
	for _, v := range runes {
		outString += string(v)
	}
	return
}
