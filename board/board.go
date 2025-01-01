package board

import (
	"errors"
	"fmt"
	"strconv"
)

type Piece uint16
type Colour uint8

func ColourString(colour Colour) string {
	if colour == White {
		return "white"
	} else {
		return "black"
	}
}
func oppositeColour(colour Colour) Colour {
	if colour == White {
		return Black
	}
	if colour == Black {
		return White
	}
	return None
}

/*
check_square attacked pinned piece colour clear
0            01       100    011   0      1

if there is a white piece on a square and it white attacked then that piece is defended
i.e. cannot be taken by the black king

check_square is a square a piece needs to move to _resolve_ a check
this includes the square the checking piece is on,
so the check is resolved by the piece being captures
or a blocking square
*/

// clear (no piece on that square) = 0b0
const (
	Clear    Piece = 0b0
	NotClear       = 0b1
)

const (
	ClearMask    Piece = 0b00000000001
	ColourMask         = 0b00000000010
	PieceMask          = 0b00000011100
	PinMask            = 0b00011100000
	AttackedMask       = 0b01100000000
	CheckMask          = 0b10000000000
)

const (
	ColourShift   uint8 = 1
	PieceShift          = 2
	PinShift            = 5
	AttackedShift       = 7
	CheckShift          = 8
)

type PinDirection = uint8

const (
	NoPin        PinDirection = 0b000
	DownRightPin              = 0b001
	DownLeftPin               = 0b010
	DownPin                   = 0b011
	RightPin                  = 0b100
)

func PinToString(pin PinDirection) string {
	switch pin {
	case NoPin:
		return "no pin"
	case DownRightPin:
		return "down right/up left pin"
	case DownLeftPin:
		return "down left/up right pin"
	case DownPin:
		return "down/up pin"
	case RightPin:
		return "right/left pin"
	default:
		return ""
	}
}

const (
	White Colour = iota
	Black
	None
	Both
)

const (
	King Piece = iota
	Queen
	Bishop
	Knight
	Pawn
	Rook
)

const shiftedWhite = Piece(White) << ColourShift
const shiftedBlack = Piece(Black) << ColourShift
const (
	WKing   Piece = NotClear | King<<PieceShift | shiftedWhite
	WQueen        = NotClear | Queen<<PieceShift | shiftedWhite
	WBishop       = NotClear | Bishop<<PieceShift | shiftedWhite
	WKnight       = NotClear | Knight<<PieceShift | shiftedWhite
	WPawn         = NotClear | Pawn<<PieceShift | shiftedWhite
	WRook         = NotClear | Rook<<PieceShift | shiftedWhite
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
func (piece Piece) Is(other Piece) bool {
	return piece&PieceMask == other&PieceMask
}
func (piece Piece) IsPieceAndColour(other Piece) bool {
	return piece&(PieceMask|ColourMask) == other&(PieceMask|ColourMask)
}
func (piece Piece) IsWhite() bool {
	return piece.Colour() == White
}
func (piece Piece) IsBlack() bool {
	return piece.Colour() == Black
}
func (piece Piece) IsClear() bool {
	return piece&ClearMask == Clear
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

func (piece Piece) Pin(pin PinDirection) Piece {
	return piece | (Piece(pin) << PinShift)
}
func (piece Piece) IsPinned() bool {
	return piece&PinMask != 0b0
}
func (piece Piece) GetPin() PinDirection {
	return PinDirection(piece&PinMask) >> PinShift
}

func (piece Piece) GetAttacked() Colour {
	return Colour(piece&AttackedMask) >> AttackedShift
}
func (piece Piece) Attacked(colour Colour) Piece {
	return piece | (Piece(colour) << AttackedShift)
}

func (piece Piece) IsCheckSquare() bool {
	return piece&CheckMask == CheckMask
}

func (piece Piece) Reset() Piece {
	return piece & (ColourMask | PieceMask | ClearMask)
}

func directionToPinDirection(dir Direction) PinDirection {
	switch dir {
	case UpLeft:
		fallthrough
	case DownRight:
		return DownRightPin

	case UpRight:
		fallthrough
	case DownLeft:
		return DownLeftPin

	case Up:
		fallthrough
	case Down:
		return DownPin

	case Left:
		fallthrough
	case Right:
		return RightPin

	default:
		panic("can not pin with a knight")
	}
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

type Check = uint8

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
		WKing, WRook, WBishop, WPawn, WPawn, Clear, Clear, Clear,
		WRook, WQueen, WKnight, WPawn, Clear, Clear, Clear, Clear,
		WKnight, WBishop, WPawn, Clear, Clear, Clear, Clear, Clear,
		WPawn, WPawn, Clear, Clear, Clear, Clear, Clear, BPawn,
		WPawn, Clear, Clear, Clear, Clear, Clear, BPawn, BPawn,
		Clear, Clear, Clear, Clear, Clear, BPawn, BBishop, BKnight,
		Clear, Clear, Clear, Clear, BPawn, BKnight, BQueen, BRook,
		Clear, Clear, Clear, BPawn, BPawn, BBishop, BRook, BKing,
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

func (board *BoardState) Move(start, end Position) error {
	if start == end {
		return errors.New("positions are same")
	}
	if start.X >= 8 || end.Y >= 8 {
		return errors.New("move out of bounds")
	}

	board.SetSquare(end, board.GetSquare(start))
	board.SetSquare(start, Clear)
	board.ResetPieceStates()

	return nil
}

func (board *BoardState) MoveStr(start, end string) error {
	startPos, err := StringToPosition(start)
	if err != nil {
		return err
	}
	endPos, err := StringToPosition(end)
	if err != nil {
		return err
	}
	return board.Move(startPos, endPos)
}

type Move struct {
	From Position
	To   Position
}

// check stuff

func (board *BoardState) GetKingPositions() (wKing *Position, bKing *Position) {
	for i, piece := range board.State {
		if piece == WKing {
			newKing := IndexToPosition(i)
			wKing = &newKing
		}
		if piece == BKing {
			newKing := IndexToPosition(i)
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

	if piece == Clear {
		return check, nil
	}

	if AmBeingAttacked(king, piece, colour, piecePosition, diagonal) {
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
			board.SetSquare(piecePosition,
				piece.Pin(directionToPinDirection(dir)))
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

func (board *BoardState) ResetPieceStates() {
	for index, piece := range board.State {
		board.State[index] = piece.Reset()
	}
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
	index := (NotClear & piece) * (1 + piece.PieceType() + Piece(6*piece.Colour()))
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

func (board *BoardState) GetLegalMoves() []Move {
	return nil
}
