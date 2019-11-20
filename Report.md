# Concurrent Computing Coursework 1: Game of life

### Ethan Williams and Cassandra McCormack

## Funtionality and Design

The first problem we encountered was using the mod operator in Go. As the world grid wraps around the edges, a cell on the edge of the world has neighbours on the other side of the world. When a neighbour is checked for a cell, the coordinate must be modded by the image height or width in case this neighbour is on the other side of the world. However, if the cell had an x or y coordinate of 0, its neighbours will x and y coordinates of -1 mod image size were not checkced correctly, because 
```
-1 % p.imageWidth
``` 
will return -1, giving an array out of bounds error. 

To fix this, we made our own mod function, which repeatedly adds or subtracts the denominator from the numerator until the numerator is positive and less than the denominator. Using this,
```
mod(-1,p.imageWidth)
```
will return p.imageWidth, which is the correct value.

