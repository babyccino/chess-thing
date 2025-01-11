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
so the check is resolved by the piece being captured
or moving to a blocking square. This also allows the king to capture
the piece as that square is not necessarily also _attacked_
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
	AttackedShift       = 8
	CheckShift          = 10
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

type PieceType uint16

const (
	King PieceType = iota
	Queen
	Bishop
	Knight
	Pawn
	Rook
)

const (
	shiftedWhite = Piece(White) << ColourShift
	shiftedBlack = Piece(Black) << ColourShift

	shiftedKing   = Piece(King) << PieceShift
	shiftedQueen  = Piece(Queen) << PieceShift
	shiftedBishop = Piece(Bishop) << PieceShift
	shiftedKnight = Piece(Knight) << PieceShift
	shiftedPawn   = Piece(Pawn) << PieceShift
	shiftedRook   = Piece(Rook) << PieceShift
)

const (
	WKing   Piece = NotClear | shiftedKing | shiftedWhite
	WQueen        = NotClear | shiftedQueen | shiftedWhite
	WBishop       = NotClear | shiftedBishop | shiftedWhite
	WKnight       = NotClear | shiftedKnight | shiftedWhite
	WPawn         = NotClear | shiftedPawn | shiftedWhite
	WRook         = NotClear | shiftedRook | shiftedWhite
	BKing         = NotClear | shiftedKing | shiftedBlack
	BQueen        = NotClear | shiftedQueen | shiftedBlack
	BBishop       = NotClear | shiftedBishop | shiftedBlack
	BKnight       = NotClear | shiftedKnight | shiftedBlack
	BPawn         = NotClear | shiftedPawn | shiftedBlack
	BRook         = NotClear | shiftedRook | shiftedBlack
)

func (piece Piece) Colour() Colour {
	return Colour((piece & ColourMask) >> ColourShift)
}
func (piece Piece) Is(other PieceType) bool {
	return piece.PieceType() == other
}

const pieceAndColourMask = ClearMask | ColourMask | PieceMask

