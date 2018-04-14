package main

import (
	"fmt"
	"os"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

/*Experiments measures memory, cpu etc on the system*/
func Experiments() {
	//MEMORY!!!!
	mem, _ := mem.VirtualMemory()

	// almost every return value is a struct
	fmt.Printf("Total: %v, Free:%v, UsedPercent:%f%%\n", mem.Total, mem.Free, mem.UsedPercent)

	// convert to JSON. String() is also implemented
	//fmt.Println(v)

	//CPU!!
	cpuRet, _ := cpu.Info()
	percentage, _ := cpu.Percent(0, true)
	//cpu per process?

	fmt.Printf("CPU: %+v\n", cpuRet)
	fmt.Printf("CPU percentage: %f \n", percentage)
	writeToFile(percentage[0])

	//DISK
	diskStat, _ := disk.Usage("/")
	diskCount, _ := disk.IOCounters()

	//diskRead, _ := disk.IOCountersStat()
	//diskWrite, _ := disk.IOCountersStat()

	//diskReadBytes, _ := disk.IOCountersStat.ReadBytes()
	//diskWriteBytes, _ := disk.IOCountersStat.WriteBytes()

	fmt.Printf("DISKstat: %+v\n", diskStat)
	fmt.Printf("DISK count: %+v\n", diskCount)

	//NET

	//PROCESS
	//Number of processes running?

}

func writeToFile(text float64) {
	filename := "results/logfiletest.log"
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	_, err = f.WriteString(fmt.Sprintln(text))
	if err != nil {
		panic(err)
	}

	f.Close()
}
