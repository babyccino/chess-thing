package main

import (
	"chess/board"
	"fmt"
)

func main() {
	boardState := board.NewBoard()
	println("before:\n")
	boardState.Print()

	move1, _ := board.StringToPosition("A5")
	move2, _ := board.StringToPosition("A6")
	boardState.Move(move1, move2)
	println("after:\n")
	boardState.Print()
	println(boardState.Fen())

	fen := "KRBPP3/RQKP4/KBP5/PP5p/6pp/P4pbk/4pkqr/3ppbrk w 0"
	fenBoard, err := board.ParseFen(fen)

	if err != nil {
		panic(fmt.Errorf("%e", err))
	}

	fenBoard.Print()
}
