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
	for i, move := range moveList {
		if i < len(moveList)-1 {
			ret += move.String() + ", "
		} else {
			ret += move.String()
		}
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
	to, err := StringToPosition(parts[1])
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

func CanPieceDoMove(
	from, to Position,
	fromPiece, toPiece Piece,
	dir Direction,
) bool {
	diagonal := dir <= UpRight
	if fromPiece.IsClear() || !pieceAbleToMoveDirection(fromPiece, dir) {
		return false
	}

	// todo long pawn moves
	toPieceColour := toPiece.Colour()
	if fromPiece.IsPieceAndColour(BPawn) {
		diff := to.Diff(from)
		return toPieceColour == White && (diff == UpVec || diff == LeftVec)
	} else if fromPiece.IsPieceAndColour(WPawn) {
		diff := to.Diff(from)
		return toPieceColour == Black && (diff == DownVec || diff == RightVec)
	} else if diagonal {
		return fromPiece.IsDiagonalAttacker()
	} else {
		return fromPiece.IsStraightLongAttacker()
	}
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

func (moveMaker *LegalMoveCreator) addKnightMoves(piece Piece, from Position) error {
	pin := piece.GetPin()
	if pin != NoPin {
		return nil
	}
	for dir := Knight1; dir <= Knight8; dir += 1 {
		to, inBounds := from.AddInBounds(directionToVec(dir))
		if !inBounds {
			continue
		}

		toPiece := moveMaker.state.GetSquare(to)
		if toPiece.IsClear() {
			moveMaker.addMove(from, to)
			continue
		}
		if toPiece.Colour() != moveMaker.colour {
			if toPiece.Is(King) {
				return errors.New("move to king found")
			}
			moveMaker.addMove(from, to)
		}
	}
	return nil
}

func (moveMaker *LegalMoveCreator) addPawnMoveLong(piece Piece, from Position, dir Direction) {
	pin := piece.GetPin()
	if !ableToMoveDirection(pin, dir) {
		return
	}

	vec := directionToVec(dir)
	to, inBounds := from.AddInBounds(vec)
	if !inBounds {
		return
	}

	toPiece := moveMaker.state.GetSquare(to)
	if toPiece.IsClear() {
		moveMaker.addMove(from, to)
	} else {
		return
	}

	if piece.IsMoved() {
		return
	}
	to, inBounds = from.AddInBoundsMult(vec, 2)
	if !inBounds {
		return
	}

	if toPiece.IsClear() {
		moveMaker.addMove(from, to)
	}
}
func (moveMaker *LegalMoveCreator) addPawnMoveStraight(piece Piece, from Position, dir Direction) error {
	pin := piece.GetPin()
	if !ableToMoveDirection(pin, dir) {
		return nil
	}

	to, inBounds := from.AddInBounds(directionToVec(dir))
	if !inBounds {
		return nil
	}

	toPiece := moveMaker.state.GetSquare(to)
	if !toPiece.IsClear() && toPiece.Colour() != moveMaker.colour {
		if toPiece.Is(King) {
			return errors.New("move to king found")
		}
		moveMaker.addMove(from, to)
	}
	return nil
}
func (moveMaker *LegalMoveCreator) addPawnMoves(piece Piece, from Position) error {
	if piece.Colour() == White {
		err := moveMaker.addPawnMoveStraight(piece, from, Down)
		if err != nil {
			return err
		}
		err = moveMaker.addPawnMoveStraight(piece, from, Right)
		if err != nil {
			return err
		}
		moveMaker.addPawnMoveLong(piece, from, DownRight)
	} else {
		err := moveMaker.addPawnMoveStraight(piece, from, Up)
		if err != nil {
			return err
		}
		err = moveMaker.addPawnMoveStraight(piece, from, Left)
		if err != nil {
			return err
		}
		moveMaker.addPawnMoveLong(piece, from, UpLeft)
	}
	return nil
}

func (moveMaker *LegalMoveCreator) addMoveKing(from Position, dir Direction) error {
	to, inBounds := from.AddInBounds(directionToVec(dir))
	if !inBounds {
		return nil
	}
	toPiece := moveMaker.state.GetSquare(to)
	if toPiece.Is(King) {
		return errors.New("move to king found")
	}
	if (toPiece.IsClear() || toPiece.Colour() != moveMaker.colour) && !toPiece.IsAttacked() {
		moveMaker.addMove(from, to)
	}
	return nil
}
func (moveMaker *LegalMoveCreator) addKingMoves(from Position) error {
	for dir := range Knight1 {
		err := moveMaker.addMoveKing(from, dir)
		if err != nil {
			return err
		}
	}
	return nil
}
func (moveMaker *LegalMoveCreator) addMovesInDirection(piece Piece, from Position, dir Direction) error {
	pin := piece.GetPin()
	if !ableToMoveDirection(pin, dir) {
		return nil
	}

	to := from
	dirVec := directionToVec(dir)
	for {
		var inBounds bool
		to, inBounds = to.AddInBounds(dirVec)
		if !inBounds {
			return nil
		}

		toPiece := moveMaker.state.GetSquare(to)
		if toPiece.IsClear() {
			moveMaker.addMove(from, to)
		} else {
			if toPiece.Colour() != moveMaker.colour {
				if toPiece.Is(King) {
					return errors.New("move to king found")
				}
				moveMaker.addMove(from, to)
			}
			return nil
		}
	}
}

func (moveMaker *LegalMoveCreator) getLegalMovesNoCheck() error {
	for index, piece := range moveMaker.state.State {
		if piece.IsClear() || piece.Colour() != moveMaker.colour {
			continue
		}

		from := IndexToPosition(index)

		if piece.Is(King) {
			err := moveMaker.addKingMoves(from)
			if err != nil {
				return err
			}
			continue
		}

		if piece.Is(Knight) {
			err := moveMaker.addKnightMoves(piece, from)
			if err != nil {
				return err
			}
			continue
		}

		if piece.Is(Pawn) {
			err := moveMaker.addPawnMoves(piece, from)
			if err != nil {
				return err
			}
			continue
		}

		if piece.IsDiagonalAttacker() {
			for dir := Direction(0); dir <= UpRight; dir += 1 {
				err := moveMaker.addMovesInDirection(piece, from, dir)
				if err != nil {
					return err
				}
			}
		}

		if piece.IsStraightLongAttacker() {
			for dir := Up; dir <= Right; dir += 1 {
				err := moveMaker.addMovesInDirection(piece, from, dir)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (moveMaker *LegalMoveCreator) getLegalMovesCheckImpl(to Position, toPiece Piece, dir Direction) {
	vec := directionArray[dir]

	fromPiece, from := moveMaker.state.FindInDirection(vec, &to)

	if fromPiece.IsClear() ||
		fromPiece.Is(King) ||
		fromPiece.Colour() != moveMaker.colour ||
		toPiece.Colour() == moveMaker.colour {
		return
	}

	if CanPieceDoMove(
		from,
		to,
		fromPiece,
		toPiece,
		dir,
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
		if piece.Is(King) && piece.Colour() == moveMaker.colour {
			moveMaker.addKingMoves(IndexToPosition(index))
			return
		}
	}
}

func (moveMaker *LegalMoveCreator) getLegalMoves() ([]Move, error) {
	switch moveMaker.check {
	case colourLessNoCheck:
		err := moveMaker.getLegalMovesNoCheck()
		if err != nil {
			return nil, err
		}
	case colourLessCheck:
		moveMaker.getLegalMovesCheck()
	case colourLessDoubleCheck:
		moveMaker.getLegalKingMoves()
	}
	return moveMaker.moves, nil
}
