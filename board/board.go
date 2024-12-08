package board

import (
	"errors"
	"fmt"
)

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

func (pos *Position) AddInBounds(other Position) (Position, bool) {
	newX := pos.X + other.X
	newY := pos.Y + other.Y

	if newX < 0 || newX >= 8 || newY < 0 || newY >= 8 {
		return Position{}, false
	}

	return Position{newX, newY}, true
}

type Piece int8

const (
	Clear Piece = iota
	WKing
	WQueen
	WBishop
	WKnight
	WPawn
	WRook
	BKing
	BQueen
	BBishop
	BKnight
	BPawn
	BRook
)

func (piece *Piece) IsWhite() bool {
	return *piece != Clear && *piece <= BRook
}
func (piece *Piece) IsBlack() bool {
	return *piece >= BKing
}
func (piece *Piece) IsDiagonalAttacker() bool {
	switch *piece {
	case WQueen:
	case WBishop:
	case BQueen:
	case BBishop:
		return true
	}
	return false
}
func (piece *Piece) IsStraightLongAttacker() bool {
	switch *piece {
	case WRook:
	case BRook:
		return true
	}
	return false
}

var pieceToStrArr = [...]string{
	" ",
	"♔",
	"♕",
	"♗",
	"♘",
	"♙",
	"♖",
	"♚",
	"♛",
	"♝",
	"♞",
	"♟",
	"♜",
}

func (piece *Piece) ToString() string {
	return pieceToStrArr[*piece]
}

// func isDiagonalAttacking(piece Piece) bool {
// 	return direction <= UpRight
// }
// func isStraight(direction Direction) bool {
// 	return direction >= Up && direction <= Right
// }
// func isKnight(direction Direction) bool {
// 	return direction >= Knight1
// }

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

var directionArray = [...]Vector{
	{1, 1}, {1, -1},
	{-1, -1}, {-1, 1},
	{-1, 0}, {1, 0},
	{0, -1}, {0, 1},

	{1, 2}, {2, 1}, // Knight up-right and right-up
	{2, -1}, {1, -2}, // Knight right-down and down-right
	{-1, -2}, {-2, -1}, // Knight down-left and left-down
	{-2, 1}, {-1, 2}, // Knight left-up and up-left
}

func directionToVec(dir Direction) Vector {
	return directionArray[dir]
}

type Check = int8

const (
	NoCheck Check = iota
	WhiteCheck
	BlackCheck
	WhiteDoubleCheck
	BlackDoubleCheck
)

type CheckState struct {
	check Check
	from  Position
}

func (state *CheckState) thingyWhite() bool {
	return debug || check.check == NoCheck || check.check == BlackCheck
}
func (state *CheckState) thingyBlack() bool {
	return debug || check.check == NoCheck || check.check == WhiteCheck
}

type BoardState struct {
	state [64]Piece
	check Check
}

func positionToIndex(pos Position) int8 {
	return pos.X + 8*pos.Y
}

func indexToPosition(index int8) Position {
	y := index / 8
	x := index % 8
	return Position{x, y}
}

func NewBoard() *BoardState {
	state := [64]Piece{
		BKing, BRook, BBishop, BPawn, BPawn, Clear, Clear, Clear,
		BRook, BQueen, BKnight, BPawn, Clear, Clear, Clear, Clear,
		BKnight, BBishop, BPawn, Clear, Clear, Clear, Clear, Clear,
		BPawn, BPawn, Clear, Clear, Clear, Clear, Clear, WPawn,
		BPawn, Clear, Clear, Clear, Clear, Clear, WPawn, WPawn,
		Clear, Clear, Clear, Clear, Clear, WPawn, WBishop, WKnight,
		Clear, Clear, Clear, Clear, WPawn, WKnight, WQueen, WRook,
		Clear, Clear, Clear, WPawn, WPawn, WBishop, WRook, WKing,
	}
	return &BoardState{state: state, check: NoCheck}
}

func (board *BoardState) ToString() string {
	str := " A B C D E F G H  \n\n "
	for i, piece := range board.state {
		str += piece.ToString() + " "
		if i%8 == 7 {
			str += fmt.Sprintf(" %d\n ", i/8+1)
		}
	}
	str += "\n"
	return str
}

func (board *BoardState) Print() {
	println(board.ToString())
}

