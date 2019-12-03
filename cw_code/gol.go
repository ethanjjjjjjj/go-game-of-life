package main

import (
	"fmt"
	"strconv"
	"strings"
)

type workerExchange struct {
	rTop <-chan byte //receiving the top row
	sTop chan<- byte //sending the top row
	rBot <-chan byte //receiving the bottom row
	sBot chan<- byte //sending the bottom row
}

type sliceInfo struct {
	index    int
	height   int
	width    int
	numAlive int
}

type workerIO struct {
	inputCell      chan cell
	outputCell     chan cell
	workerFinished chan bool
}

func printGrid(world [][]byte) {
	for _, row := range world {
		for _, cell := range row {
			if cell == 255 {
				fmt.Print("1 ")
			} else {
				fmt.Print("0 ")
			}
		}
		fmt.Println(" ")
	}
	fmt.Println(" ")
}

//getx and gety are used for checking neighbours when the x or y coordindate of the cell is at
//the edges of the world. This is used because modding is slow
func getx(x int, width int) int {
	if x == width {
		return 0
	} else if x == -1 {
		return width - 1
	}
	return x
}

func gety(y int, height int) int {
	if y == height {
		return 0
	} else if y == -1 {
		return height - 1
	}
	return y
}

// returns number of alive neighbours to a cell
func numNeighbours(x int, y int, world [][]byte) int {
	var num = 0
	Height := len(world)
	Width := len(world[0])
	x1 := getx(x-1, Width)
	x2 := getx(x+1, Width)
	y1 := getx(y-1, Height)
	y2 := getx(y+1, Height)

	if world[y][x1] != 0 {
		num++
	}
	if world[y2][x1] != 0 {
		num++
	}
	if world[y2][x] != 0 {
		num++
	}
	if world[y2][x2] != 0 {
		num++
	}
	if world[y][x2] != 0 {
		num++
	}
	if world[y1][x2] != 0 {
		num++
	}
	if world[y1][x] != 0 {
		num++
	}
	if world[y1][x1] != 0 {
		num++
	}

	return num
}

//Returns a slice of alive cells in the world
func aliveCells(world [][]byte) []cell {
	var alive []cell
	for y := 0; y < len(world); y++ {
		for x := 0; x < len(world[0]); x++ {
			if world[y][x] != 0 {
				alive = append(alive, cell{x: x, y: y})
			}
		}
	}
	return alive
}

