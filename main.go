package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("NYT Pips Puzzle Solver")
	fmt.Println("======================")

	input := promptForInput("Enter the grid dimensions ['x y']:")
	x, y := parseDimensions(input)
	// Initial grid squares, we will gradually populate and refine it from the user input
	grid := make([][]*GridSquare, y)

	fmt.Println("Let's enter the blank squares...")
	for i := 0; i < y; i++ {
		input = promptForInput(fmt.Sprintf("Enter the blank squares for row %d [1-indexed, space-separated]:", i+1))
		grid[i] = parseRow(x, input, i)
	}

	fmt.Println("Grid:")
	for _, row := range grid {
		for _, square := range row {
			if square != nil {
				fmt.Printf("p")
			} else {
				fmt.Printf("b")
			}
		}
		fmt.Println()
	}

	fmt.Println("Now for the restricted regions...")
	fmt.Println("The format is '<type> <args> <x1> <y1> <x2> <y2>...', 1-indexed")
	fmt.Println("The types and args are:")
	fmt.Println("  - greater than: 'gt <value>'")
	fmt.Println("  - less than: 'lt <value>'")
	fmt.Println("  - sums to: 'sum <value>'")
	fmt.Println("  - all equal: 'eq'")
	fmt.Println("Examples:")
	fmt.Println("  - 'gt 4 3 1'; square at index (3,1) is greater than 4")
	fmt.Println("  - 'lt 2 6 6'; square at index (6,6) is less than 2")
	fmt.Println("  - 'sum 12 5 5 5 6'; two squares at indices (5,5) and (5,6) sum to 12")
	fmt.Println("  - 'eq 1 1 1 2 2 1 2 2'; a 2x2 region starting at (1,1) and ending at (2,2) are all equal")
	fmt.Println("Now your turn!")

	for {
		input = promptForInput("Enter the next restricted region (or 'done' to finish):")
		if input == "done" {
			break
		}
		// Apply the restricted region to the grid
		parseRestrictedRegion(input, grid)
	}

	// Build a graph of node to neighbor. While this is not entirely necessary, it will allow us to keep the main algorithm cleaner and delegate neighbor checking down a level
	buildGraph(grid)

	fmt.Println("Finally, enter the dominos...")
	fmt.Println("Enter them as space-separated pairs of numbers, e.g. '1 2 0 6' represents a 1-2 domino and a 0-6 domino")
	input = promptForInput("Your turn:")
	dominoes := parseDominoes(input)
	// Keep track of  moves, we are finally ready to solve the puzzle
	moveQueue := make(MoveQueue, 0)

	// Pick a restricted square to start with
	emptySquare := pickEmptySquare(grid)

	if success := makeNextMove(grid, dominoes, &moveQueue, emptySquare); !success {
		fmt.Println("No solution found")
		return
	}

	fmt.Println("Solution found!")
	fmt.Println(moveQueue)
}

func makeNextMove(grid [][]*GridSquare, dominoes DominoSet, moveQueue *MoveQueue, emptySquare *GridSquare) (success bool) {
	// Get candidate dominos for this square.
	candidates := dominoes.FindAvailableCandidates(emptySquare)
	if len(candidates) == 0 {
		if len(dominoes) == 0 {
			// success condition: we have assigned all dominos
			return true
		}
		// failure condition: none of the dominos we have left can satisfy the puzzle
		return false
	}

	// Check candidate dominos in all possible orientations until we find one that we can assign
	for _, candidate := range candidates {
		numIterations := 8
		if !candidate.isRightMatch {
			// In this instance, we can skip half the combos immediately
			numIterations /= 2
		}
		if !candidate.isLeftMatch {
			// In this instance, we can skip half the combos immediately and go right for the swap
			numIterations /= 2
			moveQueue.TryPush(&Move{
				Label:      fmt.Sprintf("Swap domino %d-%d", candidate.Domino.Square1Value, candidate.Domino.Square2Value),
				Domino:     candidate.Domino,
				GridSquare: emptySquare,
				MoveType:   MoveTypeSwap,
			})
			defer func() {
				if !success {
					moveQueue.Pop()
				}
			}()
		}
		for i := 0; i < numIterations; i++ {
			move := &Move{
				Label:      fmt.Sprintf("Assign domino %d-%d to square %d,%d", candidate.Domino.Square1Value, candidate.Domino.Square2Value, emptySquare.X+1, emptySquare.Y+1),
				Domino:     candidate.Domino,
				GridSquare: emptySquare,
				MoveType:   MoveTypeAssign,
			}
			// We were able to place this domino in its current state, try to make another move
			if moveQueue.TryPush(move) {
				if success := makeNextMove(grid, dominoes, moveQueue, pickEmptySquare(grid)); success {
					// success condition: puzzle solved from this current state
					return true
				} else {
					// failure condition: no valid next move can be made from this new state, undo the move and try the next candidate state
					moveQueue.Pop()
				}
			}
			// Try the next orientation
			move = &Move{
				Label:      fmt.Sprintf("Rotate domino %d-%d", candidate.Domino.Square1Value, candidate.Domino.Square2Value),
				Domino:     candidate.Domino,
				GridSquare: emptySquare,
				MoveType:   MoveTypeRotate,
			}
			moveQueue.TryPush(move)
			defer func() {
				if !success {
					moveQueue.Pop()
				}
			}()
		}
	}

	return false
}

