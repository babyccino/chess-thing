package board

import "fmt"

type Position struct {
	X int8
	Y int8
}
type Vector = Position

func (pos *Position) Print() {
	fmt.Printf("Position{x: %d, y: %d}\n", pos.X, pos.Y)
}
func (pos *Position) Add(other Position) Position {
	return Position{pos.X + other.X, pos.Y + other.Y}
}
func (pos *Position) AddMult(other Position, mult int8) Position {
	return Position{pos.X + mult*other.X, pos.Y + mult*other.Y}
}
func (pos *Position) Diff(other Position) Position {
	return pos.AddMult(other, -1)
}

func positionToIndex(pos Position) int8 {
	return pos.X + 8*pos.Y
}

func indexToPosition(index int8) Position {
	y := index / 8
	x := index % 8
	return Position{x, y}
}

func (pos *Position) AddInBounds(other Position) (Position, bool) {
	newX := pos.X + other.X
	newY := pos.Y + other.Y

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
	DownLeftVec  = Vector{1, -1}
	UpLeftVec    = Vector{-1, -1}
	UpRightVec   = Vector{-1, 1}
	UpVec        = Vector{-1, 0}
	DownVec      = Vector{1, 0}
	LeftVec      = Vector{0, -1}
	RightVec     = Vector{0, 1}
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

func directionToVec(dir Direction) Vector {
	return directionArray[dir]
}
