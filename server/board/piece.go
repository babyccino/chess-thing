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
