package board_test

import (
	"chess/board"
	"fmt"
	"strings"
	"testing"
)

func Test_piece_functions(test *testing.T) {
	test.Parallel()

	test.Run("test piece is", func(test *testing.T) {
		test.Parallel()

		assertBoolEq(test, true, board.WKing.Is(board.King))
		assertBoolEq(test, false, board.WKing.Is(board.Queen))
	})
}
func Test_fen(test *testing.T) {
	test.Parallel()

	test.Run("test fen creation", func(test *testing.T) {
		test.Parallel()

		boardState := board.NewBoard()
		assertStrEquality(
			test,
			"krbpp3/rqnp4/nbp5/pp5P/p5PP/5PBN/4PNQR/3PPBRK w 0",
			boardState.Fen(),
		)

		err := boardState.MoveStr("A5", "A6")
		assertSuccess(test, err)
		assertStrEquality(
			test,
			"krbpp3/rqnp4/nbp5/pp5P/6PP/p4PBN/4PNQR/3PPBRK w 0",
			boardState.Fen(),
		)
	})

	test.Run("test test functions", func(test *testing.T) {
		test.Parallel()

		boardState1 := board.NewBoard()
		boardState2 := board.NewBoard()

		assertBoardEquality(test, boardState1, boardState2)

		move1, _ := board.StringToPosition("A5")
		move2, _ := board.StringToPosition("A6")
		boardState1.Move(move1, move2)
		boardState2.Move(move1, move2)

		assertBoardEquality(test, boardState1, boardState2)
	})

	test.Run("test fen parsing", func(test *testing.T) {
		test.Parallel()

		received, err := board.ParseFen("KRBPP3/RQNP4/NBP5/PP5p/P5pp/5pbn/4pnqr/3ppbrk w 0")
		assertSuccess(test, err)
		expected := board.NewBoard()

		assertBoardEquality(test, expected, received)

		boardState, err := board.ParseFen(
			"K7/2n5/8/8/8/8/8/7k w 0")
		assertSuccess(test, err)

		wKing, bKing := boardState.GetKingPositions()
		board.AssertPositionsEqual(test, *bKing, board.Position{0, 0})
		board.AssertPositionsEqual(test, *wKing, board.Position{7, 7})

		// need both kings
		received, err = board.ParseFen("8/8/8/8/8/8/8/7k w 0")
		assertFailure(test, err)
	})
}

type PinMap = map[board.Position]board.PinDirection

// func Test_check(test *testing.T) {
// 	test.Parallel()

// 	test.Run("test checks", func(test *testing.T) {
// 		test.Parallel()
// 		helper := func(
// 			fen string,
// 			endingCheck *board.CheckState,
// 			shouldError bool,
// 			pinnedPieces PinMap,
// 		) {
// 			boardState, err := board.ParseFen(fen)
// 			assertSuccess(test, err)

// 			err = boardState.UpdateCheckState(shouldError)

// 			if shouldError {
// 				assertFailure(test, err)
// 				return
// 			} else {
// 				assertSuccess(test, err)
// 			}

// 			check := &boardState.Check
// 			assertCheckEquality(test, endingCheck, check)

// 			for i := range 64 {
// 				pos := board.IndexToPosition(i)
// 				expectedPin, found := pinnedPieces[pos]
// 				piece := boardState.GetSquare(pos)
// 				receivedPin := piece.GetPin()
// 				if found {
// 					if receivedPin != expectedPin {
// 						test.Errorf(
// 							"The %s at %s was expected to be pinned %s but was pinned %s",
// 							piece.StringDebug(), pos.String(),
// 							board.PinToString(expectedPin), board.PinToString(receivedPin))
// 					}
// 				} else {
// 					if piece.IsPinned() {
// 						test.Errorf("The %s at %s was expected to not be pinned but was not",
// 							piece.StringDebug(), pos.String())
// 					}
// 				}
// 			}
// 		}

// 		helper("K6P/1ppppppp/8/8/8/8/8/7k w 0",
// 			&board.CheckState{board.NoCheck, board.Position{}},
// 			false,
// 			nil)

