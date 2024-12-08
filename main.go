package main

import boardPkg "chess/board"

func main() {
	board := boardPkg.NewBoard()
	println("before:\n")
	board.Print()
	move1, _ := boardPkg.StringToPosition("A5")
	move2, _ := boardPkg.StringToPosition("A6")
	board.Move(move1, move2)
	println("after:\n")
	board.Print()
}
