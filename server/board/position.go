package board

import (
	"errors"
	"fmt"
	"testing"
)

type Position struct {
	X int8
	Y int8
}
type Vector = Position

func (pos *Position) String() string {
	return fmt.Sprintf("Position{x: %d, y: %d}", pos.X, pos.Y)
}

func (pos *Position) Print() {
	fmt.Print(pos.String() + "\n")
}

func (pos *Position) CoordsString() string {
	bytes := []byte{byte('H' - pos.X), byte('1' + pos.Y)}
	return string(bytes)
}

func (pos *Position) Add(other Position) Position {
	return Position{pos.X + other.X, pos.Y + other.Y}
}

func (pos *Position) AddMult(other Position, mult int8) Position {
	return Position{pos.X + mult*other.X, pos.Y + mult*other.Y}
}

func (pos *Position) Mult(mult int8) Position {
	return Position{pos.X * mult, pos.Y * mult}
}

func (pos *Position) Diff(other Position) Position {
	return Position{pos.X - other.X, pos.Y - other.Y}
}

func positionToIndex(pos Position) int8 {
	return pos.X + 8*pos.Y
}

func IndexToPosition(index int) Position {
	y := index / 8
	x := index % 8
	return Position{int8(x), int8(y)}
}

func StringToPosition(str string) (Position, error) {
	if len(str) != 2 {
		return Position{}, errors.New("string must be of length 2")
	}

	file := str[0]
	rank := str[1]
	if file < 'A' || file > 'H' {
		return Position{}, errors.New("rank out of bounds")
	}
	if rank < '1' || rank > '8' {
		return Position{}, errors.New("rank out of bounds")
	}

	parsedFile := int8('H' - file)
	parsedRank := int8(rank - '1')
	return Position{X: parsedFile, Y: parsedRank}, nil
}

func (pos *Position) AddInBounds(other Position) (Position, bool) {
	return pos.AddInBoundsMult(other, 1)
}

func (pos *Position) AddInBoundsMult(other Position, mult int8) (Position, bool) {
	newX := pos.X + mult*other.X
	newY := pos.Y + mult*other.Y

	if newX < 0 || newX >= 8 || newY < 0 || newY >= 8 {
		return Position{}, false
	}

	return Position{newX, newY}, true
}

type Direction uint8

const (
	DownRight Direction = iota
	DownLeft
	UpLeft
	UpRight
	Up
	Down
	Left
	Right
	Knight1
	Knight2
	Knight3
	Knight4
	Knight5
	Knight6
	Knight7
	Knight8
)

var reverseDirectionLookup = []Direction{
	UpLeft,
	UpRight,
	DownRight,
	DownLeft,
	Down,
	Up,
	Right,
	Left,
	Knight8,
	Knight7,
	Knight6,
	Knight5,
	Knight4,
	Knight3,
	Knight2,
	Knight1,
}

func reverseDirection(direction Direction) Direction {
	return reverseDirectionLookup[direction]
}

func isDiagonal(direction Direction) bool {
	return direction <= UpRight
}

func isStraight(direction Direction) bool {
	return direction >= Up && direction <= Right
}

func isKnight(direction Direction) bool {
	return direction >= Knight1
}

var (
	DownRightVec = Vector{1, 1}
	DownLeftVec  = Vector{-1, 1}
	UpLeftVec    = Vector{-1, -1}
	UpRightVec   = Vector{1, -1}
	UpVec        = Vector{0, -1}
	DownVec      = Vector{0, 1}
	LeftVec      = Vector{-1, 0}
	RightVec     = Vector{1, 0}
	Knight1Vec   = Vector{1, 2}
	Knight2Vec   = Vector{2, 1}
	Knight3Vec   = Vector{2, -1}
	Knight4Vec   = Vector{1, -2}
	Knight5Vec   = Vector{-1, -2}
	Knight6Vec   = Vector{-2, -1}
	Knight7Vec   = Vector{-2, 1}
	Knight8Vec   = Vector{-1, 2}
)

var directionArray = [...]Vector{
	DownRightVec, DownLeftVec,
	UpLeftVec, UpRightVec,
	UpVec, DownVec,
	LeftVec, RightVec,

	Knight1Vec, Knight2Vec,
	Knight3Vec, Knight4Vec,
	Knight5Vec, Knight6Vec,
	Knight7Vec, Knight8Vec,
}

var (
	diagonalDirectionArray  = directionArray[:Up]
	straightDirectionArray  = directionArray[Up:Knight1]
	nonKnightDirectionArray = directionArray[:Knight1]
	knightDirectionArray    = directionArray[Knight1:]
)

func directionToVec(dir Direction) Vector {
	return directionArray[dir]
}

func AssertPositionsEqual(test *testing.T, pos1 Position, pos2 Position) {
	test.Helper()
	if pos1 != pos2 {
		test.Fatalf("expected %s, received %s",
			pos1.String(), pos2.String())
	}
}
