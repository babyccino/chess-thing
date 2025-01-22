package board

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
moved check_square attacked pinned piece colour
1     0            01       100    011   01

if there is a white piece on a square and it white attacked then that piece is defended
i.e. cannot be taken by the black king

check_square is a square a piece needs to move to _resolve_ a check
this includes the square the checking piece is on,
so the check is resolved by the piece being captured
or moving to a blocking square. This also allows the king to capture
the piece as that square is not necessarily also _attacked_

moved is to check whether pawns are on their initial squares
this is needed because the rear flank pawns can end up on the starting squares
of the two pawns in front of them
*/

const (
	ColourMask   Piece = 0b00000000011
	PieceMask          = 0b00000011100
	PinMask            = 0b00011100000
	AttackedMask       = 0b00100000000
	CheckMask          = 0b01000000000
	MovedMask          = 0b10000000000
)

const (
	PieceShift    uint8 = 2
	PinShift            = 5
	AttackedShift       = 8
	CheckShift          = 9
	MovedShift          = 10
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
	None Colour = iota
	White
	Black
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
	shiftedKing   = Piece(King) << PieceShift
	shiftedQueen  = Piece(Queen) << PieceShift
	shiftedBishop = Piece(Bishop) << PieceShift
	shiftedKnight = Piece(Knight) << PieceShift
	shiftedPawn   = Piece(Pawn) << PieceShift
	shiftedRook   = Piece(Rook) << PieceShift
)

const (
	Clear   Piece = 0
	WKing         = shiftedKing | Piece(White)
	WQueen        = shiftedQueen | Piece(White)
	WBishop       = shiftedBishop | Piece(White)
	WKnight       = shiftedKnight | Piece(White)
	WPawn         = shiftedPawn | Piece(White)
	WRook         = shiftedRook | Piece(White)
	BKing         = shiftedKing | Piece(Black)
	BQueen        = shiftedQueen | Piece(Black)
	BBishop       = shiftedBishop | Piece(Black)
	BKnight       = shiftedKnight | Piece(Black)
	BPawn         = shiftedPawn | Piece(Black)
	BRook         = shiftedRook | Piece(Black)
)

func (piece Piece) Colour() Colour {
	return Colour(piece & ColourMask)
}
func (piece Piece) Is(other PieceType) bool {
	return !piece.IsClear() && piece.PieceType() == other
}

const pieceAndColourMask = ColourMask | PieceMask

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
	return piece&ColourMask == Clear
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

func (piece Piece) IsAttacked() bool {
	return piece&AttackedMask == AttackedMask
}
func (piece Piece) Attacked() Piece {
	return piece | AttackedMask
}

func (piece Piece) IsCheckSquare() bool {
	return piece&CheckMask == CheckMask
}
func (piece Piece) CheckSquare() Piece {
	return piece | CheckMask
}

func (piece Piece) Moved() Piece {
	return piece | MovedMask
}
func (piece Piece) IsMoved() bool {
	return piece&MovedMask == MovedMask
}

func (piece Piece) Reset() Piece {
	return piece & (ColourMask | PieceMask | MovedMask)
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
