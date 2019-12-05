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

//Defines channels that the keyboard inputs use to communicate to the workers
type keyChans struct {
	startSend    chan bool
	finishedSend chan bool
	currentCells chan cell
	turnsPrinted chan bool
	printTurns   chan bool
	pause        *sync.WaitGroup
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
	periodicOutput chan bool
	stop           *sync.WaitGroup
	threadsyncin   chan bool
	threadsyncout  chan byte

	periodicNumber chan int
	numberLogged   chan byte
	turnsync       chan int
	turnsback      chan int
}

// ioToDistributor defines all chans that the io goroutine will have to communicate with the distributor goroutine.
// Note the restrictions on chans being send-only or receive-only to prevent bugs.
type ioToDistributor struct {
	command <-chan ioCommand
	idle    chan<- bool

	filename <-chan string
	inputVal chan<- uint8

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

func collateBoard(dChans distributorChans, p golParams, k keyChans) []cell {
	var receivedFrom = 0
	for {
		<-k.finishedSend
		receivedFrom++
		if receivedFrom == p.threads {
			break
		}
	}

	var currentAlive []cell
	finishedloop := false
	for {
		select {
		case c := <-k.currentCells:
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
	return currentAlive
}

func keyboardInputs(p golParams, keyChan <-chan rune, dChans distributorChans, kChans keyChans) {
	paused := false
	for {
		time.Sleep(17 * time.Millisecond)
		select {
		case key := <-keyChan:
			switch key {
			case 's':
				kChans.startSend <- true
				currentAlive := collateBoard(dChans, p, kChans)
				go writePgmTurn(p, currentAlive)
			case 'p':
				kChans.printTurns <- true
				<-kChans.turnsPrinted
				kChans.pause.Add(1)
				fmt.Println("Paused")

				//Can continue on the next p press
				for {
					select {
					case key := <-keyChan:
						switch key {
						case 'p':
							kChans.pause.Done()
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
				kChans.startSend <- true
				currentAlive := collateBoard(dChans, p, kChans)
				kChans.pause.Add(1)
				writePgmTurn(p, currentAlive)
				StopControlServer()

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
	var keyChans keyChans

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

	startSend := make(chan bool)
	keyChans.startSend = startSend

	finishedSend := make(chan bool, p.threads)
	keyChans.finishedSend = finishedSend

	currentCells := make(chan cell, p.imageHeight*p.imageWidth)
	keyChans.currentCells = currentCells

	printTurns := make(chan bool)
	turnsPrinted := make(chan bool, 1)

	keyChans.printTurns = printTurns
	keyChans.turnsPrinted = turnsPrinted

	var pause sync.WaitGroup
	keyChans.pause = &pause

	aliveCells := make(chan []cell)
	go periodic(dChans, p)
	go distributor(p, dChans, aliveCells, keyChans)

	go keyboardInputs(p, keyChan, dChans, keyChans)
	stop.Add(1)
	go pgmIo(p, ioChans)

	alive := <-aliveCells
	dChans.io.stop.Wait()
	return alive
}

func periodic(d distributorChans, p golParams) {
	for {
		d.io.periodicOutput <- true
		number := 0
		for i := 0; i < p.threads; i++ {
			number += <-d.io.periodicNumber
		}
		fmt.Println("Cells alive: ", number)
		time.Sleep(2 * time.Second)
	}
}

// main is the function called when starting Game of Life with 'make gol'
// Do not edit until Stage 2.
func main() {
	var params golParams

	flag.IntVar(
		&params.threads,
		"t",
		8,
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

	params.turns = 500000

	startControlServer(params)
	keyChannel := make(chan rune, 60)
	go getKeyboardCommand(keyChannel)
	gameOfLife(params, keyChannel)
	StopControlServer()
}
