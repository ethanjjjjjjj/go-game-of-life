package main

import (
	"fmt"
	"strconv"
	"strings"
)

type workerExchange struct {
	//receiving the top row
	rTop <-chan byte
	//sending the top row
	sTop chan<- byte
	//receiving the bottom row
	rBot <-chan byte
	//sending the bottom row
	sBot chan<- byte
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

	if world[y][getx(x-1, Width)] != 0 {
		num++
	}
	if world[gety(y+1, Height)][getx(x-1, Width)] != 0 {
		num++
	}
	if world[gety(y+1, Height)][x] != 0 {
		num++
	}
	if world[gety(y+1, Height)][getx(x+1, Width)] != 0 {
		num++
	}
	if world[y][getx(x+1, Width)] != 0 {
		num++
	}
	if world[gety(y-1, Height)][getx(x+1, Width)] != 0 {
		num++
	}
	if world[gety(y-1, Height)][x] != 0 {
		num++
	}
	if world[gety(y-1, Height)][getx(x-1, Width)] != 0 {
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

func golWorker(workerChans workerExchange, worldData chan cell, index int, slicereturns chan cell, height int, width int, numAlive int, p golParams, workerFinished chan bool) {

	worldslice := make([][]byte, height)
	rows := p.imageHeight / p.threads
	remainder := p.imageHeight % p.threads
	for i := 0; i < height; i++ {
		worldslice[i] = make([]byte, width)
	}

	//Receives alive cells and puts them into the world
	for i := 0; i < numAlive; i++ {
		currentcell := <-worldData
		worldslice[currentcell.y][currentcell.x] = 255
	}

	for turns := 0; turns < p.turns; turns++ {
		worldnew := make([][]byte, height)

		for i := 0; i < height; i++ {
			worldnew[i] = make([]byte, width)
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
		if index%2 != 0 { 
			for i := 0; i < width; i++ {
				workerChans.sTop <- worldnew[1][i]
				workerChans.sBot <- worldnew[height-2][i]
			}
			for i := 0; i < width; i++ {
				worldnew[height-1][i] = <-workerChans.rBot
				worldnew[0][i] = <-workerChans.rTop
			}
		} else if index%2 == 0 { //Even indexed workers receive their rows before sending
			for i := 0; i < width; i++ {
				worldnew[height-1][i] = <-workerChans.rBot
				worldnew[0][i] = <-workerChans.rTop	
			}
			for i := 0; i < width; i++ {
				workerChans.sTop <- worldnew[1][i]
				workerChans.sBot <- worldnew[height-2][i]
			}
		}

		copy(worldslice, worldnew)

	}
	for y := 1; y < len(worldslice)-1; y++ {
		for x := 0; x < len(worldslice[y]); x++ {
			if worldslice[y][x] == 255 {
				if index < remainder {
					cell1 := cell{x: x, y: (index * rows) + index + y - 1}
					slicereturns <- cell1
				} else {
					cell1 := cell{x: x, y: (index * rows) + remainder + y - 1}
					slicereturns <- cell1
				}
			}
		}
	}
	workerFinished <- true
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell) {
	//channels for passing the cells through to workers
	worldData := make(chan cell)

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

	slicereturns := make(chan cell, p.imageHeight*p.imageWidth)
	workerfinished := make(chan bool)
	rows, remainder := p.imageHeight/p.threads, p.imageHeight%p.threads

	//rowsindex is used to append the correct amount of rows to each slice
	rowsindex := 0

	//For last thread bottom
	rTop1 := make(chan byte,p.imageWidth*p.threads)
	sTop1 := make(chan byte,p.imageWidth*p.threads)

	//Current thread top, next thread bottom
	rememberBotR := make(chan byte,p.imageWidth*p.threads)
	rememberBotS := make(chan byte,p.imageWidth*p.threads)
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
		if i == 0 {
			var workerChans workerExchange
			workerChans.rTop = rTop1
			workerChans.sTop = sTop1
			workerChans.rBot = rememberBotR
			workerChans.sBot = rememberBotS
			go golWorker(workerChans, worldData, i, slicereturns, len(worldslice), len(worldslice[0]), len(alive), p, workerfinished)
		} else if i == p.threads-1 {
			var workerChans workerExchange
			workerChans.rTop = rememberBotS
			workerChans.sTop = rememberBotR
			workerChans.rBot = sTop1
			workerChans.sBot = rTop1
			go golWorker(workerChans, worldData, i, slicereturns, len(worldslice), len(worldslice[0]), len(alive), p, workerfinished)
		} else {
			var workerChans workerExchange
			workerChans.rTop = rememberBotS
			workerChans.sTop = rememberBotR
			var newChanR = make(chan byte,p.imageWidth*p.threads)
			var newChanS = make(chan byte,p.imageWidth*p.threads)
			rememberBotR = newChanR
			rememberBotS = newChanS
			workerChans.rBot = rememberBotR
			workerChans.sBot = rememberBotS
			go golWorker(workerChans, worldData, i, slicereturns, len(worldslice), len(worldslice[0]), len(alive), p, workerfinished)
		}
		for _, alivecell := range alive {
			worldData <- alivecell
		}

	}

	//Will wait if paused
	//d.io.pause.Wait()

	//Creates a 2D slice to reform the slices together
	worldnew := make([][]byte, p.imageHeight)
	for i := range world {
		worldnew[i] = make([]byte, p.imageWidth)
	}

	//to indicate how many threads have finished outputting their alive cells
	finished := 0

	for i := 0; i < p.threads; i++ {
		<-workerfinished
		finished++
	}

	//once all the threads have finished, the alive cells can be received
	finishedloop := false
	for {
		select {
		case cell := <-slicereturns:
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

	//sends all the alive cells to the keyboardInputs go routine after every turn
	//so they can be used in creating pgm files when needed
	
	//var currentAlive = aliveCells(worldnew)
	//d.io.output <- currentAlive

	//Copies the new world to the original world slice for the next turn
	for i := 0; i < len(world); i++ {
		world[i] = make([]byte, p.imageWidth)
		copy(world[i], worldnew[i])
	}

	//Outputs the number of alive cells for the periodic outouts
	select {
	case <-d.io.periodicOutput:
		fmt.Println("Number of alive cells: ", len(aliveCells(world)))
	default:
		//do nothing
	}
	//}

	var finalAlive = aliveCells(world)

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	//fmt.Println(finalAlive)

	// Telling pgm.go to start the write function
	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")
	d.io.output <- finalAlive
	// Return the coordinates of cells that are still alive.
	d.io.aliveOutput <- finalAlive
	alive <- finalAlive
}
