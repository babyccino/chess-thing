package board

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
)

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
	CaptureMoveCounter uint16
	MoveHistory        []Move
	MoveCounter        uint16
	LegalMoves         []Move
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
		CaptureMoveCounter: 0,
		MoveHistory:        make([]Move, 0),
		MoveCounter:        0,
		LegalMoves:         nil,
	}
}

// loop
func (board *BoardState) Init() error {
	err := board.UpdateBoardState()
	if err != nil {
		return err
	}

	return board.UpdateLegalMoves()
}

func (board *BoardState) MakeMove(move Move) error {
	if !slices.Contains(board.LegalMoves, move) {
		return errors.New("move is not in legal moves")
	}

	captured, err := board.Move(move.From, move.To)
	if err != nil {
		return err
	}

	err = board.UpdateBoardState()
	if err != nil {
		return err
	}

	board.MoveHistory = append(board.MoveHistory, move)
	board.MoveCounter += 1
	if captured {
		board.CaptureMoveCounter = 0
	} else {
		board.CaptureMoveCounter += 1
	}

	return board.UpdateLegalMoves()
}

type WinState = uint8

const (
	NoWin WinState = iota
	WhiteWin
	BlackWin

	Stalemate
	MoveRuleDraw
)

func (board *BoardState) HasWinner() WinState {
	if board.CaptureMoveCounter == 50 {
		return MoveRuleDraw
	}

	if len(board.LegalMoves) == 0 {
		whoseMove := board.WhoseMove()
		if board.Check.InCheck(whoseMove) {
			if whoseMove == Black {
				return WhiteWin
			} else {
				return BlackWin
			}
		} else {
			return Stalemate
		}
	}

	return NoWin
}

// utility
func (board *BoardState) WhoseMove() Colour {
	if board.MoveCounter%2 == 0 {
		return White
	}
	return Black
}

