package board

import (
	"errors"
	"fmt"
	"strconv"
)

type Piece uint8
type Colour = uint8

func ColourString(colour Colour) string {
	if colour == White {
		return "white"
	} else {
		return "black"
	}
}

/*
pinned piece colour clear
0      011   0      1
*/

// clear (no piece on that square) = 0b0
const (
	Clear    Piece = 0b0
	NotClear       = 0b1
)

const (
	ClearMask  Piece = 0b000001
	ColourMask Piece = 0b000010
	PieceMask        = 0b011100
	PinMask          = 0b100000
)

const (
	ColourShift uint8 = 1
	PieceShift        = 2
	PinShift          = 5
)

const (
	White Colour = 0
	Black        = 1
	None         = 2
)

const (
	King Piece = iota
	Queen
	Bishop
	Knight
	Pawn
	Rook
)

const shiftedBlack = Piece(Black) << ColourShift
const (
	WKing   Piece = NotClear | King<<PieceShift
	WQueen        = NotClear | Queen<<PieceShift
	WBishop       = NotClear | Bishop<<PieceShift
	WKnight       = NotClear | Knight<<PieceShift
	WPawn         = NotClear | Pawn<<PieceShift
	WRook         = NotClear | Rook<<PieceShift
	BKing         = NotClear | King<<PieceShift | shiftedBlack
	BQueen        = NotClear | Queen<<PieceShift | shiftedBlack
	BBishop       = NotClear | Bishop<<PieceShift | shiftedBlack
	BKnight       = NotClear | Knight<<PieceShift | shiftedBlack
	BPawn         = NotClear | Pawn<<PieceShift | shiftedBlack
	BRook         = NotClear | Rook<<PieceShift | shiftedBlack
)

func (piece Piece) Colour() Colour {
	return Colour((piece & ColourMask) >> ColourShift)
}
func (piece Piece) IsWhite() bool {
	return piece.Colour() == White
}
func (piece Piece) IsBlack() bool {
	return piece.Colour() == Black
}
func (piece Piece) PieceType() Piece {
	return (piece & PieceMask) >> PieceShift
}
func (piece Piece) IsDiagonalAttacker() bool {
	pieceType := piece.PieceType()
	return pieceType == Queen || pieceType == Bishop
}
func (piece Piece) IsStraightLongAttacker() bool {
	pieceType := piece.PieceType()
	return pieceType == Queen || pieceType == Rook
}

const shiftedPinned = 0b1 << PinShift