func (board *BoardState) GetSquare(pos Position) Piece {
	return board.state[positionToIndex(pos)]
}
func (board *BoardState) SetSquare(pos Position, piece Piece) {
	board.state[positionToIndex(pos)] = piece
}

func (board *BoardState) Move(start Position, end Position) error {
	if start == end {
		return errors.New("positions are same")
	}
	if start.X >= 8 || end.Y >= 8 {
		return errors.New("move out of bounds")
	}

	board.SetSquare(end, board.GetSquare(start))
	board.SetSquare(start, Clear)

	return nil
}

func (board *BoardState) getKingPositions() (wKing *Position, bKing *Position) {
	for i, piece := range board.state {
		if piece == WKing {
			newKing := indexToPosition(int8(i))
			wKing = &newKing
		}
		if piece == BKing {
			newKing := indexToPosition(int8(i))
			bKing = &newKing
		}
	}

	if wKing == nil || bKing == nil {
		panic("no king???")
	}

	return wKing, bKing
}

const debug = true

func (board *BoardState) checkKnightChecks(wKing *Position, bKing *Position) (*CheckState, error) {
	// check the knight checks first because a double knight check is not possible
	check := NoCheck
	for _, vec := range directionArray[Knight1:] {
		pos, inBounds := wKing.AddInBounds(vec)
		if inBounds && board.GetSquare(pos) == BKnight {
			check = WhiteCheck

			if !debug {
				return &CheckState{check, pos}, nil
			} else if check != NoCheck {
				return &CheckState{}, errors.New("weird board state reached\n\n" + board.ToString())
			}
		}

		pos, inBounds = bKing.AddInBounds(vec)
		if inBounds && board.GetSquare(pos) == WKnight {
			check = BlackCheck

			if !debug {
				return &CheckState{check, pos}, nil
			} else if check != NoCheck {
				return &CheckState{}, errors.New("weird board state reached\n\n" + board.ToString())
			}
		}
	}

	return &CheckState{}, nil
}

func (board *BoardState) checkInDirection(vec Vector, pos *Position) (Piece, Position) {
	posCopy := *pos
	for {
		var inBounds bool
		posCopy, inBounds = posCopy.AddInBounds(vec)
		if !inBounds {
			return Clear, posCopy
		}

		piece := board.GetSquare(posCopy)
		if piece != Clear {
			return piece, posCopy
		}
	}
}

func amBeingAttacked(king *Position, piece Piece, white bool, piecePosition Position, vec Vector, diagonal bool) bool {
	if piece == Clear {
		return false
	}
	if white && piece.IsWhite() {
		return false
	}
	if !white && piece.IsBlack() {
		return false
	}

	if piece == BPawn {
		diff := king.Diff(vec)
		return diff == Position{0, -1} || diff == Position{-1, 0}
	}
	if diagonal {
		return piece.IsDiagonalAttacker()
	} else {
		return piece.IsStraightLongAttacker()
	}
}

func (board *BoardState) checkOtherPieceChecksImpl(wKing *Position, bKing *Position, check *CheckState) (*CheckState, error) {
	for i, vec := range directionArray[:Knight1] {
		diagonal := i <= int(UpRight)
		if check.thingyWhite() {
			piece, piecePosition := board.checkInDirection(vec, wKing)
			if amBeingAttacked(wKing, piece, true, piecePosition, vec, diagonal) {
				// TODO
				// no check => check etc.
				// check => doubleCheck etc.
				// if you get to doublecheck return
				// if debug just keep going ay
			}
		}

		if check.thingyBlack() {
			// TODO
		}
	}

	return &CheckState{}, nil
}

func (board *BoardState) checkOtherPieceChecks(wKing *Position, bKing *Position, check *CheckState) (*CheckState, error) {
	return board.checkOtherPieceChecksImpl(wKing, bKing, check)
}

func (board *BoardState) UpdateCheckState() error {
	wKing, bKing := board.getKingPositions()

	check, err := board.checkKnightChecks(wKing, bKing)
	if err != nil {
		return err
	}
	check, err = board.checkOtherPieceChecks(wKing, bKing, check)
	if err != nil {
		return err
	}

	return nil
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

	parsedFile := int8(file - 'A')
	parsedRank := int8(rank - '1')
	return Position{X: parsedFile, Y: parsedRank}, nil
}
