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

	if world[y][mod((x-1), p.imageWidth)] != 0 {
		num = num + 1
	}
	if world[mod(y+1, p.imageHeight)][mod((x-1), p.imageWidth)] != 0 {
		num = num + 1
	}
	if world[mod(y+1, p.imageHeight)][x] != 0 {
		num = num + 1
	}
	if world[mod(y+1, p.imageHeight)][mod((x+1), p.imageWidth)] != 0 {
		num = num + 1
	}
	if world[y][mod((x+1), p.imageWidth)] != 0 {
		num = num + 1
	}
	if world[mod((y-1), p.imageHeight)][mod((x+1), p.imageWidth)] != 0 {
		num = num + 1
	}
	if world[mod((y-1), p.imageHeight)][x] != 0 {
		num = num + 1
	}
	if world[mod((y-1), p.imageHeight)][mod((x-1), p.imageWidth)] != 0 {
		num = num + 1
	}
	return num
}

//copies the world from one slice to another
func copyworld(world [][]byte, p golParams) [][]byte {
	worldnew := make([][]byte, p.imageHeight)
	for i := range world {
		worldnew[i] = make([]byte, p.imageWidth)
	}
	for y, row := range world {
		for x, cell := range row {
			worldnew[y][x] = cell
		}
	}
	return worldnew
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
		worldnew := copyworld(world, p)
		for y := 0; y < p.imageHeight; y++ {
			for x := 0; x < p.imageWidth; x++ {
				worldnew[y][x] = world[y][x]
				neighbours := numNeighbours(x, y, world, p)
				if neighbours < 2 && world[y][x] == 255 { // 1 or fewer neighbours dies
					worldnew[y][x] = 0
				} else if (neighbours == 2 || neighbours == 3) && world[y][x] == 255 { //2 or 3 neighbours stays alive
					//do nothing
				} else if neighbours > 3 && world[y][x] == 255 { //4 or more neighbours dies
					worldnew[y][x] = 0
				} else if world[y][x] == 0 && neighbours == 3 { //empty with 3 neighbours becomes alive
					worldnew[y][x] = 255
				}
			}
		}
		world = copyworld(worldnew, p)
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
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")
}