// promptForInput prompts the user for input and returns the raw string
func promptForInput(line string) string {
	fmt.Println(line)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return ""
	}
	return strings.TrimSpace(input)
}

func parseDimensions(input string) (int, int) {
	parts := strings.Split(input, " ")
	if len(parts) != 2 {
		return 0, 0
	}
	x, err := strconv.Atoi(parts[0])
	if err != nil {
		panic(err)
	}
	y, err := strconv.Atoi(parts[1])
	if err != nil {
		panic(err)
	}
	return x, y
}

func parseRow(width int, input string, rowIndex int) []*GridSquare {
	row := make([]*GridSquare, width)
	input = strings.TrimSpace(input)
	parts := strings.Split(input, " ")
	blankIndices := make(map[int]bool)
	if input != "" {
		for _, part := range parts {
			num, err := strconv.Atoi(part)
			if err != nil {
				panic(err)
			}
			blankIndices[num-1] = true
		}
	}
	for i := 0; i < width; i++ {
		if _, ok := blankIndices[i]; !ok {
			row[i] = &GridSquare{X: i, Y: rowIndex, Restriction: &Restriction{Type: RestrictionTypeNone}}
		}
	}
	return row
}

// parseRestrictedRegion parses a restricted region from the input string and assigns the restriction to the grid squares it affects
func parseRestrictedRegion(input string, grid [][]*GridSquare) {
	parts := strings.Split(input, " ")
	typ := RestrictionType(parts[0])
	restriction := &Restriction{Type: typ}
	parts = parts[1:]
	switch typ {
	case "gt":
		arg, err := strconv.Atoi(parts[0])
		if err != nil {
			panic(err)
		}
		restriction.Arg = arg
		parts = parts[1:]
	case "lt":
		arg, err := strconv.Atoi(parts[0])
		if err != nil {
			panic(err)
		}
		restriction.Arg = arg
		parts = parts[1:]
	case "sum":
		arg, err := strconv.Atoi(parts[0])
		if err != nil {
			panic(err)
		}
		restriction.Arg = arg
		parts = parts[1:]
	case "eq":
		restriction.Type = RestrictionTypeEqual
		parts = parts[1:]
	}
	for i := 0; i < len(parts); i += 2 {
		x, err := strconv.Atoi(parts[i])
		if err != nil {
			panic(err)
		}
		y, err := strconv.Atoi(parts[i+1])
		if err != nil {
			panic(err)
		}
		x--
		y--
		fmt.Println("Adding restriction to grid square", x, y)
		grid[y][x].Restriction = restriction
		restriction.NumSquaresAffected++
	}
}

func buildGraph(grid [][]*GridSquare) {
	for y := 0; y < len(grid); y++ {
		for x := 0; x < len(grid[y]); x++ {
			if grid[y][x] == nil {
				continue
			}
			if y > 0 {
				grid[y][x].TopNeighbor = grid[y-1][x]
			}
			if y < len(grid)-1 {
				grid[y][x].BottomNeighbor = grid[y+1][x]
			}
			if x > 0 {
				grid[y][x].LeftNeighbor = grid[y][x-1]
			}
			if x < len(grid[y])-1 {
				grid[y][x].RightNeighbor = grid[y][x+1]
			}
		}
	}
}

func parseDominoes(input string) DominoSet {
	dominoes := make(DominoSet, 0)
	parts := strings.Split(input, " ")
	for i := 0; i < len(parts); i += 2 {
		v1, err := strconv.Atoi(parts[i])
		if err != nil {
			panic(err)
		}
		v2, err := strconv.Atoi(parts[i+1])
		if err != nil {
			panic(err)
		}
		dominoes = append(dominoes, &Domino{Square1Value: v1, Square2Value: v2})
	}
	return dominoes
}

func pickEmptySquare(grid [][]*GridSquare) *GridSquare {
	// find all blank squares with a restriction
	blankSquares := make([]*GridSquare, 0)
	for y := 0; y < len(grid); y++ {
		for x := 0; x < len(grid[y]); x++ {
			if grid[y][x] == nil {
				continue
			}
			if grid[y][x].DominoAssigned == nil && grid[y][x].Restriction != nil {
				blankSquares = append(blankSquares, grid[y][x])
			}
		}
	}

	// First try to find one with a single-square sum restriction
	for _, square := range blankSquares {
		if square.Restriction.Type == RestrictionTypeSumsTo && square.Restriction.NumSquaresAffected == 1 {
			return square
		}
	}
	// If none exists, try to find an equal restriction that is already partially filled
	for _, square := range blankSquares {
		if square.Restriction.Type == RestrictionTypeEqual && square.Restriction.Arg != -1 {
			return square
		}
	}
	// Next, look for a gt or lt restriction
	for _, square := range blankSquares {
		if square.Restriction.Type == RestrictionTypeGreaterThan || square.Restriction.Type == RestrictionTypeLessThan {
			return square
		}
	}
	// Give up on optimizing, just return the first blank square
	return blankSquares[0]
}
