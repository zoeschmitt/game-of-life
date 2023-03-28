package main

import (
	"fmt"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"log"
	"math/rand"
	"runtime"
	"strings"
	"time"
)

const (
	width     = 500
	height    = 500
	rows      = 100
	columns   = 100
	threshold = 0.15
	fps       = 2
	// These are strings containing GLSL source code for two shaders, one for a vertex shader and another for a fragment shader.
	// The only thing special about these strings is that they both end in a null-termination character, \x00 - a requirement for
	// OpenGL to be able to compile them. Make note of the fragmentShaderSource, this is where we define the color of our shape
	// in RGBA format using a vec4. You can change the value here, which is currently RGBA(1, 1, 1, 1) or white, to change the
	// color of the triangle.
	vertexShaderSource = `
    #version 410
    in vec3 vp;
    void main() {
        gl_Position = vec4(vp, 1.0);
    }
` + "\x00"
	fragmentShaderSource = `
    #version 410
    out vec4 frag_colour;
    void main() {
        frag_colour = vec4(1, 1, 1, 1);
    }
` + "\x00"
)

// The slice contains 9 values, three for each vertex of a triangle.
// The top line, 0, 0.5, 0, is the top vertex represented as X, Y, and Z coordinates,
// the second line is the left vertex, and the third line is the right vertex.
// Each of these pairs of three represents the X, Y, and Z coordinates of the vertex relative
// to the center of the window, between -1 and 1.
var (
	triangle = []float32{
		-0.5, 0.5, 0, // top
		-0.5, -0.5, 0, // left
		0.5, -0.5, 0, // right
	}
	square = []float32{
		-0.5, 0.5, 0,
		-0.5, -0.5, 0,
		0.5, -0.5, 0,

		-0.5, 0.5, 0,
		0.5, 0.5, 0,
		0.5, -0.5, 0,
	}
)

type cell struct {
	// A drawable is a square Vertex Array Object.
	drawable uint32

	alive     bool
	aliveNext bool

	x int
	y int
}

func main() {
	// Ensures we will always execute in the same operating system thread,
	// which is important for GLFW which must always be called from the same thread it was initialized on.
	runtime.LockOSThread()

	window := initGlfw()
	defer glfw.Terminate()
	program := initOpenGL()

	cells := makeCells()
	for !window.ShouldClose() {
		t := time.Now()

		for x := range cells {
			for _, c := range cells[x] {
				c.checkState(cells)
			}
		}

		draw(cells, window, program)

		// reduce the game speed by introducing a frames-per-second limitation in the main loop.
		// 2 game iterations per second.
		time.Sleep(time.Second/time.Duration(fps) - time.Since(t))
	}
}

