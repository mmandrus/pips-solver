package main

import "fmt"

// Gridsquare represents a single square on the grid and all the metadata associated with it, including its logical restrictions
type GridSquare struct {
	X int
	Y int

	DominoAssigned *Domino
	PipValue       int

	Restriction *Restriction

	TopNeighbor    *GridSquare
	BottomNeighbor *GridSquare
	LeftNeighbor   *GridSquare
	RightNeighbor  *GridSquare
}

// Restriction represents a restriction on the grid. Each square maintains a list of its own restrictions
type Restriction struct {
	Type RestrictionType
	// gt/lt: target value
	// eq: initial target value, will be modified by the domino assignment
	// sum: blank initial value (-1), will be modified by the domino assignment
	Arg int

	NumSquaresLeft int
}

func (r *Restriction) Check(value int, numSquares int) bool {
	switch r.Type {
	case RestrictionTypeNone:
		return true
	case RestrictionTypeGreaterThan:
		return value > r.Arg
	case RestrictionTypeLessThan:
		return value < r.Arg
	case RestrictionTypeEqual:
		return r.Arg == -1 || value == r.Arg
	case RestrictionTypeSumsTo:
		if r.Arg-value < 0 {
			return false
		}
		if r.NumSquaresLeft == numSquares && value != r.Arg {
			return false
		}
		return true
	}
	return false
}

type RestrictionType string

const (
	RestrictionTypeNone        RestrictionType = "none"
	RestrictionTypeGreaterThan RestrictionType = "gt"
	RestrictionTypeLessThan    RestrictionType = "lt"
	RestrictionTypeEqual       RestrictionType = "eq"
	RestrictionTypeSumsTo      RestrictionType = "sum"
)

// Domino represents a single domino and all the metadata associated with it, including its value and its rotation
// A domino has no awareness of where it sits on the grid, just its orientation and whether it is available for assignment
type Domino struct {
	Square1Value int
	Square2Value int

	IsAssigned bool

	rotation int // value % 4 is the number of 90 degree rotations
}

func (d *Domino) GetRotation() int {
	return d.rotation % 4
}

// Possible move
func (d *Domino) Rotate90DegreesClockwise() {
	d.rotation++
}

// Undo action for Rotate90DegreesClockwise
func (d *Domino) Rotate90DegreesCounterClockwise() {
	d.rotation--
}

// Possible move (and its own undo action)
func (d *Domino) Swap() {
	d.Square1Value, d.Square2Value = d.Square2Value, d.Square1Value
}

// Possible move, returns its own undo action depending on what was done during placement
func (d *Domino) Assign(s *GridSquare, neighbor *GridSquare) func() {
	s.DominoAssigned = d
	neighbor.DominoAssigned = d
	wasBlank, neighborWasBlank := false, false
	if s.Restriction.Type == RestrictionTypeEqual {
		if s.Restriction.Arg == -1 {
			s.Restriction.Arg = d.Square1Value
			wasBlank = true
		}
	}
	if neighbor.Restriction.Type == RestrictionTypeEqual {
		if neighbor.Restriction.Arg == -1 {
			neighbor.Restriction.Arg = d.Square2Value
			neighborWasBlank = true
		}
	}
	if s.Restriction.Type == RestrictionTypeSumsTo {
		s.Restriction.Arg -= d.Square1Value
		s.Restriction.NumSquaresLeft--
	}
	if neighbor.Restriction.Type == RestrictionTypeSumsTo {
		neighbor.Restriction.Arg -= d.Square2Value
		neighbor.Restriction.NumSquaresLeft--
	}

	d.IsAssigned = true

	undoFunc := func() {
		s.DominoAssigned = nil
		d.IsAssigned = false
		neighbor.DominoAssigned = nil
		if wasBlank {
			s.Restriction.Arg = -1
		}
		if neighborWasBlank {
			neighbor.Restriction.Arg = -1
		}
		if s.Restriction.Type == RestrictionTypeSumsTo {
			s.Restriction.Arg += d.Square1Value
			s.Restriction.NumSquaresLeft++
		}
		if neighbor.Restriction.Type == RestrictionTypeSumsTo {
			neighbor.Restriction.Arg += d.Square2Value
			neighbor.Restriction.NumSquaresLeft++
		}
	}

	return undoFunc
}

