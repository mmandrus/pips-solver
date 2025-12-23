package main

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

	NumSquaresAffected int
}

func (r *Restriction) Check(value int) bool {
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
		return r.Arg-value >= 0
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
	}
	if neighbor.Restriction.Type == RestrictionTypeSumsTo {
		neighbor.Restriction.Arg -= d.Square2Value
	}

	d.IsAssigned = true

	undoFunc := func() {
		s.DominoAssigned = nil
		d.IsAssigned = false
		if wasBlank {
			s.Restriction.Arg = -1
		}
		if neighborWasBlank {
			neighbor.Restriction.Arg = -1
		}
		if s.Restriction.Type == RestrictionTypeSumsTo {
			s.Restriction.Arg += d.Square1Value
		}
		if neighbor.Restriction.Type == RestrictionTypeSumsTo {
			neighbor.Restriction.Arg += d.Square2Value
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
			if !s.Restriction.Check(d.Square1Value) {
				return false, nil
			}
		}
		if s.Restriction.Type == RestrictionTypeSumsTo {
			sumOfDomino := d.Square1Value + d.Square2Value
			if !s.Restriction.Check(sumOfDomino) {
				return false, nil
			}
		}
	} else {
		// Only check neighbor, we already checked the current square during candidate selection
		if neighbor.Restriction != nil {
			if !neighbor.Restriction.Check(d.Square2Value) {
				return false, nil
			}
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
		return true
	case MoveTypeSwap:
		m.Domino.Swap()
		m.UndoFunc = m.Domino.Swap
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

		leftMatch := square.Restriction.Check(domino.Square1Value)
		rightMatch := square.Restriction.Check(domino.Square2Value)
		if !leftMatch && !rightMatch {
			continue
		}
		available = append(available, &DominoCandidate{Domino: domino, isLeftMatch: leftMatch, isRightMatch: rightMatch})
	}
	return available
}
