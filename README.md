# Peer Observation of Observation Units

## Approach and purpose

This project will develop an approach to:
* Let observation units observe data observed by observation units. 
* To gradually accumulate the data to observation units being a DAO Store (there can be multiple DAO Stores depending on user needs).
* Do a prototype of such a system focused on three levels of observation units: (i) In-situ observation units being (ii) observed by back-end observation units, being (iii) observed by a DAO Store observation unit.

The purpose is to fetch and accumulate data observed by observation units for further use.

### Prerequisites

What things you need to install the software and how to install them

```
Give examples
```

### Installing

A step by step series of examples that tell you have to get a development env running

Say what the step will be

```
Give the example
```


End with an example of getting some data out of the system or using it for a little demo

### Running the program/tests

- Start with the simulator:
    - `numCH`: number of cluster heads (e.g.: 5 equals 5%)

```
go run simulation.go -numCH=5

```

- Run observationunits:
    - `-Simport=8080`: port of Simulator
    - `-host=localhost`: host of OU
    - `-port=8082`: port of OU

```
go run server.go run -Simport=8080 -host=localhost -port=:8082

```

## Built With

* [Golang](https://golang.org/) - The program language


## Authors

* **Camilla Stormoen** - *Initial work* - [GitHub](https://github.com/camillavilla1)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details