func (piece Piece) Pinned() Piece {
	return piece | shiftedPinned
}
func (piece Piece) IsPinned() bool {
	return (piece & PinMask) == shiftedPinned
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

func (piece Piece) Rune() rune {
	index := Piece(0)
	index += (NotClear & piece) * (1 + piece.PieceType() + Piece(6*piece.Colour()))
	return pieceToStrArr[index]
}
func (piece Piece) String() string {
	return string(piece.Rune())
}
func (piece Piece) StringDebug() string {
	pieceChar := piece.String()
	if piece == Clear {
		return pieceChar
	} else if piece.IsBlack() {
		return "black " + pieceChar
	} else {
		return "white " + pieceChar
	}
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

func checkIsWhite(check Check) bool {
	return check == WhiteCheck || check == WhiteDoubleCheck
}
func checkIsBlack(check Check) bool {
	return check >= BlackCheck
}

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

func (state *CheckState) String() string {
	return fmt.Sprintf("check: %s, position %s",
		CheckToString(state.Check), state.From.String())
}
func (state *CheckState) InCheck(colour Colour) bool {
	if colour == White {
		return debug || state.Check == NoCheck || state.Check == BlackCheck
	} else {
		return debug || state.Check == NoCheck || state.Check == WhiteCheck
	}
}
func (state *CheckState) Promote(colour Colour) error {
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

func (board *BoardState) String() string {
	str := " A B C D E F G H  \n\n "
	for i, piece := range board.State {
		str += piece.String() + " "
		if i%8 == 7 {
			str += fmt.Sprintf(" %d\n ", i/8+1)
		}
	}
	str += "\n"
	return str
}

func (board *BoardState) Print() {
	println(board.String())
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

// check stuff

func (board *BoardState) GetKingPositions() (wKing *Position, bKing *Position) {
	for i, piece := range board.State {
		if piece == WKing {
			newKing := IndexToPosition(int8(i))
			wKing = &newKing
		}
		if piece == BKing {
			newKing := IndexToPosition(int8(i))
			bKing = &newKing
		}
	}

	if wKing == nil || bKing == nil {
		panic("no king???")
	}

	return wKing, bKing
}

const debug = true

func (board *BoardState) CheckKnightChecks(
	wKing *Position, bKing *Position,
) (*CheckState, error) {
	// check the knight checks first because a double knight check is not possible
	check := NoCheck
	from := Position{}
	for _, vec := range directionArray[Knight1:] {
		pos, inBounds := wKing.AddInBounds(vec)
		if inBounds && board.GetSquare(pos) == BKnight {
			if check != NoCheck {
				err := fmt.Errorf("weird board state reached, check: %s\n\n%s",
					CheckToString(check), board.String())
				return nil, err
			}

			check = WhiteCheck
			from = pos
		}

		pos, inBounds = bKing.AddInBounds(vec)
		if inBounds && board.GetSquare(pos) == WKnight {
			if check != NoCheck {
				err := fmt.Errorf("weird board state reached, check: %s\n\n%s",
					CheckToString(check), board.String())
				return nil, err
			}

			check = BlackCheck
			from = pos
		}
	}

	return &CheckState{check, from}, nil
}

func (board *BoardState) FindInDirection(vec Vector, pos *Position) (Piece, Position) {
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

func AmBeingAttacked(
	king *Position, piece Piece, colour Colour,
	piecePosition Position, diagonal bool,
) bool {
	if piece == Clear {
		return false
	}
	if (colour == White) == piece.IsWhite() {
		return false
	}

	diff := king.Diff(piecePosition)
	if piece == BPawn {
		return colour == White && (diff == DownVec || diff == RightVec)
	} else if piece == WPawn {
		return colour == Black && (diff == UpVec || diff == LeftVec)
	} else if diagonal {
		return piece.IsDiagonalAttacker()
	} else {
		return piece.IsStraightLongAttacker()
	}
}

func (board *BoardState) otherPieceChecksImpl(
	king *Position,
	check *CheckState,
	colour Colour,
	dir Direction,
) (*CheckState, error) {
	diagonal := dir <= UpRight
	vec := directionArray[dir]

	piece, piecePosition := board.FindInDirection(vec, king)
	fmt.Printf("color: %s, piece: %s\n",
		ColourString(colour),
		piece.StringDebug())

	if piece == Clear {
		return check, nil
	}

	if AmBeingAttacked(king, piece, colour, piecePosition, diagonal) {
		// fmt.Printf("color: %t, check: %s, piece: %s\n", colour, check, piece)
		if (colour == White && checkIsBlack(check.Check)) ||
			(colour == Black && checkIsWhite(check.Check)) {
			return nil, errors.New("both white and black kings are being attacked simultaneously")
		}

		err := check.Promote(colour)
		if err != nil {
			return nil, err
		}
		check.From = piecePosition

		if check.Check == WhiteDoubleCheck {
			return check, nil
		}
	} else if piece.Colour() == colour {
		pinningPiece, pinningPiecePosition := board.FindInDirection(vec, &piecePosition)
		if AmBeingAttacked(
			king,
			pinningPiece,
			colour,
			pinningPiecePosition,
			diagonal,
		) {
			board.SetSquare(piecePosition, piece.Pinned())
		}
	}

	return check, nil
}

func (board *BoardState) CheckOtherPieceChecks(
	wKing, bKing *Position,
	check *CheckState,
) (*CheckState, error) {
	var err error
	for dir := Direction(DownRight); dir < Knight1; dir += 1 {
		check, err = board.otherPieceChecksImpl(wKing, check, White, dir)
		if err != nil {
			return nil, err
		}

		check, err = board.otherPieceChecksImpl(bKing, check, Black, dir)
		if err != nil {
			return nil, err
		}
	}

	return check, nil
}

func (board *BoardState) UpdateCheckState(findErr bool) error {
	wKing, bKing := board.GetKingPositions()

	check, err := board.CheckKnightChecks(wKing, bKing)
	if err != nil {
		return err
	}
	fmt.Printf("%s", check.String())

	check, err = board.CheckOtherPieceChecks(wKing, bKing, check)
	if err != nil {
		return err
	}

	board.Check = *check

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

// fen stuff

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

func (piece Piece) FenByte() byte {
	index := Piece(0)
	index += (NotClear & piece) * (1 + piece.PieceType() + Piece(6*piece.Colour()))
	return pieceToFenArr[index]
}
func (piece Piece) FenString() string {
	return string(piece.FenByte())
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
			ret += piece.FenString()
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

func getPiece(char rune) (Piece, error) {
	switch char {
	case 'k':
		return WKing, nil
	case 'q':
		return WQueen, nil
	case 'b':
		return WBishop, nil
	case 'n':
		return WKnight, nil
	case 'p':
		return WPawn, nil
	case 'r':
		return WRook, nil
	case 'K':
		return BKing, nil
	case 'Q':
		return BQueen, nil
	case 'B':
		return BBishop, nil
	case 'N':
		return BKnight, nil
	case 'P':
		return BPawn, nil
	case 'R':
		return BRook, nil
	case '/':
		return Clear, nil
	default:
		return 0, errors.New("the charcter is an invalid")
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

		piece, err := getPiece(char)

		if err != nil {
			return nil, fmt.Errorf("unexpected character found: %s ", string(char))
		}

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
		} else {
			state[stateIndex] = piece
		}

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
	if color == Black {
		moveCounter += 1
	}

	return &BoardState{State: state, Check: CheckState{}, MoveCounter: uint16(moveCounter)}, nil
}
