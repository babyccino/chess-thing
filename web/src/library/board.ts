export type Piece = number

export const Clear = 0
export const WKing = 1
export const WQueen = 2
export const WBishop = 3
export const WKnight = 4
export const WPawn = 5
export const WRook = 6
export const BKing = 7
export const BQueen = 8
export const BBishop = 9
export const BKnight = 10
export const BPawn = 11
export const BRook = 12

export const piecesToIcon = [
  "",
  "fa6-solid:chess-king",
  "fa6-solid:chess-queen",
  "fa6-solid:chess-bishop",
  "fa6-solid:chess-knight",
  "fa6-solid:chess-pawn",
  "fa6-solid:chess-rook",
  "fa6-regular:chess-king",
  "fa6-regular:chess-queen",
  "fa6-regular:chess-bishop",
  "fa6-regular:chess-knight",
  "fa6-regular:chess-pawn",
  "fa6-regular:chess-rook",
] as const

export function pieceToIcon(piece: Piece) {
  return piecesToIcon[piece]
}

export const initialBoardState = [
  WKing,
  WRook,
  WBishop,
  WPawn,
  WPawn,
  Clear,
  Clear,
  Clear,
  WRook,
  WQueen,
  WKnight,
  WPawn,
  Clear,
  Clear,
  Clear,
  Clear,
  WKnight,
  WBishop,
  WPawn,
  Clear,
  Clear,
  Clear,
  Clear,
  Clear,
  WPawn,
  WPawn,
  Clear,
  Clear,
  Clear,
  Clear,
  Clear,
  BPawn,
  WPawn,
  Clear,
  Clear,
  Clear,
  Clear,
  Clear,
  BPawn,
  BPawn,
  Clear,
  Clear,
  Clear,
  Clear,
  Clear,
  BPawn,
  BBishop,
  BKnight,
  Clear,
  Clear,
  Clear,
  Clear,
  BPawn,
  BKnight,
  BQueen,
  BRook,
  Clear,
  Clear,
  Clear,
  BPawn,
  BPawn,
  BBishop,
  BRook,
  BKing,
]

export function padNumber(num: number): string {
  if (num < 10) return `0${num}`
  return num.toString()
}

export function serialiseMove(from: number, to: number): string {
  return padNumber(from) + padNumber(to)
}

const aChar = "A".charCodeAt(0)
const zeroChar = "1".charCodeAt(0)
export function indexToFile(index: number): string {
  const x = index % 8
  return String.fromCharCode(aChar + 7 - x)
}
export function indexToRow(index: number): string {
  const y = index / 8
  return String.fromCharCode(zeroChar + y)
}
export function indexToPosition(index: number): string {
  const y = index / 8
  const x = index % 8
  return indexToFile(index) + indexToRow(index)
}
