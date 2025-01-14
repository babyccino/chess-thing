package board

import (
	"errors"
	"fmt"
	"strings"
)

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

func (move *Move) Serialise() string {
	return fmt.Sprintf("%s:%s", move.From.CoordsString(), move.To.CoordsString())
}
func DeserialiseMove(str string) (Move, error) {
	// TODO don't do this like a js andy
	parts := strings.Split(str, ":")
	if len(parts) != 2 {
		return Move{}, errors.New("failed deserialising moves")
	}

	from, err := StringToPosition(parts[0])
	if err != nil {
		return Move{}, err
	}
	to, err := StringToPosition(parts[0])
	if err != nil {
		return Move{}, err
	}

	return Move{From: from, To: to}, nil
}

// todo don't use json arrays
// just do serialisation better in general
func SerialiseMoveList(moveList []Move) []string {
	ret := make([]string, len(moveList))
	for i, move := range moveList {
		ret[i] = move.Serialise()
	}
	return ret
}

type LegalMoveCreator struct {
	moves        []Move
	colour       Colour
	check        ColourLessCheck
	state        *BoardState
	checkSquares []Position
}

func newLegalMoveCreator(board *BoardState) *LegalMoveCreator {
	colour := board.WhoseMove()
	colourLessCheck := checkToColourlessCheck(board.Check.Check)
	moves := make([]Move, 0)

	return &LegalMoveCreator{
		moves,
		colour,
		colourLessCheck,
		board,
		nil,
	}
}

func (moveMaker *LegalMoveCreator) addMove(from, to Position) {
	moveMaker.moves = append(moveMaker.moves, Move{from, to})
}
func (moveMaker *LegalMoveCreator) addKnightMoves(from Position, pin PinDirection) {
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
func (moveMaker *LegalMoveCreator) addPawnMove(from Position, dir Direction, pin PinDirection) {
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
func (moveMaker *LegalMoveCreator) addMoveKing(from Position, dir Direction) {
	to, inBounds := from.AddInBounds(directionToVec(dir))
	if !inBounds {
		return
	}
	toPiece := moveMaker.state.GetSquare(to)
	if (toPiece.IsClear() || toPiece.Colour() != moveMaker.colour) && !toPiece.IsAttacked(moveMaker.colour) {
		moveMaker.addMove(from, to)
	}
}
func (moveMaker *LegalMoveCreator) addKingMoves(from Position) {
	for dir := range Knight1 {
		moveMaker.addMoveKing(from, dir)
	}
}
func (moveMaker *LegalMoveCreator) addMovesInDirection(from Position, dir Direction, pin PinDirection) {
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

func (moveMaker *LegalMoveCreator) getLegalMovesNoCheck() {
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

func (moveMaker *LegalMoveCreator) getLegalMovesCheckImpl(to Position, toPiece Piece, dir Direction) {
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

func (moveMaker *LegalMoveCreator) getLegalMovesCheck() {
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

func (moveMaker *LegalMoveCreator) getLegalKingMoves() {
	for index, piece := range moveMaker.state.State {
		if piece.Colour() == moveMaker.colour && piece.Is(King) {
			moveMaker.addKingMoves(IndexToPosition(index))
			return
		}
	}
}

func (moveMaker *LegalMoveCreator) getLegalMoves() []Move {
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