// initGlfw initializes glfw and returns a Window to use.
func initGlfw() *glfw.Window {
	if err := glfw.Init(); err != nil {
		panic(err)
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	// Binding the window to our current thread.
	window, err := glfw.CreateWindow(width, height, "Conway's Game of Life", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	return window
}

// initOpenGL initializes OpenGL and returns an intiialized program.
func initOpenGL() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	// A program gives us a reference to store shaders, which can then be used for drawing.
	prog := gl.CreateProgram()
	gl.AttachShader(prog, vertexShader)
	gl.AttachShader(prog, fragmentShader)
	gl.LinkProgram(prog)
	return prog
}

func draw(cells [][]*cell, window *glfw.Window, program uint32) {
	// Remove anything from the window that was drawn last frame, giving us a clean slate.
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(program)

	// Loop over each cell and have it draw itself.
	for x := range cells {
		for _, c := range cells[x] {
			c.draw()
		}
	}

	// Check if there were any mouse or keyboard events.
	glfw.PollEvents()
	// Buffer swapping is important because GLFW (like many graphics libraries) uses double buffering,
	// meaning everything you draw is actually drawn to an invisible canvas, and only put onto the
	// visible canvas when you’re ready - which in this case, is indicated by calling SwapBuffers
	window.SwapBuffers()
}

// makeVao initializes and returns a vertex array from the points provided.
// vao = Vertex Array Object
func makeVao(points []float32) uint32 {
	// First we create a Vertex Buffer Object or vbo to bind our vao to, which is created by providing
	// the size (4 x len(points)) and a pointer to the points (gl.Ptr(points)).
	// You may be wondering why it’s 4 x len(points) - why not 6 or 3 or 1078?
	// The reason is we are using float32 slices, and a 32-bit float has 4 bytes, so we are saying
	// the size of the buffer, in bytes, is 4 times the number of points.
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

	return vao
}

// a vertex shader manipulates the vertices to be drawn by OpenGL and generates the data passed to the fragment shader,
// which then determines the color of each fragment (you can just consider a fragment to be a pixel) to be drawn to the screen.
// The purpose of this function is to receive the shader source code as a string as well as its type,
// and return a pointer to the resulting compiled shader.
func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

// makeCells creates and returns a 2D slice of pointers to cell structs.
// The slice has a length of 'rows', and each row has a length of 'columns'.
// Each cell in the slice is a new cell struct created using the newCell function.
// Returns the 2D slice of cell pointers.
func makeCells() [][]*cell {
	// use the current time as the randomization seed, giving each game a unique starting state.
	rand.Seed(time.Now().UnixNano())

	cells := make([][]*cell, rows, rows)
	for x := 0; x < rows; x++ {
		for y := 0; y < columns; y++ {
			c := newCell(x, y)

			// set cells alive state equal to the result of a random float, between 0.0 and 1.0,
			// being less than threshold (0.15). Each cell has a 15% chance of starting out alive.
			c.alive = rand.Float64() < threshold
			c.aliveNext = c.alive

			cells[x] = append(cells[x], c)
		}
	}

	return cells
}

func newCell(x, y int) *cell {
	// Create a copy of our square definition. This allows us to change its contents to customize
	// the current cell’s position, without impacting any other cells that are also using the square slice.
	points := make([]float32, len(square), len(square))
	copy(points, square)

	// Next we iterate over the points copy and act based on the current index. We use a modulo operation
	// to determine if we’re at an X (i % 3 == 0) or Y (i % 3 == 1) coordinate of the shape (skipping Z since
	// we’re operating in two dimensions) and determine the size (as a percentage of the entire game board)
	// of the cell accordingly, as well as it’s position based on the X and Y coordinate of the cell on the game board.
	for i := 0; i < len(points); i++ {
		var position float32
		var size float32
		switch i % 3 {
		case 0:
			size = 1.0 / float32(columns)
			position = float32(x) * size
		case 1:
			size = 1.0 / float32(rows)
			position = float32(y) * size
		default:
			continue
		}

		// Modify the points which currently contain a combination of 0.5, 0 and -0.5 as we defined them in the
		// square slice. If the point is less than zero, we set it equal to the position times 2 (because OpenGL
		// coordinates have a range of 2, between -1 and 1), minus 1 to normalize to OpenGL coordinates.
		// If the position is greater than or equal to zero, we do the same thing but add the size we calculated.
		// The purpose of this is to set the scale of each cell so that it fills only its percentage of the game board.
		// Since we have 10 rows and 10 columns, each cell will be given 10% of the width and 10% of the height of the game board.
		if points[i] < 0 {
			points[i] = (position * 2) - 1
		} else {
			points[i] = ((position + size) * 2) - 1
		}
	}

	// After all the points have been scaled and positioned, we create a cell with the X and Y coordinate provided,
	// and set the drawable field equal to a Vertex Array Object created from the points slice we just manipulated.
	return &cell{
		drawable: makeVao(points),

		x: x,
		y: y,
	}
}

// Each cell needs to know how to draw itself.
func (c *cell) draw() {
	if !c.alive {
		return
	}

	gl.BindVertexArray(c.drawable)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(square)/3))
}

// checkState determines the state of the cell for the next tick of the game.
func (c *cell) checkState(cells [][]*cell) {
	c.alive = c.aliveNext
	c.aliveNext = c.alive

	liveCount := c.liveNeighbors(cells)
	if c.alive {
		// 1. Any live cell with fewer than two live neighbours dies, as if caused by underpopulation.
		if liveCount < 2 {
			c.aliveNext = false
		}

		// 2. Any live cell with two or three live neighbours lives on to the next generation.
		if liveCount == 2 || liveCount == 3 {
			c.aliveNext = true
		}

		// 3. Any live cell with more than three live neighbours dies, as if by overpopulation.
		if liveCount > 3 {
			c.aliveNext = false
		}
	} else {
		// 4. Any dead cell with exactly three live neighbours becomes a live cell, as if by reproduction.
		if liveCount == 3 {
			c.aliveNext = true
		}
	}
}

// liveNeighbors returns the number of live neighbors for a cell.
func (c *cell) liveNeighbors(cells [][]*cell) int {
	var liveCount int
	add := func(x, y int) {
		// If we're at an edge, check the other side of the board.
		if x == len(cells) {
			x = 0
		} else if x == -1 {
			x = len(cells) - 1
		}
		if y == len(cells[x]) {
			y = 0
		} else if y == -1 {
			y = len(cells[x]) - 1
		}

		if cells[x][y].alive {
			liveCount++
		}
	}

	add(c.x-1, c.y)   // To the left
	add(c.x+1, c.y)   // To the right
	add(c.x, c.y+1)   // up
	add(c.x, c.y-1)   // down
	add(c.x-1, c.y+1) // top-left
	add(c.x+1, c.y+1) // top-right
	add(c.x-1, c.y-1) // bottom-left
	add(c.x+1, c.y-1) // bottom-right

	return liveCount
}
