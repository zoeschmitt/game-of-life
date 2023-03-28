# [The Game of Life,](https://en.wikipedia.org/wiki/Conway's_Game_of_Life)

Implementation using Go and OpenGL with [go-gl](https://github.com/go-gl) following this [tutorial](https://kylewbanks.com/blog/tutorial-opengl-with-golang-part-1-hello-opengl).

also known simply as Life, is a cellular automaton devised by the British mathematician John Horton Conway in 1970.

The “game” is a zero-player game, meaning that its evolution is determined by its initial state, requiring no further input. One interacts with the Game of Life by creating an initial configuration and observing how it evolves, or, for advanced “players”, by creating patterns with particular properties.

Rules

The universe of the Game of Life is an infinite two-dimensional orthogonal grid of square cells, each of which is in one of two possible states, alive or dead, or “populated” or “unpopulated” (the difference may seem minor, except when viewing it as an early model of human/urban behaviour simulation or how one views a blank space on a grid). Every cell interacts with its eight neighbours, which are the cells that are horizontally, vertically, or diagonally adjacent. At each step in time, the following transitions occur:

Any live cell with fewer than two live neighbours dies, as if caused by underpopulation.
Any live cell with two or three live neighbours lives on to the next generation.
Any live cell with more than three live neighbours dies, as if by overpopulation.
Any dead cell with exactly three live neighbours becomes a live cell, as if by reproduction.

One of the keys to Conway’s game is that each cell must determine its next state based on the current state of the board, at the same time. This means that if Cell (X=3, Y=4) changes state during its calculation, its neighbor at (X=4, Y=4) must determine its own state based on what (X=3, Y=4) was, not what is has become. Basically, this means we must loop through the cells and determine their next state without modifying their current state before we draw, and then on the next loop of the game we apply the new state and repeat.
