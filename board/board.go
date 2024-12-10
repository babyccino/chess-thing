package board

import (
	"errors"
	"fmt"
	"strconv"
)

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

	ErrorPiece Piece = -1
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

type Check = int8

const (
	NoCheck Check = iota

	WhiteCheck
	WhiteDoubleCheck

	BlackCheck
	BlackDoubleCheck
)

type CheckState struct {
	check Check
	from  Position
}

func defaultCheckState() CheckState {
	return CheckState{check: NoCheck, from: Position{}}
}

func (state *CheckState) thingyWhite() bool {
	return debug || state.check == NoCheck || state.check == BlackCheck
}
func (state *CheckState) thingyBlack() bool {
	return debug || state.check == NoCheck || state.check == WhiteCheck
}
func (state *CheckState) promote(colour Colour) error {
	// TODO
	if state.check == NoCheck {
		if colour == White {
			state.check = WhiteCheck
			return nil
		}
		if colour == Black {
			state.check = BlackCheck
			return nil
		}
	}

	if state.check == WhiteCheck || state.check == BlackCheck {
		state.check += 1
		return nil
	}

	if state.check == WhiteDoubleCheck || state.check > BlackDoubleCheck {
		return errors.New("a third piece checked an already double checked king")
	}

	return nil
}

type BoardState struct {
	state       [64]Piece
	check       CheckState
	moveCounter uint16
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
	return &BoardState{state: state, check: defaultCheckState(), moveCounter: 0}
}

func (board *BoardState) ToMove() Colour {
	if board.moveCounter%2 == 0 {
		return White
	}
	return Black
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

type Colour = bool

const (
	White Colour = false
	Black        = true
)

func amBeingAttacked(king *Position, piece Piece, white Colour, piecePosition Position, diagonal bool) bool {
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
		diff := king.Diff(piecePosition)
		return diff == Position{0, -1} || diff == Position{-1, 0}
	} else if diagonal {
		return piece.IsDiagonalAttacker()
	} else {
		return piece.IsStraightLongAttacker()
	}
}

func (board *BoardState) checkOtherPieceChecks(wKing *Position, bKing *Position, check *CheckState) (*CheckState, error) {
	for i, vec := range directionArray[:Knight1] {
		diagonal := i <= int(UpRight)
		if check.thingyWhite() {
			piece, piecePosition := board.checkInDirection(vec, wKing)
			if amBeingAttacked(wKing, piece, White, piecePosition, diagonal) {
				err := check.promote(White)
				if err != nil {
					return nil, err
				}

				if check.check == WhiteDoubleCheck {
					return check, nil
				}
			}
		}

		if check.thingyBlack() {
			piece, piecePosition := board.checkInDirection(vec, bKing)
			if amBeingAttacked(bKing, piece, Black, piecePosition, diagonal) {
				err := check.promote(Black)
				if err != nil {
					return nil, err
				}

				if check.check == BlackDoubleCheck {
					return check, nil
				}
			}
		}
	}

	return check, nil
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

var pieceToFenArr = [...]byte{
	'k',
	'q',
	'b',
	'n',
	'p',
	'r',
	'K',
	'Q',
	'B',
	'N',
	'P',
	'R',
}

func (board *BoardState) Fen() string {
	counter := 0
	ret := ""
	for index, piece := range board.state {
		if piece == Clear {
			counter += 1
		} else if counter != 0 {
			ret += string('0' + counter)
			counter = 0
		}

		if piece > Clear {
			ret += string(pieceToFenArr[piece])
		}

		if index%8 == 7 {
			if counter != 0 {
				ret += string('0' + counter)
			}
			if index != len(board.state)-1 {
				ret += "/"
			}
			counter = 0
		}
	}

	if board.ToMove() == White {
		ret += " w"
	} else {
		ret += " b"
	}

	ret += fmt.Sprintf(" %d", board.moveCounter)

	return ret
}

const fen string = "KRBPP3/RQKP4/KBP5/PP5p/6pp/P4pbk/4pkqr/3ppbrk w 1"

func getPiece(char rune) Piece {
	switch char {
	case 'k':
		return WKing
	case 'q':
		return WQueen
	case 'b':
		return WBishop
	case 'n':
		return WKnight
	case 'p':
		return WPawn
	case 'r':
		return WRook
	case 'K':
		return BKing
	case 'Q':
		return BQueen
	case 'B':
		return BBishop
	case 'N':
		return BKnight
	case 'P':
		return BPawn
	case 'R':
		return BRook
	case '/':
		return Clear
	default:
		return ErrorPiece
	}

}

func ParseFen(fen string) (*BoardState, error) {
	state := [64]Piece{}
	stateIndex := 0
	rowIndex := 0

	boardStrLen := 0
	for strIndex, char := range fen {

		if stateIndex == 64 {
			if char != ' ' {
				errorStr := fmt.Sprintf("space not found at end of pieces")
				return nil, errors.New(errorStr)
			}

			boardStrLen = strIndex + 1
			break
		}

		if char >= '1' && char <= '8' {
			delta := int(char - '0')
			stateIndex += delta
			rowIndex += delta

			if rowIndex > 7 {
				errorStr := fmt.Sprintf("unexpected character found: %b, row index %d: ", char, rowIndex)
				return nil, errors.New(errorStr)
			}

			continue
		}

		piece := getPiece(char)

		if piece == Clear {
			if stateIndex%8 == 7 {
				stateIndex += 1
				rowIndex = 0
				continue
			}

			errorStr := fmt.Sprintf("/ found in wrong place %d", stateIndex)
			return nil, errors.New(errorStr)
		} else if piece == ErrorPiece {
			errorStr := fmt.Sprintf("unexpected character found: %b ", char)
			return nil, errors.New(errorStr)
		} else {
			state[stateIndex] = piece
		}

		stateIndex += 1
		rowIndex += 1

		if rowIndex > 7 {
			errorStr := fmt.Sprintf("row index too large: %d", rowIndex)
			return nil, errors.New(errorStr)
		}
	}

	color := White
	if fen[boardStrLen] == 'w' {
		color = White
	} else if fen[boardStrLen] == 'b' {
		color = Black
	} else {
		errorStr := fmt.Sprintf("unexpected character, should be w or b: %b", fen[boardStrLen])
		return nil, errors.New(errorStr)
	}

	boardStrLen += 1
	if fen[boardStrLen] != ' ' {
		errorStr := fmt.Sprintf("unexpected character, should be space: %b", fen[boardStrLen])
		return nil, errors.New(errorStr)
	}

	boardStrLen += 1
	moveCounter, err := strconv.ParseUint(fen[boardStrLen:], 10, 0)

	if err != nil {
		return nil, err
	}

	moveCounter = moveCounter * 2
	if !color {
		moveCounter += 1
	}

	return &BoardState{state: state, check: CheckState{}, moveCounter: uint16(moveCounter)}, nil
}
