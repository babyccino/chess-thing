package main

import boardState "chess/board"

func main() {
	board := boardState.NewBoard()
	println("before:\n")
	board.Print()

	move1, _ := boardState.StringToPosition("A5")
	move2, _ := boardState.StringToPosition("A6")
	board.Move(move1, move2)
	println("after:\n")
	board.Print()
	println(board.Fen())

	fen := "KRBPP3/RQKP4/KBP5/PP5p/6pp/P4pbk/4pkqr/3ppbrk w"
	fenBoard, err := boardState.ParseFen(fen)
	fenBoard.Print()
}
