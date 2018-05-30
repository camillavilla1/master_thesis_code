# Peer Observation of Observation Units

## Approach and purpose

This project will develop an approach to:
* Let observation units observe data observed by observation units. 
* To gradually accumulate the data to observation units being a DAO Store (there can be multiple DAO Stores depending on user needs).
* Do a prototype of such a system focused on three levels of observation units: (i) In-situ observation units being (ii) observed by back-end observation units, being (iii) observed by a DAO Store observation unit.

The purpose is to fetch and accumulate data observed by observation units for further use.

### Installing

Download and install Golang - https://golang.org/doc/install
Get and install gopsutil

```
go get github.com/shirou/gopsutil/...

```


End with an example of getting some data out of the system or using it for a little demo

### Running the program/tests

- Start with the simulator:
    - `numCH`: number of cluster heads (e.g.: 5 equals 5%)

```
go run simulation.go -numCH=5

```

- Run one observationunit:
    - `-Simport=8080`: port of Simulator
    - `-host=localhost`: host of OU
    - `-port=8082`: port of OU

```
go run server.go run -Simport=8080 -host=localhost -port=:8082

```

- Run multiple observationunits:
	- `3`: number of observationunits

```
go run run_all.go 3

```


## Built With

* [Golang](https://golang.org/) - The program language
* [Gopsutil](https://github.com/shirou/gopsutil) - gopsutil: psutil for golang


## Authors

* **Camilla Stormoen** - *Initial work* - [GitHub](https://github.com/camillavilla1)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details


