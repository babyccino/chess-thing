import type { SendMoveEvent } from "./events"

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

export type Position = number
export type Move = { from: Position; to: Position }

export type Board = {
  state: Piece[]
  colour: Colour
  moveCounter: number
  legalMoves: Move[]
  moveHistory: Move[]
}

export function newBoard(): Board {
  return {
    state: initialBoardState(),
    colour: White,
    moveCounter: 0,
    legalMoves: [],
    moveHistory: [],
  }
}

export function padNumber(num: number): string {
  if (num < 10) return `0${num}`
  return num.toString()
}

export function serialiseMove(from: Position, to: Position): string {
  return indexToPosition(from) + ":" + indexToPosition(to)
}
export function deSerialiseMove(str: string): Move {
  const parts = str.split(":")
  if (parts.length != 2) {
    throw new Error("failed deserialising moves")
  }

  const from = stringToIndex(parts[0])
  const to = stringToIndex(parts[1])

  return { from, to }
}

const aChar = "A".charCodeAt(0)
const hChar = "H".charCodeAt(0)
const zeroChar = "0".charCodeAt(0)
const oneChar = "1".charCodeAt(0)
const eightChar = "8".charCodeAt(0)
export function indexToFile(index: number): string {
  const x = index % 8
  return String.fromCharCode(aChar + 7 - x)
}
export function indexToRow(index: number): string {
  const y = index / 8
  return String.fromCharCode(oneChar + y)
}
export function indexToPosition(index: number): string {
  const y = index / 8
  const x = index % 8
  return indexToFile(index) + indexToRow(index)
}

function stringToIndex(str: string): Position {
  if (str.length !== 2) throw new Error("string must be of length 2")

  const file = hChar - str.charCodeAt(0)
  const rank = str.charCodeAt(1) - oneChar
  if (file < 0 || file > 7) throw new Error("file out of bounds")
  if (rank < 0 || rank > 7) throw new Error("rank out of bounds")

  return file + rank * 8
}
export function parseMove(str: string): Move {
  if (str.length !== 5) throw new Error()
  const parts = str.split(":")
  if (parts.length !== 2) throw new Error()
  return { from: stringToIndex(parts[0]), to: stringToIndex(parts[1]) }
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
export type Colour = typeof White | typeof Black

export function parseFen(fen: string): Board {
  const state = emptyBoardState()
  let stateIndex = 0
  let rowIndex = 0

  let wKing = false
  let bKing = false
  let boardStrLen = 0

  for (let strIndex = 0; strIndex < fen.length; ++strIndex) {
    const char = fen[strIndex]
    if (stateIndex == 64) {
      if (char != " ") throw new Error("space not found at end of pieces")

      boardStrLen = strIndex + 1
      break
    }

    const charCode = char.charCodeAt(0)
    if (charCode >= oneChar && charCode <= eightChar) {
      const delta = charCode - zeroChar
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

  let colour: Colour
  if (fen[boardStrLen] == "w") {
    colour = White
  } else if (fen[boardStrLen] == "b") {
    colour = Black
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
  if (colour == Black) {
    moveCounter += 1
  }

  return {
    state,
    colour,
    moveCounter,
    legalMoves: [],
    moveHistory: [],
  }
}
