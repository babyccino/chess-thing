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

type BoardState = Piece[]
// prettier-ignore
const _initialBoardState = [
  WKing, WRook, WBishop, WPawn, WPawn, Clear, Clear, Clear,
  WRook, WQueen, WKnight, WPawn, Clear, Clear, Clear, Clear,
  WKnight, WBishop, WPawn, Clear, Clear, Clear, Clear, Clear,
  WPawn, WPawn, Clear, Clear, Clear, Clear, Clear, BPawn,
  WPawn, Clear, Clear, Clear, Clear, Clear, BPawn, BPawn,
  Clear, Clear, Clear, Clear, Clear, BPawn, BBishop, BKnight,
  Clear, Clear, Clear, Clear, BPawn, BKnight, BQueen, BRook,
  Clear, Clear, Clear, BPawn, BPawn, BBishop, BRook, BKing,
]
export const initialBoardState = () => Array.from(_initialBoardState)
export const emptyBoardState = (): BoardState => new Array(64).fill(Clear)

type Check = number
const NoCheck: Check = 0
const WhiteCheck: Check = 1
const BlackCheck: Check = 3

type Board = {
  state: BoardState
  check: Check
  moveCounter: number
}

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

function getPiece(char: string): Piece {
  if (char === "k") return WKing
  if (char === "q") return WQueen
  if (char === "b") return WBishop
  if (char === "n") return WKnight
  if (char === "p") return WPawn
  if (char === "r") return WRook
  if (char === "K") return BKing
  if (char === "Q") return BQueen
  if (char === "B") return BBishop
  if (char === "N") return BKnight
  if (char === "P") return BPawn
  if (char === "R") return BRook
  if (char === "/") return Clear
  throw new Error("invalid character")
}

const White = "w"
const Black = "b"
type Colour = typeof White | typeof Black

function ParseFen(fen: string): Board {
  const state = emptyBoardState()
  let stateIndex = 0
  let rowIndex = 0

  let wKing = false
  let bKing = false
  let boardStrLen = 0

  for (let strIndex = 0; strIndex < fen.length; ++strIndex) {
    const char = fen[strIndex]
    if (stateIndex == 64) {
      if (char != " ") {
        throw new Error("space not found at end of pieces")
      }

      boardStrLen = strIndex + 1
      break
    }

    if (char >= "1" && char <= "8") {
      const delta = char.charCodeAt(0) - zeroChar
      stateIndex += delta
      rowIndex += delta

      continue
    }

    const piece = getPiece(char)

    if (piece == Clear) {
      if (stateIndex % 8 != 0 || rowIndex != 8) {
        throw new Error(`/ found in wrong place stateIndex: ${stateIndex}, rowIndex: ${rowIndex}`)
      }

      rowIndex = 0
      continue
    } else {
      if (piece == WKing && wKing) {
        throw new Error("multiple white kings")
      }
      wKing = wKing || piece == WKing

      if (piece == BKing && bKing) {
        throw new Error("multiple black kings")
      }
      bKing = bKing || piece == BKing

      state[stateIndex] = piece
    }

    stateIndex += 1
    rowIndex += 1

    if (rowIndex > 8) {
      throw new Error("row index too large:" + rowIndex)
    }
  }

  if (!wKing || !bKing) {
    throw new Error("need both black and white king on the board")
  }

  let color: Colour
  if (fen[boardStrLen] == "w") {
    color = White
  } else if (fen[boardStrLen] == "b") {
    color = Black
  } else {
    const errorStr = `unexpected character, should be w or b: ${fen[boardStrLen]}`
    throw new Error(errorStr)
  }

  boardStrLen += 1
  if (fen[boardStrLen] != " ") {
    const errorStr = `unexpected character, should be space: ${fen[boardStrLen]}`
    throw new Error(errorStr)
  }

  boardStrLen += 1
  let moveCounter = parseInt(fen.substring(boardStrLen))

  moveCounter = moveCounter * 2
  if (color == Black) {
    moveCounter += 1
  }

  return { state, check: 0, moveCounter: moveCounter }
}