func (board *BoardState) String() string {
	str := " H G F E D C B A  \n\n "
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
	if piece.IsClear() {
		return 0
	}
	return 1 + int(piece.PieceType()) + int(6*(piece.Colour()-1))
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
	if piece.IsClear() {
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
		if piece.IsClear() {
			counter += 1
		} else if counter != 0 {
			ret += string(rowIntToByte(counter))
			counter = 0
		}

		if !piece.IsClear() {
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

		if piece.IsPieceAndColour(Clear) {
			if stateIndex%8 != 0 || rowIndex != 8 {
				errorStr := fmt.Sprintf("/ found in wrong place stateIndex: %d, rowIndex: %d",
					stateIndex, rowIndex)
				return nil, errors.New(errorStr)
			}

			rowIndex = 0
			continue
		} else {
			if piece.IsPieceAndColour(WKing) && wKing {
				return nil, errors.New("multiple white kings")
			}
			wKing = wKing || piece.IsPieceAndColour(WKing)

			if piece.IsPieceAndColour(BKing) && bKing {
				return nil, errors.New("multiple black kings")
			}
			bKing = bKing || piece.IsPieceAndColour(BKing)

			if piece.IsPieceAndColour(WPawn) {
				if !(stateIndex == 3 ||
					stateIndex == 4 ||
					stateIndex == 11 ||
					stateIndex == 18 ||
					stateIndex == 24 ||
					stateIndex == 25 ||
					stateIndex == 32) {
					piece |= MovedMask
				}
			} else if piece.IsPieceAndColour(BPawn) {
				if !(stateIndex == 31 ||
					stateIndex == 38 ||
					stateIndex == 39 ||
					stateIndex == 45 ||
					stateIndex == 52 ||
					stateIndex == 59 ||
					stateIndex == 60) {
					piece |= MovedMask
				}
			}
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

	var colour Colour
	if fen[boardStrLen] == 'w' {
		colour = White
	} else if fen[boardStrLen] == 'b' {
		colour = Black
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
	if colour == Black {
		moveCounter += 1
	}

	return &BoardState{State: state, Check: CheckState{}, MoveCounter: uint16(moveCounter)}, nil
}

// check stuff

func (board *BoardState) GetKingPositions() (wKing *Position, bKing *Position, err error) {
	for i, piece := range board.State {
		if piece.IsPieceAndColour(WKing) {
			newKing := IndexToPosition(i)
			wKing = &newKing
		}
		if piece.IsPieceAndColour(BKing) {
			newKing := IndexToPosition(i)
			bKing = &newKing
		}
	}

	if wKing == nil && bKing == nil {
		return nil, nil, errors.New("neither king was found")
	} else if wKing == nil {
		return nil, nil, errors.New("no white king was found")
	} else if bKing == nil {
		return nil, nil, errors.New("no black king was found")
	}

	return wKing, bKing, nil
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
		if inBounds && board.GetSquare(pos).IsPieceAndColour(BKnight) {
			if check != NoCheck {
				err := fmt.Errorf("weird board state reached, check: %s\n\n%s",
					CheckToString(check), board.String())
				return nil, err
			}

			check = WhiteCheck
			from = pos
		}

		pos, inBounds = bKing.AddInBounds(vec)
		if inBounds && board.GetSquare(pos).IsPieceAndColour(WKnight) {
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
	if piece.IsPieceAndColour(BPawn) {
		return colour == White && (diff == UpVec || diff == LeftVec)
	} else if piece.IsPieceAndColour(WPawn) {
		return colour == Black && (diff == DownVec || diff == RightVec)
	} else if diagonal {
		return piece.IsDiagonalAttacker()
	} else {
		return piece.IsStraightLongAttacker()
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

func (board *BoardState) UpdateCheckState() error {
	wKing, bKing, err := board.GetKingPositions()
	if err != nil {
		return err
	}

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

func (board *BoardState) attackSquare(start, vec Position) {
	moved, bounds := start.AddInBounds(vec)
	if !bounds {
		return
	}
	piece := board.GetSquare(moved)
	board.SetSquare(moved, piece.Attacked())
}

func (board *BoardState) attackDirection(otherKing Piece, start, vec Position) {
	for {
		var inBounds bool
		start, inBounds = start.AddInBounds(vec)
		if !inBounds {
			return
		}

		piece := board.GetSquare(start)
		board.SetSquare(start, piece.Attacked())

		if piece.IsClear() || piece.IsPieceAndColour(otherKing) {
			continue
		} else {
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
	whoseMove := board.WhoseMove()

	var king Piece
	if whoseMove == White {
		king = WKing
	} else {
		king = BKing
	}

	for index, piece := range board.State {
		pos := IndexToPosition(index)
		if piece.IsClear() || piece.Colour() == whoseMove {
			continue
		}

		if piece.Is(Knight) {
			for _, move := range knightDirectionArray {
				board.attackSquare(pos, move)
			}
			continue
		}

		if piece.IsPieceAndColour(WPawn) {
			board.attackSquare(pos, DownVec)
			board.attackSquare(pos, RightVec)
			continue
		}
		if piece.IsPieceAndColour(BPawn) {
			board.attackSquare(pos, UpVec)
			board.attackSquare(pos, LeftVec)
			continue
		}

		if piece.Is(King) {
			for _, move := range diagonalDirectionArray {
				board.attackSquare(pos, move)
			}
			for _, move := range straightDirectionArray {
				board.attackSquare(pos, move)
			}
			continue
		}

		if piece.IsDiagonalAttacker() {
			for _, move := range diagonalDirectionArray {
				board.attackDirection(king, pos, move)
			}
		}

		if piece.IsStraightLongAttacker() {
			for _, move := range straightDirectionArray {
				board.attackDirection(king, pos, move)
			}
		}
	}
}

func (board *BoardState) UpdateBoardState() error {
	board.ResetPieceStates()

	err := board.UpdateCheckState()
	if err != nil {
		return err
	}
	board.UpdateAttackedSquares()
	return nil
}

// moves

func (board *BoardState) Move(start, end Position) (bool, error) {
	if start == end {
		return false, errors.New("positions are same")
	}
	if start.X >= 8 || end.Y >= 8 {
		return false, errors.New("move out of bounds")
	}

	endPiece := board.GetSquare(end)
	startPiece := board.GetSquare(start)

	board.SetSquare(end, startPiece.Moved())
	board.SetSquare(start, Clear)

	captured := !endPiece.IsClear() && (startPiece.Colour() != endPiece.Colour())
	return captured, nil
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
	_, err = board.Move(startPos, endPos)
	return err
}

func (board *BoardState) UpdateLegalMoves() error {
	board.LegalMoves = nil
	moveMaker := newLegalMoveCreator(board)
	legalMoves, err := moveMaker.getLegalMoves()
	if err != nil {
		return err
	}
	board.LegalMoves = legalMoves
	return nil
}

type ColourLessCheck = uint8

const (
	colourLessNoCheck ColourLessCheck = iota
	colourLessCheck
	colourLessDoubleCheck
)

func checkToColourlessCheck(check Check) ColourLessCheck {
	if check >= BlackCheck {
		return check - 2
	}
	return check
}

func ableToMoveDirection(pin PinDirection, dir Direction) bool {
	switch pin {
	case NoPin:
		return true
	case DownRightPin:
		return dir == DownRight || dir == UpLeft
	case DownLeftPin:
		return dir == DownLeft || dir == UpRight
	case DownPin:
		return dir == Up || dir == Down
	case RightPin:
		return dir == Left || dir == Right
	}
	return true
}

func pieceAbleToMoveDirection(piece Piece, dir Direction) bool {
	pin := piece.GetPin()
	if pin == NoPin {
		return true
	}
	if piece.Is(Knight) {
		return false
	}
	return ableToMoveDirection(pin, dir)
}
