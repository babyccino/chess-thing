import { parseFen, type Board, parseMove, serialiseMove, type Position } from "./board"

export type ConnectEvent = {
  type: "connect"
  fen: string
  moveHistory?: string[]
  colour: "w" | "b"
  legalMoves?: string[]
}
export type ConnectOtherEvent = {
  type: "connect"
  colour: "w" | "b"
}
export type ConnectViewerEvent = {
  type: "connectViewer"
  fen: string
  moveHistory?: string[]
}
export type ConnectOtherViewerEvent = {
  type: "connectViewer"
}
export type MoveEvent = {
  type: "move"
  move: string
  fen: string
  legalMoves?: string[]
}
export type SendMoveEvent = {
  type: "sendMove"
  move: string
}
export type DuplicateSessionEvent = {
  type: "connect"
  fen: string
  moveHistory?: string[]
  colour: "w" | "b"
  legalMoves?: string[]
}
export type WinEvent = {
  type: "end"
  outcome: "win"
  victor: "w" | "b"
}
export type DrawEvent = {
  type: "end"
  outcome: "moveRuleDraw" | "stalemate" | "draw"
}
export type ChatEvent = {
  type: "chat"
  text: string
}
export type ErrorEvent = {
  type: "error"
  text: string
}
export type GameEvent =
  | ConnectEvent
  | ConnectViewerEvent
  | MoveEvent
  | SendMoveEvent
  | WinEvent
  | DrawEvent
  | ChatEvent
  | ErrorEvent

export function parseBoardState(event: ConnectEvent): Board {
  const board = parseFen(event.fen)
  board.legalMoves = event.legalMoves?.map(move => parseMove(move)) ?? []
  board.moveHistory = event.moveHistory?.map(move => parseMove(move)) ?? []
  return board
}

export function sendMove(from: Position, to: Position): SendMoveEvent {
  return { type: "sendMove", move: serialiseMove(from, to) }
}
