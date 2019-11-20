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
func mod(a, b int) int {
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

//copies the world from one slice to another
func copyworld(world [][]byte) [][]byte {
	worldnew := make([][]byte, len(world))
	for i := range world {
		worldnew[i] = make([]byte, len(world[i]))
	}
	for y, row := range world {
		for x, cell := range row {
			worldnew[y][x] = cell
		}
	}
	return worldnew
}
func golWorker(p golParams, worldslice [][]byte, index int, slicereturns chan worldpart) {
	worldnew := copyworld(worldslice)
	for y := 1; y < len(worldslice)-1; y++ {
		for x := 0; x < len(worldslice[y]); x++ {
			worldnew[y][x] = worldslice[y][x]
			neighbours := numNeighbours(x, y, worldslice, p)
			if neighbours < 2 && worldslice[y][x] == 255 { // 1 or fewer neighbours dies
				worldnew[y][x] = 0
			} else if (neighbours == 2 || neighbours == 3) && worldslice[y][x] == 255 { //2 or 3 neighbours stays alive
				//do nothing
			} else if neighbours > 3 && worldslice[y][x] == 255 { //4 or more neighbours dies
				worldnew[y][x] = 0
			} else if worldslice[y][x] == 0 && neighbours == 3 { //empty with 3 neighbours becomes alive
				worldnew[y][x] = 255
			}
		}
	}
	part := worldpart{index: index, worldslice: worldnew}
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
		//splitworld
		var worlds [][][]byte
		//printGrid(world, p)
		if p.imageHeight == 1 || p.imageHeight < p.threads {
			worlds[0] = world
		} else {
			for i := 0; i < (p.threads); i++ {
				//fmt.Println("i: ", i)
				//fmt.Println("p/imageHeight/p.threads: ", p.imageHeight/p.threads)
				var worldslice [][]byte
				if i == 0 {
					worldslice = append(worldslice, world[p.imageHeight-1:p.imageHeight]...)
					worldslice = append(worldslice, world[0:(p.imageHeight/p.threads)+1]...)
					worlds = append(worlds, worldslice)
				} else if i == (p.threads - 1) {

					worldslice = append(worldslice, world[(i*(p.imageHeight/p.threads))-1:(i*(p.imageHeight/p.threads))+(p.imageHeight/p.threads)]...)
					worldslice = append(worldslice, world[0:1]...)
					worlds = append(worlds, worldslice)
				} else {
					worldslice = append(worldslice, world[(i*(p.imageHeight/p.threads))-1:(i*(p.imageHeight/p.threads))+(p.imageHeight/p.threads)+1]...)
					worlds = append(worlds, worldslice)
				}
				//printGrid(worlds[i], p)
			}
		}
		slicereturns := make(chan worldpart, p.threads)
		for x, worldslice := range worlds {

			go golWorker(p, worldslice, x, slicereturns)
		}
		returns := make([][][]byte, p.threads)
		for i := 0; i < p.threads; i++ {

			something := <-slicereturns
			returns[something.index] = something.worldslice
		}

		var worldnew [][]byte
		//fmt.Println("length of returns: ", len(returns))
		for _, part := range returns {
			//printGrid(part, p)
			worldnew = append(worldnew, part[1:len(part)-1]...)
		}
		//fmt.Println("whole world: ")
		//printGrid(worldnew, p)
		world = copyworld(worldnew)
		//printGrid(world, p)
	}

	// Create an empty slice to store coordinates of cells that are still alive after p.turns are done.
	var finalAlive []cell
	// Go through the world and append the cells that are still alive.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] != 0 {
				finalAlive = append(finalAlive, cell{x: x, y: y})
			}
		}
	}

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	fmt.Println(finalAlive)

	// Telling pgm.go to start the write function
	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")
	d.io.output <- finalAlive

	alive <- finalAlive
}
