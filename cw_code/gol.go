package main

import (
	"fmt"
	"strconv"
	"strings"
)

//This struct is used when piecing the world back together
type worldpart struct {
	index      int
	worldslice [][]byte
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

//a different mod function because go doesn't like modding negatives
/*func mod(a, b int) int {
	if a < 0 {
		for {
			a = a + b
			if a >= 0 {
				break
			}
		}
	} else if a >= b {
		for {
			a = a - b
			if a < b {
				break
			}
		}
	}
	return a
}*/

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
	//This case is for when the x and y value of a cell and its neighbours
	//are not near the boundary of the world so the getx and gety functions
	//are not used unnecessarily

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

//Returns an array of alive cells in the world
func aliveCells(p golParams, world [][]byte) []cell {
	var alive []cell
	// Go through the world and append the cells that are still alive.
	for y := 0; y < len(world); y++ {
		for x := 0; x < len(world[0]); x++ {
			if world[y][x] != 0 {
				alive = append(alive, cell{x: x, y: y})
			}
		}
	}
	return alive
}

func golWorker(worldData chan cell, index int, slicereturns chan cell, height int, width int, numAlive int, p golParams, workerFinished chan bool) {
	worldslice := make([][]byte, height)
	worldnew := make([][]byte, height)
	rows := p.imageHeight / p.threads
	remainder := p.imageHeight % p.threads
	for i := 0; i < height; i++ {
		worldslice[i] = make([]byte, width)
		worldnew[i] = make([]byte, width)

	}

	for i := 0; i < numAlive; i++ {
		currentcell := <-worldData
		worldslice[currentcell.y][currentcell.x] = 255
	}
	for i, row := range worldslice {
		copy(worldnew[i], row)
	}
	//copies the slice to another one so the current slice is not overwritten prematurely
	//fmt.Println("thread started")
	//Will not compute on the top and bottom rows
	for y := 1; y < len(worldslice)-1; y++ {
		for x := 0; x < len(worldslice[y]); x++ {
			worldnew[y][x] = worldslice[y][x]
			neighbours := numNeighbours(x, y, worldslice)
			if neighbours < 2 && worldslice[y][x] == 255 { // 1 or fewer neighbours dies
				worldnew[y][x] = 0
			} else if neighbours > 3 && worldslice[y][x] == 255 { //4 or more neighbours dies
				worldnew[y][x] = 0
			} else if worldslice[y][x] == 0 && neighbours == 3 { //empty with 3 neighbours becomes alive
				worldnew[y][x] = 255

				//slicereturns <- cell{x: x, y: (index * rows) + remainder + y - 1}
				if index > remainder {
					cell1 := cell{x: x, y: (index * rows) + index + y - 1}
					fmt.Println(cell1)
					slicereturns <- cell1
				} else {
					cell1 := cell{x: x, y: (index * rows) + remainder + y - 1}
					fmt.Println(cell1)
					slicereturns <- cell1

				}

			} else if worldslice[y][x] == 255 {
				if index > remainder {
					cell1 := cell{x: x, y: (index * rows) + index + y - 1}
					fmt.Println(cell1)
					slicereturns <- cell1
				} else {
					cell1 := cell{x: x, y: (index * rows) + remainder + y - 1}
					fmt.Println(cell1)
					slicereturns <- cell1
				}
			}
		}
	}
	//fmt.Println("thread finished")
	workerFinished <- true
	//fmt.Println("thread returned")

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell) {
	//channels for passing the world through to workers
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

	// Calculate the new state of Game of Life after the given number of turns.
	for turns := 0; turns < p.turns; turns++ {

		//Will wait if paused
		d.io.pause.Wait()

		//splitworld
		slicereturns := make(chan cell, p.imageHeight*p.imageWidth)
		workerfinished := make(chan bool)
		rows, remainder := p.imageHeight/p.threads, p.imageHeight%p.threads
		//rowsindex is used to append the correct amount of rows to each slice
		rowsindex := 0
		fmt.Println("worldbefore")
		printGrid(world)
		for i := 0; i < (p.threads); i++ {
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
			alive := aliveCells(p, worldslice)
			go golWorker(worldData, i, slicereturns, len(worldslice), len(worldslice[0]), len(alive), p, workerfinished)

			for _, alivecell := range alive {
				worldData <- alivecell
			}

		}

		//Creates a 2D slice to reform the threads' slices together
		worldnew := make([][]byte, p.imageHeight)
		for i := range world {
			worldnew[i] = make([]byte, p.imageWidth)
		}

		threadsfinished := 0

		for {
			select {
			case <-workerfinished:
				threadsfinished++
			case cell := <-slicereturns:
				worldnew[cell.y][cell.x] = 255
			default:
				//do nothing
			}
			if threadsfinished == p.threads {
				break
			}

		}
		fmt.Println("world after")
		printGrid(worldnew)

		//sends all the alive cells to the keyboardInputs go routine after every turn
		//so they can be used in creating pgm files when needed

		var currentAlive = aliveCells(p, worldnew)
		d.io.output <- currentAlive

		//Copies the new world to the original world slice for the next turn
		for i := 0; i < len(world); i++ {
			world[i] = make([]byte, p.imageWidth)
			copy(world[i], worldnew[i])
		}

		//Outputs the number of alive cells for the periodic outouts
		select {
		case <-d.io.periodicOutput:
			fmt.Println("Number of alive cells: ", len(aliveCells(p, world)))
		default:
			//do nothing
		}
	}

	var finalAlive = aliveCells(p, world)

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	fmt.Println(finalAlive)

	// Telling pgm.go to start the write function
	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")
	d.io.output <- finalAlive
	// Return the coordinates of cells that are still alive.
	d.io.aliveOutput <- finalAlive
	alive <- finalAlive
}
