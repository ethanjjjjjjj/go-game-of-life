package main

import (
	"fmt"
	"strconv"
	"strings"
)

type worldpart struct {
	index      int
	worldslice [][]byte
}

func printGrid(world [][]byte, p golParams) {
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
	fmt.Println("    ")
}

//a different mod function because go doesn't like modding negatives
func modold(a, b int) int {
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
}

func mod(d, m int) int {
	var res int = d % m
	if (res < 0 && m > 0) || (res > 0 && m < 0) {
		return res + m
	}
	return res
}

// returns number of alive neighbours to a cell
func numNeighbours(x int, y int, world [][]byte, p golParams) int {
	var num = 0
	Height := len(world)
	Width := len(world[0])
	if world[y][mod((x-1), Width)] != 0 {
		num = num + 1
	}
	if world[mod(y+1, Height)][mod((x-1), Width)] != 0 {
		num = num + 1
	}
	if world[mod(y+1, Height)][x] != 0 {
		num = num + 1
	}
	if world[mod(y+1, Height)][mod((x+1), Width)] != 0 {
		num = num + 1
	}
	if world[y][mod((x+1), Width)] != 0 {
		num = num + 1
	}
	if world[mod((y-1), Height)][mod((x+1), Width)] != 0 {
		num = num + 1
	}
	if world[mod((y-1), Height)][x] != 0 {
		num = num + 1
	}
	if world[mod((y-1), Height)][mod((x-1), Width)] != 0 {
		num = num + 1
	}
	return num
}

func aliveCells(p golParams, world [][]byte) []cell {
	// Create an empty slice to store coordinates of cells that are still alive after p.turns are done.
	var alive []cell
	// Go through the world and append the cells that are still alive.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] != 0 {
				alive = append(alive, cell{x: x, y: y})
			}
		}
	}
	return alive
}

func golWorker(p golParams, worldslice [][]byte, index int, slicereturns chan worldpart) {

	worldnew := make([][]byte, len(worldslice))
	for i := 0; i < len(worldslice); i++ {
		worldnew[i] = make([]byte, p.imageWidth)
		copy(worldnew[i], worldslice[i])
	}

	for y := 1; y < len(worldslice)-1; y++ {
		for x := 0; x < len(worldslice[y]); x++ {
			worldnew[y][x] = worldslice[y][x]
			neighbours := numNeighbours(x, y, worldslice, p)
			if neighbours < 2 && worldslice[y][x] == 255 { // 1 or fewer neighbours dies
				worldnew[y][x] = 0
			} else if neighbours > 3 && worldslice[y][x] == 255 { //4 or more neighbours dies
				worldnew[y][x] = 0
			} else if worldslice[y][x] == 0 && neighbours == 3 { //empty with 3 neighbours becomes alive
				worldnew[y][x] = 255
			}
		}
	}

	part := worldpart{index: index, worldslice: worldnew[1 : len(worldnew)-1]}
	slicereturns <- part
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell) {

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
		//fmt.Println("turn: ", turns, " threads: ", p.threads)
		d.io.pause.Wait()

		//splitworld
		slicereturns := make(chan worldpart, p.threads)
		rows, remainder := p.imageHeight/p.threads, p.imageHeight%p.threads
		//fmt.Println("rows: ", rows, "remainder: ", remainder)
		rowsindex := 0
		for i := 0; i < (p.threads); i++ {
			//fmt.Println("turn: ", i)
			var worldslice [][]byte
			if i == 0 {
				worldslice = append(worldslice, world[len(world)-1:len(world)]...)
			} else {
				worldslice = append(worldslice, world[rowsindex-1:rowsindex]...)
			}

			worldslice = append(worldslice, world[rowsindex:rowsindex+rows]...)
			rowsindex += rows
			if remainder > 0 {

				worldslice = append(worldslice, world[rowsindex:rowsindex+1]...)
				rowsindex++
				remainder--
			}
			if i == p.threads-1 {
				worldslice = append(worldslice, world[0:1]...)

			} else {
				worldslice = append(worldslice, world[rowsindex:rowsindex+1]...)
			}
			go golWorker(p, worldslice, i, slicereturns)
		}

		//returns := make([][][]byte, p.threads)
		worldnew := make([][]byte, p.imageHeight)
		for i := range world {
			worldnew[i] = make([]byte, p.imageWidth)
		}
		for i := 0; i < p.threads; i++ {
			something := <-slicereturns
			//returns[something.index] = something.worldslice
			for j := 0; j < len(something.worldslice); j++ {
				if something.index < p.imageHeight%p.threads {
					worldnew[(something.index*rows)+something.index+j] = something.worldslice[j]
				} else {
					worldnew[something.index*rows+p.imageHeight%p.threads+j] = something.worldslice[j]
				}

			}
		}

		var currentAlive = aliveCells(p, world)
		d.io.output <- currentAlive

		for i := 0; i < len(world); i++ {
			world[i] = make([]byte, p.imageWidth)
			copy(world[i], worldnew[i])
		}

		select {
		case <-d.io.periodicOutput:
			var currentAlive = 0
			for y := 0; y < p.imageHeight; y++ {
				for x := 0; x < p.imageWidth; x++ {
					if world[y][x] != 0 {
						currentAlive++
					}
				}
			}
			fmt.Println("Alive cells: ", currentAlive)
		default:
			//do nothing
		}
	}

	var finalAlive = aliveCells(p, world)

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	fmt.Println(finalAlive)

	// prints the grid every time a signal is received from the timer goroutine

	// Telling pgm.go to start the write function
	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")
	d.io.output <- finalAlive
	d.io.aliveOutput <- finalAlive

	alive <- finalAlive
}