func threadSyncer(d distributorChans, p golParams, k keyChans) {
	var signal byte
	for {
		signal = 0
		select {

		case <-d.io.periodicOutput:
			signal = 1
			fmt.Println("Received")

		case <-k.startSend:
			signal = 2
		case <-k.printTurns:
			signal = 3
		default:
		}

		for i := 0; i < p.threads; i++ {
			<-d.io.threadsyncin
		}

		for i := 0; i < p.threads; i++ {
			d.io.threadsyncout <- signal
		}
	}
}
func golWorker(workerIO workerIO, workerChans workerExchange, sliceInfo sliceInfo, p golParams, d distributorChans, k keyChans) {

	worldslice := make([][]byte, sliceInfo.height)
	rows := p.imageHeight / p.threads
	remainder := p.imageHeight % p.threads
	for i := 0; i < sliceInfo.height; i++ {
		worldslice[i] = make([]byte, sliceInfo.width)
	}

	//Receives alive cells and puts them into the world
	for i := 0; i < sliceInfo.numAlive; i++ {
		currentcell := <-workerIO.inputCell
		worldslice[currentcell.y][currentcell.x] = 255
	}

	for turns := 0; turns < p.turns; turns++ {

		d.io.threadsyncin <- true
		signal := <-d.io.threadsyncout
		if signal == 1 {
			d.io.periodicNumber <- len(aliveCells(worldslice[1 : len(worldslice)-1]))

		} else if signal == 2 {
			alive := aliveCells(worldslice[1 : len(worldslice)-1])
			n := len(alive)
			var yActual = 0
			for i := 0; i < n; i++ {
				if sliceInfo.index < remainder {
					yActual = (sliceInfo.index * rows) + sliceInfo.index + alive[i].y
				} else {
					yActual = (sliceInfo.index * rows) + remainder + alive[i].y
				}
				toSend := cell{x: alive[i].x, y: yActual}
				k.currentCells <- toSend
			}
			k.finishedSend <- true
		} else if signal == 3 && sliceInfo.index == 0 {
			fmt.Println("Turn: ", turns)
			k.turnsPrinted <- true
		}
		k.pause.Wait()

		worldnew := make([][]byte, sliceInfo.height)
		for i := 0; i < sliceInfo.height; i++ {
			worldnew[i] = make([]byte, sliceInfo.width)
			copy(worldnew[i], worldslice[i])
		}

		for y := 1; y < len(worldslice)-1; y++ {
			for x := 0; x < len(worldslice[y]); x++ {
				neighbours := numNeighbours(x, y, worldslice)
				if neighbours < 2 && worldslice[y][x] == 255 { // 1 or fewer neighbours dies
					worldnew[y][x] = 0
				} else if neighbours > 3 && worldslice[y][x] == 255 { //4 or more neighbours dies
					worldnew[y][x] = 0
				} else if worldslice[y][x] == 0 && neighbours == 3 { //empty with 3 neighbours becomes alive
					worldnew[y][x] = 255
				}
			}
		}

		//Odd indexed workers send their rows before receiving
		if sliceInfo.index%2 != 0 {
			for i := 0; i < sliceInfo.width; i++ {
				workerChans.sTop <- worldnew[1][i]
				workerChans.sBot <- worldnew[sliceInfo.height-2][i]
			}
			for i := 0; i < sliceInfo.width; i++ {
				worldnew[sliceInfo.height-1][i] = <-workerChans.rBot
				worldnew[0][i] = <-workerChans.rTop
			}
		} else if sliceInfo.index%2 == 0 { //Even indexed workers receive their rows before sending
			for i := 0; i < sliceInfo.width; i++ {
				worldnew[sliceInfo.height-1][i] = <-workerChans.rBot
				worldnew[0][i] = <-workerChans.rTop
			}
			for i := 0; i < sliceInfo.width; i++ {
				workerChans.sTop <- worldnew[1][i]
				workerChans.sBot <- worldnew[sliceInfo.height-2][i]
			}
		}
		copy(worldslice, worldnew)

	}
	for y := 1; y < len(worldslice)-1; y++ {
		for x := 0; x < len(worldslice[y]); x++ {
			if worldslice[y][x] == 255 {
				if sliceInfo.index < remainder {
					cell1 := cell{x: x, y: (sliceInfo.index * rows) + sliceInfo.index + y - 1}
					workerIO.outputCell <- cell1
				} else {
					cell1 := cell{x: x, y: (sliceInfo.index * rows) + remainder + y - 1}
					workerIO.outputCell <- cell1
				}
			}
		}
	}
	workerIO.workerFinished <- true
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell, k keyChans) {
	go threadSyncer(d, p, k)

	//channels for passing the cells through to workers
	//worldData := make(chan cell)

	// Create the 2D slice to store the world.
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	// The io goroutine sends the requested image byte by byte, in rows.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				fmt.Println("Alive cell at", x, y)
				world[y][x] = val
			}
		}
	}

	//slicereturns := make(chan cell, p.imageHeight*p.imageWidth)
	//workerfinished := make(chan bool, p.threads)

	var workerIO workerIO
	workerIO.inputCell = make(chan cell)
	workerIO.outputCell = make(chan cell, p.imageHeight*p.imageWidth)
	workerIO.workerFinished = make(chan bool, p.threads)

	rows, remainder := p.imageHeight/p.threads, p.imageHeight%p.threads

	//rowsindex is used to append the correct amount of rows to each slice
	rowsindex := 0

	//For last thread bottom
	rTop1 := make(chan byte, p.imageWidth*p.threads*p.threads)
	sTop1 := make(chan byte, p.imageWidth*p.threads*p.threads)

	//Current thread top, next thread bottom
	rememberBotR := make(chan byte, p.imageWidth*p.threads*p.threads)
	rememberBotS := make(chan byte, p.imageWidth*p.threads*p.threads)
	for i := 0; i < p.threads; i++ {

		var worldslice [][]byte
		//The first thread needs the final row from the other side of the world appended to its slice
		if i == 0 {
			worldslice = append(worldslice, world[len(world)-1:len(world)]...)
		} else {
			//appending the first additional row to a slice
			worldslice = append(worldslice, world[rowsindex-1:rowsindex]...)
		}

		//Appends to each slice the correct number of rows
		worldslice = append(worldslice, world[rowsindex:rowsindex+rows]...)

		//the next thread will need to have the next set of rows above the last row appended
		rowsindex += rows

		//If the number of threads does not divide evenly, addtional rows are added to each slice
		if remainder > 0 {
			worldslice = append(worldslice, world[rowsindex:rowsindex+1]...)
			//The next slice needs to start from further up
			rowsindex++
			//There are fewer remainder rows to append next time
			remainder--
		}

		//The last thread needs the first row from the other side of the world appended to the end
		if i == p.threads-1 {
			worldslice = append(worldslice, world[0:1]...)
		} else {
			//the other threads have the next row appended
			worldslice = append(worldslice, world[rowsindex:rowsindex+1]...)
		}

		alive := aliveCells(worldslice)

		var sliceInfo sliceInfo
		sliceInfo.index = i
		sliceInfo.height = len(worldslice)
		sliceInfo.width = len(worldslice[0])
		sliceInfo.numAlive = len(alive)

		if i == 0 {
			var workerChans workerExchange
			workerChans.rTop = rTop1
			workerChans.sTop = sTop1
			workerChans.rBot = rememberBotR
			workerChans.sBot = rememberBotS
			go golWorker(workerIO, workerChans, sliceInfo, p, d, k)
		} else if i == p.threads-1 {
			var workerChans workerExchange
			workerChans.rTop = rememberBotS
			workerChans.sTop = rememberBotR
			workerChans.rBot = sTop1
			workerChans.sBot = rTop1
			go golWorker(workerIO, workerChans, sliceInfo, p, d, k)
		} else {
			var workerChans workerExchange
			workerChans.rTop = rememberBotS
			workerChans.sTop = rememberBotR
			var newChanR = make(chan byte, p.imageWidth*p.threads*p.threads)
			var newChanS = make(chan byte, p.imageWidth*p.threads*p.threads)
			rememberBotR = newChanR
			rememberBotS = newChanS
			workerChans.rBot = rememberBotR
			workerChans.sBot = rememberBotS
			go golWorker(workerIO, workerChans, sliceInfo, p, d, k)
		}
		for _, alivecell := range alive {
			workerIO.inputCell <- alivecell
		}

	}

	//Creates a 2D slice to reform the slices together
	worldnew := make([][]byte, p.imageHeight)
	for i := range world {
		worldnew[i] = make([]byte, p.imageWidth)
	}

	//to indicate how many threads have finished outputting their alive cells
	finished := 0

	for i := 0; i < p.threads; i++ {
		<-workerIO.workerFinished
		finished++
	}

	//once all the threads have finished, the alive cells can be received
	finishedloop := false
	for {
		select {
		case cell := <-workerIO.outputCell:
			worldnew[cell.y][cell.x] = 255
			break
		default:
			finishedloop = true
			break
		}
		if finishedloop {
			break
		}
	}

	var finalAlive = aliveCells(worldnew)

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Telling pgm.go to start the write function
	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")
	// Return the coordinates of cells that are still alive.
	d.io.aliveOutput <- finalAlive
	alive <- finalAlive
}
