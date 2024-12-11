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

var pieceToStrArr = [...]rune{
	' ',
	'♔',
	'♕',
	'♗',
	'♘',
	'♙',
	'♖',
	'♚',
	'♛',
	'♝',
	'♞',
	'♟',
	'♜',
}

func (piece Piece) ToString() string {
	return string(pieceToStrArr[piece])
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

func CheckToString(check Check) string {
	switch check {
	case NoCheck:
		return "no check"
	case WhiteCheck:
		return "white check"
	case WhiteDoubleCheck:
		return "white double check"
	case BlackCheck:
		return "black check"
	case BlackDoubleCheck:
		return "black double check"
	}
	return ""
}

type CheckState struct {
	Check Check
	From  Position
}

func defaultCheckState() CheckState {
	return CheckState{Check: NoCheck, From: Position{}}
}

func (state *CheckState) ToString() string {
	return fmt.Sprintf("check: %s, position %s",
		CheckToString(state.Check), state.From.ToString())
}
func (state *CheckState) thingyWhite() bool {
	return debug || state.Check == NoCheck || state.Check == BlackCheck
}
func (state *CheckState) thingyBlack() bool {
	return debug || state.Check == NoCheck || state.Check == WhiteCheck
}
func (state *CheckState) promote(colour Colour) error {
	// TODO
	if state.Check == NoCheck {
		if colour == White {
			state.Check = WhiteCheck
			return nil
		}
		if colour == Black {
			state.Check = BlackCheck
			return nil
		}
	}

	if state.Check == WhiteCheck || state.Check == BlackCheck {
		state.Check += 1
		return nil
	}

	if state.Check == WhiteDoubleCheck || state.Check > BlackDoubleCheck {
		return errors.New("a third piece checked an already double checked king")
	}

	return nil
}

type BoardState struct {
	State              [64]Piece
	Check              CheckState
	MoveCounter        uint16
	CaptureMoveCounter uint16
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
	return &BoardState{
		State:              state,
		Check:              defaultCheckState(),
		MoveCounter:        0,
		CaptureMoveCounter: 0,
	}
}

func (board *BoardState) ToMove() Colour {
	if board.MoveCounter%2 == 0 {
		return White
	}
	return Black
}

func (board *BoardState) ToString() string {
	str := " A B C D E F G H  \n\n "
	for i, piece := range board.State {
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
	return board.State[positionToIndex(pos)]
}
func (board *BoardState) SetSquare(pos Position, piece Piece) {
	board.State[positionToIndex(pos)] = piece
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

func (board *BoardState) GetKingPositions() (wKing *Position, bKing *Position) {
	for i, piece := range board.State {
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

func (board *BoardState) CheckKnightChecks(wKing *Position, bKing *Position, findDouble bool) (*CheckState, error) {
	// check the knight checks first because a double knight check is not possible
	check := NoCheck
	from := Position{}
	for _, vec := range directionArray[Knight1:] {
		pos, inBounds := wKing.AddInBounds(vec)
		if inBounds && board.GetSquare(pos) == BKnight {
			if !findDouble {
				return &CheckState{WhiteCheck, pos}, nil
			}

			if check != NoCheck {
				err := fmt.Errorf("weird board state reached, check: %s\n\n%s",
					CheckToString(check), board.ToString())
				return nil, err
			}

			check = WhiteCheck
			from = pos
		}

		pos, inBounds = bKing.AddInBounds(vec)
		if inBounds {
			fmt.Printf("checking for black %s, piece%s\n",
				pos.ToString(), board.GetSquare(pos).ToString())
		}
		if inBounds && board.GetSquare(pos) == WKnight {
			if !findDouble {
				return &CheckState{BlackCheck, pos}, nil
			}

			if check != NoCheck {
				err := fmt.Errorf("weird board state reached, check: %s\n\n%s",
					CheckToString(check), board.ToString())
				return nil, err
			}

			check = BlackCheck
			from = pos
		}
	}

	return &CheckState{check, from}, nil
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

				if check.Check == WhiteDoubleCheck {
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

				if check.Check == BlackDoubleCheck {
					return check, nil
				}
			}
		}
	}

	return check, nil
}

func (board *BoardState) UpdateCheckState(findErr bool) error {
	wKing, bKing := board.GetKingPositions()

	check, err := board.CheckKnightChecks(wKing, bKing, findErr)
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
	'/',
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

func rowIntToByte(row int) byte {
	return byte('0' + row)
}

func (board *BoardState) Fen() string {
	counter := 0
	ret := ""
	for index, piece := range board.State {
		if piece == Clear {
			counter += 1
		} else if counter != 0 {
			ret += string(rowIntToByte(counter))
			counter = 0
		}

		if piece > Clear {
			ret += string(pieceToFenArr[piece])
		}

		if index%8 == 7 {
			if counter != 0 {
				ret += string(rowIntToByte(counter))
			}
			if index != len(board.State)-1 {
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

	ret += fmt.Sprintf(" %d", board.MoveCounter)

	return ret
}

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

			// fmt.Printf("%s, %d, %d, delta: %d\n", string(char), stateIndex, rowIndex, delta)

			continue
		}

		piece := getPiece(char)

		if piece == Clear {
			if stateIndex%8 != 0 {
				errorStr := fmt.Sprintf("/ found in wrong place stateIndex: %d, rowIndex: %d",
					stateIndex, rowIndex)
				return nil, errors.New(errorStr)
			}

			if rowIndex != 8 {
				errorStr := fmt.Sprintf("/ found in wrong place stateIndex: %d, rowIndex: %d",
					stateIndex, rowIndex)
				return nil, errors.New(errorStr)
			}

			// fmt.Printf("%s, %d, %d\n", string(char), stateIndex, rowIndex)
			rowIndex = 0
			continue
		} else if piece == ErrorPiece {
			errorStr := fmt.Sprintf("unexpected character found: %s ", string(char))
			return nil, errors.New(errorStr)
		} else {
			state[stateIndex] = piece
		}

		// fmt.Printf("%s, %d, %d\n", string(char), stateIndex, rowIndex)

		stateIndex += 1
		rowIndex += 1

		if rowIndex > 8 {
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
		errorStr := fmt.Sprintf("unexpected character, should be w or b: %s",
			string(fen[boardStrLen]))
		return nil, errors.New(errorStr)
	}

	boardStrLen += 1
	if fen[boardStrLen] != ' ' {
		errorStr := fmt.Sprintf("unexpected character, should be space: %s",
			string(fen[boardStrLen]))
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

	return &BoardState{State: state, Check: CheckState{}, MoveCounter: uint16(moveCounter)}, nil
}
