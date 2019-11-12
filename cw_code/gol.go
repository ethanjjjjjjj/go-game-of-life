package main

import (
	"fmt"
	"strconv"
	"strings"
)

func printGrid(world [][]byte, p golParams) {
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] == 255 {
				fmt.Print("1 ")
			} else {
				fmt.Print("0 ")
			}
		}
		fmt.Println(" ")
	}
	fmt.Println("    ")
}

// returns number of alive neighbours to a cell
func numNeighbours(x int, y int, world [][]byte, p golParams) int {
	var num = 0
	var xx = x
	var yy = y
	if x < 0 {
		xx = x + p.imageWidth
	}

	if y < 0 {
		yy = y + p.imageHeight
	}

	if world[yy][xx] != 0 {
		num = num + 1
	}
	if world[(y+1)%p.imageHeight][xx] != 0 {
		num = num + 1
	}
	if world[(y+1)%p.imageHeight][x] != 0 {
		num = num + 1
	}
	if world[(y+1)%p.imageHeight][(x+1)%p.imageWidth] != 0 {
		num = num + 1
	}
	if world[y][(x+1)%p.imageWidth] != 0 {
		num = num + 1
	}
	if world[yy][(x+1)%p.imageWidth] != 0 {
		num = num + 1
	}
	if world[yy][x] != 0 {
		num = num + 1
	}
	if world[yy][xx] != 0 {
		num = num + 1
	}
	return num
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
		worldnew := make([][]byte, p.imageHeight)
		for y := 0; y < p.imageHeight; y++ {
			for x := 0; x < p.imageWidth; x++ {

				for i := range worldnew {
					worldnew[i] = make([]byte, p.imageWidth)
				}
				copy(worldnew, world)
				neighbours := numNeighbours(x, y, world, p)
				if neighbours < 2 {
					worldnew[y][x] = 0
				} else if neighbours > 1 || neighbours < 4 {

				} else if neighbours > 3 {
					worldnew[y][x] = 0
				} else if world[y][x] == 0 && neighbours == 3 {
					worldnew[y][x] = 255
				}

				// Placeholder for the actual Game of Life logic: flips alive cells to dead and dead cells to alive.
				//1 or less neighbours dies
				//2 or 3 neighbours stays alive
				//4 or more neighbours dies
				//empty with 3 neighbours becomes alive
				//world[y][x] = world[y][x] ^ 0xFF
			}
		}
		//printGrid(world, p)
		world := make([][]byte, p.imageHeight)
		copy(world, worldnew)
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
	alive <- finalAlive
}