// 		// queen should be pinned
// 		helper("K7/1pp5/8/8/4B3/8/6q1/7k w 0",
// 			&board.CheckState{board.NoCheck, board.Position{}},
// 			false,
// 			PinMap{{6, 6}: board.DownRightPin})

// 		// now a rook is between the queen and the bishop so the pin is broken
// 		helper("K7/p7/8/8/4B3/5r2/6q1/7k w 0",
// 			&board.CheckState{
// 				board.BlackCheck, board.Position{0, 1},
// 			},
// 			false,
// 			nil)

// 		helper("Kp6/1p6/8/8/8/8/8/7k w 0",
// 			&board.CheckState{
// 				board.BlackCheck, board.Position{1, 0},
// 			},
// 			false,
// 			nil)

// 		helper("KP6/P7/r1q5/8/8/8/8/7k w 0",
// 			&board.CheckState{
// 				board.BlackCheck, board.Position{2, 2},
// 			},
// 			false,
// 			PinMap{{0, 1}: board.DownPin})

// 		helper("KP6/P7/2q5/8/8/8/8/7k w 0",
// 			&board.CheckState{
// 				board.BlackCheck, board.Position{2, 2},
// 			},
// 			false,
// 			nil)

// 		helper("KP6/P7/1nq5/8/8/8/8/7k w 0",
// 			&board.CheckState{
// 				board.BlackDoubleCheck, board.Position{2, 2},
// 			},
// 			false,
// 			nil)

// 		helper("KP6/P7/b7/r7/8/8/8/7k w 0",
// 			&board.CheckState{board.NoCheck, board.Position{}},
// 			false,
// 			nil)

// 		helper("KP6/P7/1nq5/8/8/8/8/6Rk w 0", nil, true,
// 			nil)

// 		helper("kP6/8/8/8/8/1r6/8/7K w 0",
// 			&board.CheckState{board.WhiteCheck, board.Position{1, 0}},
// 			false,
// 			nil)
// 	})
// }

func Test_legal_moves(test *testing.T) {
	test.Parallel()

	test.Run("test legal moves", func(test *testing.T) {
		test.Parallel()

		helper := func(
			fen string,
			expectedMoves []string,
		) {
			parsedExpectedMoves := make([]board.Move, 0, len(expectedMoves))
			for _, move := range expectedMoves {
				split := strings.Split(move, ":")
				if len(split) != 2 {
					test.Fatalf("error in arg: %s", move)
				}

				from, err := board.StringToPosition(split[0])
				assertSuccess(test, err)
				to, err := board.StringToPosition(split[1])
				assertSuccess(test, err)
				parsedMove := board.Move{from, to}
				parsedExpectedMoves = append(parsedExpectedMoves, parsedMove)
			}

			expectedMoveMap := map[board.Move]struct{}{}
			for _, expectedMove := range parsedExpectedMoves {
				expectedMoveMap[expectedMove] = struct{}{}
			}
			if len(parsedExpectedMoves) != len(expectedMoveMap) {
				test.Fatalf(
					"board: %s\nexpectedMoves contains duplicates: %s",
					fen,
					board.MoveListToString(parsedExpectedMoves),
				)
			}

			boardState, err := board.ParseFen(fen)
			assertSuccess(test, err)

			err = boardState.UpdateCheckState(false)
			assertSuccess(test, err)
			boardState.UpdateAttackedSquares()

			moves := boardState.GetLegalMoves()

			moveMap := map[board.Move]struct{}{}
			for _, move := range moves {
				moveMap[move] = struct{}{}
			}
			if len(moves) != len(moveMap) {
				test.Fatalf("board: %s\nmoves contains duplicates: %v", fen, board.MoveListToString(moves))
			}

			if len(expectedMoveMap) != len(moveMap) {
				test.Fatalf(
					"board: %s\nexpected moves and received moves are not equal\nepxected: %v\ncalculated: %v",
					fen,
					board.MoveListToString(parsedExpectedMoves), board.MoveListToString(moves))
			}

			for move := range expectedMoveMap {
				_, found := moveMap[move]
				if !found {
					test.Fatalf("board: %s\nm%s was expected to be a legal move but was not\ncalculated: %v",
						fen, &move, board.MoveListToString(moves))
				}
			}
		}

		// // king moves
		// helper(
		// 	"k7/8/8/8/8/8/8/7K w 0",
		// 	[]string{"A1:A2", "A1:B2", "A1:B1"},
		// )
		// helper(
		// 	"k7/P7/8/8/8/8/8/7K w 0",
		// 	[]string{"A1:A2", "A1:B2", "A1:B1"},
		// )
		// helper(
		// 	"k7/1P6/8/8/8/8/8/7K w 0",
		// 	[]string{"A1:B2"},
		// )
		// helper(
		// 	"kP6/8/8/8/8/8/8/7K w 0",
		// 	[]string{"A1:A2", "A1:B2", "A1:B1"},
		// )
		// helper(
		// 	"k7/1P6/1P6/8/8/8/8/7K w 0",
		// 	[]string{},
		// )
		// //

		// // king + others
		// helper(
		// 	"k7/1p6/8/8/8/8/8/7K w 0",
		// 	[]string{"A1:A2", "A1:B1", "B2:C3"},
		// )
		// helper(
		// 	"kp6/1P6/8/8/8/8/8/7K w 0",
		// 	[]string{"A1:B2", "B1:B2", "B1:C2"},
		// )
		// //

		// checks
		helper(
			"kP6/8/8/8/8/1r6/8/7K w 0",
			[]string{"A1:B1", "A1:A2", "A1:B2", "B6:B1"},
		)
		//
	})
}

