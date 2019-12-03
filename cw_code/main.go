package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"
)

// golParams provides the details of how to run the Game of Life and which image to load.
type golParams struct {
	turns       int
	threads     int
	imageWidth  int
	imageHeight int
}

// ioCommand allows requesting behaviour from the io (pgm) goroutine.
type ioCommand uint8

// This is a way of creating enums in Go.
// It will evaluate to:
//		ioOutput 	= 0
//		ioInput 	= 1
//		ioCheckIdle = 2
const (
	ioOutput ioCommand = iota
	ioInput
	ioCheckIdle
)

// cell is used as the return type for the testing framework.
type cell struct {
	x, y int
}

// distributorToIo defines all chans that the distributor goroutine will have to communicate with the io goroutine.
// Note the restrictions on chans being send-only or receive-only to prevent bugs.
type distributorToIo struct {
	command chan<- ioCommand
	idle    <-chan bool

	filename chan<- string
	inputVal <-chan uint8

	//aliveOutput sends cells from distributer to pgm
	aliveOutput    chan []cell
	pause          *sync.WaitGroup
	output         chan<- []cell
	periodicOutput chan bool
	stop           *sync.WaitGroup
	threadsyncin   chan bool
	threadsyncout  chan byte

	periodicNumber chan int
	numberLogged   chan byte
	turnsync       chan int
	turnsback      chan int

	outputS      chan int
	currentCells chan cell
	stopS        chan int
}

// ioToDistributor defines all chans that the io goroutine will have to communicate with the distributor goroutine.
// Note the restrictions on chans being send-only or receive-only to prevent bugs.
type ioToDistributor struct {
	command <-chan ioCommand
	idle    chan<- bool

	filename <-chan string
	inputVal chan<- uint8

	output      <-chan []cell
	aliveOutput chan []cell
	stop        *sync.WaitGroup
}

// distributorChans stores all the chans that the distributor goroutine will use.
type distributorChans struct {
	io distributorToIo
}

// ioChans stores all the chans that the io goroutine will use.
type ioChans struct {
	distributor ioToDistributor
}

func keyboardInputs(p golParams, keyChan <-chan rune, dChans distributorChans, ioChans ioChans) {
	paused := false
	for {
		//Receives the cells that are currently alive from the distributer
		//currentAlive := <-ioChans.distributor.output
		select {
		case key := <-keyChan:
			switch key {
			case 's':

				dChans.io.outputS <- 1

				var receivedFrom = 0
				for {
					<-dChans.io.stopS
					receivedFrom++
					if receivedFrom == p.threads {
						break
					}
				}

				var currentAlive []cell
				finishedloop := false
				for {
					select {
					case c := <-dChans.io.currentCells:
						currentAlive = append(currentAlive, c)
						break
					default:
						finishedloop = true
						break
					}
					if finishedloop {
						break
					}
				}
				//runs a go routine each time a new pgm file is to be made
				go writePgmTurn(p, currentAlive)
			case 'p':
				dChans.io.pause.Add(1)
				fmt.Println("Paused")
				//Creates a world to print and then pauses the distributer
				/*world := make([][]byte, p.imageHeight)
				for i := range world {
					world[i] = make([]byte, p.imageWidth)
				}

				for _, cell := range currentAlive {
					world[cell.y][cell.x] = 255
				}
				fmt.Println("Current state of the world:")
				printGrid(world)*/

				//On the next 'p' press the distributer can continue
				for {
					select {
					case key := <-keyChan:
						switch key {
						case 'p':
							dChans.io.pause.Done()
							fmt.Println("Continuing")
							paused = true
							break
						}
					}
					if paused {
						paused = false
						break
					}
				}
			case 'q':
				//Prints world in current state and quits
				/*world := make([][]byte, p.imageHeight)
				for i := range world {
					world[i] = make([]byte, p.imageWidth)
				}

				for _, cell := range currentAlive {
					world[cell.y][cell.x] = 255
				}
				fmt.Println("Final state of the world:")
				printGrid(world)*/

				os.Exit(0)
			}
		default:
			//do nothing
		}
	}
}

