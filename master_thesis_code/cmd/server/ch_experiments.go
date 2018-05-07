package main

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"
)

/*ChExperiments measures cluster head count etc*/
func (ou *ObservationUnit) ChExperiments(pid int) {
	tickChan := time.NewTicker(time.Millisecond * 1000).C
	folder := "./cmd/server/results"

	path := folder + "/ch_experiments.csv"

	infoSlice2 := []string{}

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
			infoSlice2 = append(infoSlice2, t)
			//TIME END

			//PID!!
			infoSlice2 = append(infoSlice2, strconv.Itoa(pid))
			//PID END

			infoSlice2 = append(infoSlice2, strconv.Itoa(ou.Sends))
			infoSlice2 = append(infoSlice2, strconv.Itoa(ou.SendsToLeader))

			infoSlice2 = append(infoSlice2, strconv.Itoa(ou.ClusterHeadCount))

			infoSlice2 = append(infoSlice2, strconv.Itoa(ou.ReceivedDataPkt))

			AppendFile(path, writer, infoSlice2)
			infoSlice2 = []string{}

		}
	}
}