func (piece Piece) IsPieceAndColour(other Piece) bool {
	return piece&pieceAndColourMask == other&pieceAndColourMask
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
func (piece Piece) PieceType() PieceType {
	return PieceType((piece & PieceMask) >> PieceShift)
}
func (piece Piece) IsDiagonalAttacker() bool {
	return piece.Is(Queen) || piece.Is(Bishop)
}
func (piece Piece) IsStraightLongAttacker() bool {
	return piece.Is(Queen) || piece.Is(Rook)
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
	return Colour(piece & AttackedMask >> AttackedShift)
}
func (piece Piece) IsAttacked(colour Colour) bool {
	return piece.GetAttacked() != colour
}
func (piece Piece) Attacked(colour Colour) Piece {
	return piece | (Piece(colour) << AttackedShift)
}

func (piece Piece) IsCheckSquare() bool {
	return piece&CheckMask == CheckMask
}
func (piece Piece) CheckSquare() Piece {
	return piece | CheckMask
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

func (board *BoardState) WhoseMove() Colour {
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

func (piece Piece) arrIndex() int {
	return int(NotClear&piece) * (1 + int(piece.PieceType()) + int(6*piece.Colour()))
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
	return pieceToStrArr[piece.arrIndex()]
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
	return pieceToFenArr[piece.arrIndex()]
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

	if board.WhoseMove() == White {
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

	wKing := false
	bKing := false
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

			rowIndex = 0
			continue
		} else {
			if piece == WKing && wKing {
				return nil, errors.New("multiple white kings")
			}
			wKing = wKing || piece == WKing

			if piece == BKing && bKing {
				return nil, errors.New("multiple black kings")
			}
			bKing = bKing || piece == BKing

			state[stateIndex] = piece
		}

		stateIndex += 1
		rowIndex += 1

		if rowIndex > 8 {
			errorStr := fmt.Sprintf("row index too large: %d", rowIndex)
			return nil, errors.New(errorStr)
		}
	}

	if !wKing || !bKing {
		return nil, errors.New("need both black and white king on the board")
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
	wKing, bKing *Position,
) (*CheckState, error) {
	// check the knight checks first because a double knight check is not possible
	check := NoCheck
	from := Position{}

	for _, vec := range knightDirectionArray {
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
	if piece.IsClear() {
		return false
	}
	if (colour == White) == piece.IsWhite() {
		return false
	}

	diff := king.Diff(piecePosition)
	if piece == BPawn {
		return colour == White && (diff == UpVec || diff == LeftVec)
	} else if piece == WPawn {
		return colour == Black && (diff == DownVec || diff == RightVec)
	} else if diagonal {
		return piece.IsDiagonalAttacker()
	} else {
		return piece.IsStraightLongAttacker()
	}
}

func CanPieceDoMove(
	from, to Position,
	fromPiece, toPiece Piece,
	diagonal bool,
) bool {
	if fromPiece == Clear {
		return false
	}

	diff := to.Diff(from)
	toPieceColour := toPiece.Colour()

	if fromPiece.IsPieceAndColour(BPawn) {
		return toPieceColour == White && (diff == UpVec || diff == LeftVec)
	} else if fromPiece.IsPieceAndColour(WPawn) {
		return toPieceColour == Black && (diff == DownVec || diff == RightVec)
	} else if diagonal {
		return fromPiece.IsDiagonalAttacker()
	} else {
		return fromPiece.IsStraightLongAttacker()
	}
}
func (board *BoardState) addCheckSquares(
	from,
	to *Position,
	dir Direction,
) {
	piece := board.GetSquare(*to)
	board.SetSquare(*to, piece.CheckSquare())

	vec := directionArray[dir]
	added, inBounds := from.AddInBounds(vec)
	if !inBounds {
		panic("reached out of bounds while adding check squares")
	}
	if dir >= Knight1 {
		return
	}

	for added != *to {
		piece := board.GetSquare(added)
		board.SetSquare(added, piece.CheckSquare())

		var inBounds bool
		added, inBounds = added.AddInBounds(vec)
		if !inBounds {
			panic("reached out of bounds while adding check squares")
		}
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

	if piece.IsClear() {
		return check, nil
	}

	if AmBeingAttacked(king, piece, colour, piecePosition, diagonal) {
		if colour == White && checkIsBlack(check.Check) ||
			colour == Black && checkIsWhite(check.Check) {
			return nil,
				errors.New("both white and black kings are being attacked simultaneously")
		}

		err := check.Promote(colour)
		if err != nil {
			return nil, err
		}
		check.From = piecePosition

		if check.Check == WhiteDoubleCheck {
			return check, nil
		}

		board.addCheckSquares(king, &piecePosition, dir)
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
	for dir := DownRight; dir < Knight1; dir += 1 {
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

	check, err = board.CheckOtherPieceChecks(wKing, bKing, check)
	if err != nil {
		return err
	}

	board.Check = *check

	return nil
}

// moves

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

func (move *Move) String() string {
	return fmt.Sprintf("(%s -> %s)", move.From.CoordsString(), move.To.CoordsString())
}

func MoveListToString(moveList []Move) string {
	ret := "["
	for _, move := range moveList {
		ret += move.String() + ", "
	}
	return ret + "]"
}

type ColourLessCheck = uint8

const (
	colourLessNoCheck ColourLessCheck = iota
	colourLessCheck
	colourLessDoubleCheck
)

func (board *BoardState) attackSquare(colour Colour, start, vec Position) {
	moved, bounds := start.AddInBounds(vec)
	if !bounds {
		return
	}
	piece := board.GetSquare(moved)
	board.SetSquare(moved, piece.Attacked(colour))
}

func (board *BoardState) attackDirection(colour Colour, start, vec Position) {
	for {
		var inBounds bool
		start, inBounds = start.AddInBounds(vec)
		if !inBounds {
			return
		}

		piece := board.GetSquare(start)
		board.SetSquare(start, piece.Attacked(colour))
		if !piece.IsClear() {
			return
		}
	}
}

func (board *BoardState) ResetPieceStates() {
	for index, piece := range board.State {
		board.State[index] = piece.Reset()
	}
}
func (board *BoardState) UpdateAttackedSquares() {
	for index, piece := range board.State {
		pos := IndexToPosition(index)
		colour := piece.Colour()

		if piece.Is(Knight) {
			for _, move := range knightDirectionArray {
				board.attackSquare(colour, pos, move)
			}
			continue
		}

		if piece.IsPieceAndColour(WPawn) {
			board.attackSquare(colour, pos, DownVec)
			board.attackSquare(colour, pos, RightVec)
			continue
		}
		if piece.IsPieceAndColour(BPawn) {
			board.attackSquare(colour, pos, UpVec)
			board.attackSquare(colour, pos, LeftVec)
			continue
		}

		if piece.Is(King) {
			for _, move := range diagonalDirectionArray {
				board.attackSquare(colour, pos, move)
			}
			for _, move := range straightDirectionArray {
				board.attackSquare(colour, pos, move)
			}
			continue
		}

		if piece.IsDiagonalAttacker() {
			for _, move := range diagonalDirectionArray {
				board.attackDirection(colour, pos, move)
			}
		}

		if piece.IsStraightLongAttacker() {
			for _, move := range straightDirectionArray {
				board.attackDirection(colour, pos, move)
			}
		}
	}
}

func checkToColourlessCheck(check Check) ColourLessCheck {
	if check >= BlackCheck {
		return check - 2
	}
	return check
}

func isPinnedInDirection(pin PinDirection, dir Direction) bool {
	switch pin {
	case DownRightPin:
		return dir == DownLeft || dir == UpRight
	case DownLeftPin:
		return dir == DownRight || dir == UpLeft
	case DownPin:
		return dir == Left || dir == Right
	case RightPin:
		return dir == Up || dir == Down
	}
	return false
}
func isPiecePinnedInDirection(piece Piece, dir Direction) bool {
	pin := piece.GetPin()
	if pin == NoPin {
		return false
	}
	if piece.Is(Knight) {
		return true
	}
	return isPinnedInDirection(pin, dir)
}

type MoveMaker struct {
	moves        []Move
	colour       Colour
	check        ColourLessCheck
	state        *BoardState
	checkSquares []Position
}

func newMoveMaker(board *BoardState) *MoveMaker {
	colour := board.WhoseMove()
	colourLessCheck := checkToColourlessCheck(board.Check.Check)
	moves := make([]Move, 0)

	return &MoveMaker{
		moves,
		colour,
		colourLessCheck,
		board,
		nil,
	}
}

func (moveMaker *MoveMaker) addMove(from, to Position) {
	moveMaker.moves = append(moveMaker.moves, Move{from, to})
}
func (moveMaker *MoveMaker) addKnightMoves(from Position, pin PinDirection) {
	if pin != NoPin {
		return
	}
	for dir := Knight1; dir <= Knight8; dir += 1 {
		to, inBounds := from.AddInBounds(directionToVec(dir))
		if !inBounds {
			return
		}
		toPiece := moveMaker.state.GetSquare(to)
		if toPiece.Colour() == moveMaker.colour {
			return
		}
		moveMaker.addMove(from, to)
	}
}
func (moveMaker *MoveMaker) addPawnMove(from Position, dir Direction, pin PinDirection) {
	if isPinnedInDirection(pin, dir) {
		return
	}
	to, inBounds := from.AddInBounds(directionToVec(dir))
	if !inBounds {
		return
	}
	toPiece := moveMaker.state.GetSquare(to)
	if isStraight(dir) {
		if toPiece.Colour() != moveMaker.colour {
			moveMaker.addMove(from, to)
		}
	} else {
		if toPiece.IsClear() {
			moveMaker.addMove(from, to)
		}
	}
}
func (moveMaker *MoveMaker) addMoveKing(from Position, dir Direction) {
	to, inBounds := from.AddInBounds(directionToVec(dir))
	if !inBounds {
		return
	}
	toPiece := moveMaker.state.GetSquare(to)
	if (toPiece.IsClear() || toPiece.Colour() != moveMaker.colour) && !toPiece.IsAttacked(moveMaker.colour) {
		moveMaker.addMove(from, to)
	}
}
func (moveMaker *MoveMaker) addKingMoves(from Position) {
	for dir := range Knight1 {
		moveMaker.addMoveKing(from, dir)
	}
}
func (moveMaker *MoveMaker) addMovesInDirection(from Position, dir Direction, pin PinDirection) {
	if isPinnedInDirection(pin, dir) {
		return
	}
	to := from
	for {
		var inBounds bool
		to, inBounds = to.AddInBounds(to)
		if !inBounds {
			return
		}

		piece := moveMaker.state.GetSquare(to)
		if piece.IsClear() {
			moveMaker.addMove(from, to)
			continue
		}

		toPiece := moveMaker.state.GetSquare(to)
		if toPiece.Colour() != moveMaker.colour {
			moveMaker.addMove(from, to)
		}
	}
}

func (moveMaker *MoveMaker) getLegalMovesNoCheck() {
	for index, piece := range moveMaker.state.State {
		if piece.IsClear() || piece.Colour() != moveMaker.colour {
			continue
		}

		from := IndexToPosition(index)

		if piece.Is(King) {
			moveMaker.addKingMoves(from)
			continue
		}

		pin := piece.GetPin()
		if piece.Is(Knight) {
			moveMaker.addKnightMoves(from, pin)
			continue
		}

		if piece.IsPieceAndColour(WPawn) {
			moveMaker.addPawnMove(from, Down, pin)
			moveMaker.addPawnMove(from, DownRight, pin)
			moveMaker.addPawnMove(from, Right, pin)
			continue
		}
		if piece.IsPieceAndColour(BPawn) {
			moveMaker.addPawnMove(from, Up, pin)
			moveMaker.addPawnMove(from, UpLeft, pin)
			moveMaker.addPawnMove(from, Left, pin)
			continue
		}

		if piece.IsDiagonalAttacker() {
			for dir := Direction(0); dir <= UpRight; dir += 1 {
				moveMaker.addMovesInDirection(from, dir, pin)
			}
		}

		if piece.IsStraightLongAttacker() {
			for dir := Up; dir <= Right; dir += 1 {
				moveMaker.addMovesInDirection(from, dir, pin)
			}
		}
	}
}

func (moveMaker *MoveMaker) getLegalMovesCheckImpl(to Position, toPiece Piece, dir Direction) {
	diagonal := dir <= UpRight
	vec := directionArray[dir]

	fromPiece, from := moveMaker.state.FindInDirection(vec, &to)

	if fromPiece == Clear ||
		fromPiece.Is(King) ||
		fromPiece.Colour() != moveMaker.colour ||
		toPiece.Colour() == moveMaker.colour {
		return
	}

	if !isPiecePinnedInDirection(fromPiece, dir) && CanPieceDoMove(
		from,
		to,
		fromPiece,
		toPiece,
		diagonal,
	) {
		moveMaker.moves = append(moveMaker.moves, Move{from, to})
	}
}

func (moveMaker *MoveMaker) getLegalMovesCheck() {
	moveMaker.getLegalKingMoves()

	for index, piece := range moveMaker.state.State {
		if !piece.IsCheckSquare() {
			continue
		}

		to := IndexToPosition(index)

		for dir := DownRight; dir < Knight1; dir += 1 {
			moveMaker.getLegalMovesCheckImpl(to, piece, dir)
		}

		for _, move := range knightDirectionArray {
			otherSquare, bounds := to.AddInBounds(move)
			if !bounds {
				continue
			}

			otherPiece := moveMaker.state.GetSquare(otherSquare)
			if otherPiece.Colour() != moveMaker.colour ||
				!piece.Is(Knight) ||
				piece.IsPinned() {
				continue
			}

			moveMaker.moves = append(moveMaker.moves, Move{otherSquare, to})
			continue
		}
	}
}

func (moveMaker *MoveMaker) getLegalKingMoves() {
	for index, piece := range moveMaker.state.State {
		if piece.Colour() == moveMaker.colour && piece.Is(King) {
			moveMaker.addKingMoves(IndexToPosition(index))
			return
		}
	}
}

func (moveMaker *MoveMaker) getLegalMoves() []Move {
	switch moveMaker.check {
	case colourLessNoCheck:
		moveMaker.getLegalMovesNoCheck()
	case colourLessCheck:
		moveMaker.getLegalMovesCheck()
	case colourLessDoubleCheck:
		moveMaker.getLegalKingMoves()
	}
	return moveMaker.moves
}

func (board *BoardState) GetLegalMoves() []Move {
	moveMaker := newMoveMaker(board)
	return moveMaker.getLegalMoves()
}
