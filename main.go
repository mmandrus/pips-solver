package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	fmt.Println("NYT Pips Puzzle Solver")
	fmt.Println("======================")

	input := promptForInput()
	parsedInput := parseInput(input)

	fmt.Printf("Received input: %s\n", parsedInput)
}

// promptForInput prompts the user for input and returns the raw string
func promptForInput() string {
	fmt.Print("Enter puzzle input: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return ""
	}
	return input
}

// parseInput is a stub function that will parse the input string
// Currently returns the input as-is for future implementation
func parseInput(input string) string {
	// TODO: Implement parsing logic
	return input
}