// gameOfLife is the function called by the testing framework.
// It makes some channels and starts relevant goroutines.
// It places the created channels in the relevant structs.
// It returns an array of alive cells returned by the distributor.
func gameOfLife(p golParams, keyChan <-chan rune) []cell {
	var dChans distributorChans
	var ioChans ioChans

	ioCommand := make(chan ioCommand)
	dChans.io.command = ioCommand
	ioChans.distributor.command = ioCommand

	ioIdle := make(chan bool)
	dChans.io.idle = ioIdle
	ioChans.distributor.idle = ioIdle

	ioFilename := make(chan string)
	dChans.io.filename = ioFilename
	ioChans.distributor.filename = ioFilename

	inputVal := make(chan uint8)
	dChans.io.inputVal = inputVal
	ioChans.distributor.inputVal = inputVal

	output := make(chan []cell)
	dChans.io.output = output
	ioChans.distributor.output = output

	periodicOutput := make(chan bool, p.threads)
	dChans.io.periodicOutput = periodicOutput
	periodicNumber := make(chan int, p.threads*p.threads*p.threads)
	dChans.io.periodicNumber = periodicNumber
	numberLogged := make(chan byte, p.threads*p.threads*p.threads)
	dChans.io.numberLogged = numberLogged
	turnsync := make(chan int)
	dChans.io.turnsync = turnsync
	turnsback := make(chan int)
	dChans.io.turnsback = turnsback

	var stop sync.WaitGroup
	dChans.io.stop = &stop
	ioChans.distributor.stop = &stop
	threadsyncin := make(chan bool, p.threads)
	dChans.io.threadsyncin = threadsyncin

	threadsyncout := make(chan byte, p.threads)
	dChans.io.threadsyncout = threadsyncout

	aliveOutput := make(chan []cell)
	dChans.io.aliveOutput = aliveOutput
	ioChans.distributor.aliveOutput = aliveOutput

	var pause sync.WaitGroup
	dChans.io.pause = &pause

	outputS := make(chan int, p.threads)
	dChans.io.outputS = outputS
	currentCells := make(chan cell, p.imageHeight*p.imageWidth)
	dChans.io.currentCells = currentCells
	stopS := make(chan int, p.threads)
	dChans.io.stopS = stopS

	aliveCells := make(chan []cell)
	go periodic(dChans, p)
	go distributor(p, dChans, aliveCells)

	go keyboardInputs(p, keyChan, dChans, ioChans)
	stop.Add(1)
	go pgmIo(p, ioChans)

	alive := <-aliveCells
	dChans.io.stop.Wait()
	return alive
}

func periodic(d distributorChans, p golParams) {
	for {
		//fmt.Println("send output signal")
		time.Sleep(2 * time.Second)
		fmt.Println("main:", 1)
		d.io.periodicOutput <- true
		number := 0
		for i := 0; i < p.threads; i++ {
			number += <-d.io.periodicNumber
		}
		fmt.Println(number)
	}
}

// main is the function called when starting Game of Life with 'make gol'
// Do not edit until Stage 2.
func main() {
	var params golParams

	flag.IntVar(
		&params.threads,
		"t",
		4,
		"Specify the number of worker threads to use. Defaults to 8.")

	flag.IntVar(
		&params.imageWidth,
		"w",
		512,
		"Specify the width of the image. Defaults to 512.")

	flag.IntVar(
		&params.imageHeight,
		"h",
		512,
		"Specify the height of the image. Defaults to 512.")

	flag.Parse()

	params.turns = 100000000

	startControlServer(params)
	keyChannel := make(chan rune)
	go getKeyboardCommand(keyChannel)
	gameOfLife(params, keyChannel)
	StopControlServer()
}