func assertEq(test *testing.T, expected, received fmt.Stringer) {
	test.Helper()
	if expected != received {
		test.Fatalf("expected %s\nreceived: %s",
			expected.String(), received.String())
	}
}

func assertSuccess(test *testing.T, err error) {
	test.Helper()
	if err != nil {
		test.Fatal(err)
	}
}

func assertFailure(test *testing.T, err error) {
	test.Helper()
	if err == nil {
		test.Fatal("no error found")
	}
}

func assertBoardEquality(test *testing.T, expected, received *board.BoardState) {
	test.Helper()

	equal := true
	if expected.CaptureMoveCounter != expected.CaptureMoveCounter {
		equal = false
		test.Errorf("expected CaptureMoveCounter: %d, received: %d",
			expected.CaptureMoveCounter, received.CaptureMoveCounter)
	}
	if expected.MoveCounter != expected.MoveCounter {
		equal = false
		test.Errorf("expected MoveCounter: %d, received: %d",
			expected.MoveCounter, received.MoveCounter)
	}
	if expected.State != expected.State {
		equal = false
		test.Errorf("expected State: %v, received: %v",
			expected.State, received.State)
	}
	if expected.Check != expected.Check {
		equal = false
		test.Errorf("expected Check: %s, received: %s",
			expected.Check.String(), received.Check.String())
	}

	if equal {
		return
	}

	test.Errorf("expected:\n%sreceived:\n%s",
		expected.String(), received.String())

	test.Errorf("expected:\n%sreceived:\n%s",
		expected.String(), received.String())

	test.FailNow()
}

func assertStrEquality(test *testing.T, expected, received string) {
	test.Helper()
	if expected != received {
		test.Fatalf("expected:\n%s\nreceived:\n%s",
			expected, received)
	}
}

func assertCheckEquality(test *testing.T, expected, received *board.CheckState) {
	test.Helper()
	if *expected != *received {
		test.Fatalf("expected: %s\nreceived: %s",
			expected.String(), received.String())
	}
}

type Number interface {
	int | int32 | int64 | int16 | int8
}

func assertNumEq[T Number](test *testing.T, expected, received T) {
	test.Helper()
	if expected != received {
		test.Fatalf("expected %d\nreceived: %d",
			expected, received)
	}
}
func assertBoolEq(test *testing.T, expected, received bool) {
	test.Helper()
	if expected != received {
		test.Fatalf("expected %t\nreceived: %t",
			expected, received)
	}
}