func (d *Domino) TryAssign(s *GridSquare) (bool, func()) {
	// Fail-fast: ensure that the domino doesn't overlap with a placed domino on an adjacent square or go out of bounds
	var neighbor *GridSquare
	switch d.GetRotation() {
	case 0: // horizontal going to right (starting state)
		neighbor = s.RightNeighbor
	case 1: // vertical going down
		neighbor = s.BottomNeighbor
	case 2: // horizontal going to left
		neighbor = s.LeftNeighbor
	case 3: // vertical going up
		neighbor = s.TopNeighbor
	}
	if neighbor == nil {
		return false, nil
	}
	if neighbor.DominoAssigned != nil {
		return false, nil
	}

	// We now know the two squares that placing this domino would affect, check both of their restrictions
	// If they are the same restriction, we can check this easily in one operation
	if s.Restriction == neighbor.Restriction {
		if s.Restriction.Type == RestrictionTypeEqual {
			if d.Square1Value != d.Square1Value {
				return false, nil
			}
			if !s.Restriction.Check(d.Square1Value, 1) {
				return false, nil
			}
		}
		if s.Restriction.Type == RestrictionTypeSumsTo {
			sumOfDomino := d.Square1Value + d.Square2Value
			if !s.Restriction.Check(sumOfDomino, 2) {
				return false, nil
			}
		}
	} else {
		// Only check neighbor, we already checked the current square during candidate selection
		if !neighbor.Restriction.Check(d.Square2Value, 1) {
			return false, nil
		}
	}

	undoFunc := d.Assign(s, neighbor)

	return true, undoFunc
}

// A move represents rotating a domino about its first square, swapping the first and second square, or assigning a domino
type Move struct {
	Label      string
	Domino     *Domino
	GridSquare *GridSquare
	MoveType   MoveType
	UndoFunc   func()

	Pruned bool
}

type MoveType string

const (
	MoveTypeRotate MoveType = "rotate"
	MoveTypeSwap   MoveType = "swap"
	MoveTypeAssign MoveType = "assign"
)

type MoveQueue []*Move

func (q *MoveQueue) TryPush(m *Move) bool {
	switch m.MoveType {
	case MoveTypeRotate:
		m.Domino.Rotate90DegreesClockwise()
		m.UndoFunc = m.Domino.Rotate90DegreesCounterClockwise
		*q = append(*q, m)
		return true
	case MoveTypeSwap:
		m.Domino.Swap()
		m.UndoFunc = m.Domino.Swap
		*q = append(*q, m)
		return true
	case MoveTypeAssign:
		success, undoFunc := m.Domino.TryAssign(m.GridSquare)
		if success {
			m.UndoFunc = undoFunc
			*q = append(*q, m)
			return true
		}
	}

	return false
}

func (q *MoveQueue) Pop() *Move {
	if len(*q) == 0 {
		return nil
	}
	m := (*q)[len(*q)-1]
	*q = (*q)[:len(*q)-1]
	m.UndoFunc() // Undo the move
	return m
}

func (q *MoveQueue) String() string {
	s := "Move queue:\n"
	for _, m := range *q {
		if !m.Pruned {
			s += fmt.Sprintf("  - %s\n", m.Label)
		}
	}
	return s
}

// My code structure is funky and allows for quadruple rotates not resulting not in a placement
// I am too lazy to change things right now so instead lets just prune them
// If there are four consecutive rotates of the same domino followed NOT by a placement move of that domino, remove all four rotates
func (q *MoveQueue) PruneUselessMoves() {
	firstRotateIndex := 0
	firstRotateMove := (*q)[0]
	for i := 1; i < len(*q); i++ {
		if (*q)[i].MoveType == MoveTypeRotate {
			if firstRotateMove.Domino != (*q)[i].Domino {
				firstRotateIndex = i
				firstRotateMove = (*q)[i]
			} else {
				if i-firstRotateIndex == 3 && !((*q)[i+1].MoveType == MoveTypeAssign && (*q)[i+1].Domino == firstRotateMove.Domino) {
					for j := firstRotateIndex; j <= i; j++ {
						(*q)[j].Pruned = true
					}
					firstRotateIndex = i + 1
					firstRotateMove = (*q)[firstRotateIndex]
				}
			}
		}
	}
}

type DominoCandidate struct {
	Domino       *Domino
	isLeftMatch  bool
	isRightMatch bool
}

type DominoSet []*Domino

func (ds *DominoSet) FindAvailableCandidates(square *GridSquare) []*DominoCandidate {
	available := make([]*DominoCandidate, 0)
	for _, domino := range *ds {
		if domino.IsAssigned {
			continue
		}

		leftMatch := square.Restriction.Check(domino.Square1Value, 1)
		rightMatch := square.Restriction.Check(domino.Square2Value, 1)
		if !leftMatch && !rightMatch {
			continue
		}
		available = append(available, &DominoCandidate{Domino: domino, isLeftMatch: leftMatch, isRightMatch: rightMatch})
	}
	return available
}

func (ds *DominoSet) HasUnassigned() bool {
	for _, domino := range *ds {
		if !domino.IsAssigned {
			return true
		}
	}
	return false
}
