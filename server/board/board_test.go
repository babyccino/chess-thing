package board_test

import (
	"chess/board"
	"chess/utility"
	"fmt"
	"testing"
)

func Test_piece_functions(test *testing.T) {

	test.Run("test piece is", func(test *testing.T) {

		assertBoolEq(test, true, board.WKing.Is(board.King))
		assertBoolEq(test, false, board.WKing.Is(board.Queen))
	})
}
func Test_fen(test *testing.T) {

	test.Run("test fen creation", func(test *testing.T) {

		boardState := board.NewBoard()
		err := boardState.Init()
		assertSuccess(test, err)
		assertStrEquality(
			test,
			"krbpp3/rqnp4/nbp5/pp5P/p5PP/5PBN/4PNQR/3PPBRK w 0",
			boardState.Fen(),
		)

		err = boardState.MoveStr("H5", "H6")
		assertSuccess(test, err)
		assertStrEquality(
			test,
			"krbpp3/rqnp4/nbp5/pp5P/6PP/p4PBN/4PNQR/3PPBRK w 0",
			boardState.Fen(),
		)
	})

	test.Run("test test functions", func(test *testing.T) {

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

func Test_check(test *testing.T) {

	test.Run("test checks", func(test *testing.T) {
		helper := func(
			fen string,
			endingCheck *board.CheckState,
			shouldError bool,
			pinnedPieces PinMap,
		) {
			boardState, err := board.ParseFen(fen)
			assertSuccess(test, err)

			err = boardState.UpdateCheckState()

			if shouldError {
				assertFailure(test, err)
				return
			} else {
				assertSuccess(test, err)
			}

			check := &boardState.Check
			assertCheckEquality(test, endingCheck, check)

			for i := range 64 {
				pos := board.IndexToPosition(i)
				expectedPin, found := pinnedPieces[pos]
				piece := boardState.GetSquare(pos)
				receivedPin := piece.GetPin()
				if found {
					if receivedPin != expectedPin {
						test.Errorf(
							"The %s at %s was expected to be pinned %s but was pinned %s",
							piece.StringDebug(), pos.String(),
							board.PinToString(expectedPin), board.PinToString(receivedPin))
					}
				} else {
					if piece.IsPinned() {
						test.Errorf("The %s at %s was expected to not be pinned but was not",
							piece.StringDebug(), pos.String())
					}
				}
			}
		}

		helper("k6p/1PPPPPPP/8/8/8/8/8/7K w 0",
			&board.CheckState{board.NoCheck, board.Position{}},
			false,
			nil)

		// queen should be pinned
		helper("k7/1PP5/8/8/4b3/8/6Q1/7K w 0",
			&board.CheckState{board.NoCheck, board.Position{}},
			false,
			PinMap{{6, 6}: board.DownRightPin})

		// now a rook is between the queen and the bishop so the pin is broken
		helper("k7/P7/8/8/4b3/5R2/6Q1/7K w 0",
			&board.CheckState{
				board.WhiteCheck, board.Position{0, 1},
			},
			false,
			nil)

		helper("kP6/1P6/8/8/8/8/8/7K w 0",
			&board.CheckState{
				board.WhiteCheck, board.Position{1, 0},
			},
			false,
			nil)

		helper("kp6/p7/R1Q5/8/8/8/8/7K w 0",
			&board.CheckState{
				board.WhiteCheck, board.Position{2, 2},
			},
			false,
			PinMap{{0, 1}: board.DownPin})

		helper("kp6/p7/2Q5/8/8/8/8/7K w 0",
			&board.CheckState{
				board.WhiteCheck, board.Position{2, 2},
			},
			false,
			nil)

		helper("kp6/p7/1NQ5/8/8/8/8/7K w 0",
			&board.CheckState{
				board.WhiteDoubleCheck, board.Position{2, 2},
			},
			false,
			nil)

		helper("kp6/p7/B7/R7/8/8/8/7K w 0",
			&board.CheckState{board.NoCheck, board.Position{}},
			false,
			nil)

		helper("kp6/p7/1NQ5/8/8/8/8/6rK w 0", nil, true,
			nil)

		helper("kP6/8/8/8/8/1r6/8/7K w 0",
			&board.CheckState{board.WhiteCheck, board.Position{1, 0}},
			false,
			nil)
	})
}

func getExpectedMoves(test *testing.T, expectedMoves []string) []board.Move {
	test.Helper()
	parsedExpectedMoves := make([]board.Move, 0, len(expectedMoves))
	for _, move := range expectedMoves {
		parsedMove, err := board.DeserialiseMove(move)
		assertSuccess(test, err)
		parsedExpectedMoves = append(parsedExpectedMoves, parsedMove)
	}
	return parsedExpectedMoves
}

func Test_legal_moves(test *testing.T) {

	test.Run("test legal moves", func(test *testing.T) {

		helperFromBoard := func(
			boardState *board.BoardState,
			expectedMoves []string,
		) *board.BoardState {
			parsedExpectedMoves := getExpectedMoves(test, expectedMoves)

			fen := boardState.Fen()

			expectedMoveSet := utility.NewSet[board.Move]()
			for _, expectedMove := range parsedExpectedMoves {
				expectedMoveSet.Add(expectedMove)
			}
			if len(parsedExpectedMoves) != expectedMoveSet.Len() {
				test.Fatalf(
					"board: %s\nexpectedMoves contains duplicates: %s",
					fen,
					board.MoveListToString(parsedExpectedMoves),
				)
			}

			moves := boardState.GetLegalMoves()

			moveSet := utility.NewSet[board.Move]()
			for _, move := range moves {
				moveSet.Add(move)
			}
			if len(moves) != moveSet.Len() {
				test.Fatalf("board: %s\ncalculated moves contains duplicates: %v",
					fen, board.MoveListToString(moves))
			}

			if expectedMoveSet.Len() != moveSet.Len() {
				test.Fatalf(
					"board: %s\nexpected moves and received moves are not equal\nepxected: %s\ncalculated: %s\nin expected, not in calculated: %s\nvice versa: %s",
					fen,
					board.MoveListToString(parsedExpectedMoves),
					board.MoveListToString(moves),
					board.MoveListToString(expectedMoveSet.DiffArr(&moveSet)),
					board.MoveListToString(moveSet.DiffArr(&expectedMoveSet)),
				)
			}
			for move := range expectedMoveSet.Iter() {
				found := moveSet.Has(move)
				if !found {
					test.Fatalf("board: %s\nm%s was expected to be a legal move but was not\ncalculated: %v",
						fen, &move, board.MoveListToString(moves))
				}
			}

			return boardState
		}

		helper := func(
			fen string,
			expectedMoves []string,
		) *board.BoardState {
			parsedExpectedMoves := getExpectedMoves(test, expectedMoves)

			expectedMoveSet := utility.NewSet[board.Move]()
			for _, expectedMove := range parsedExpectedMoves {
				expectedMoveSet.Add(expectedMove)
			}
			if len(parsedExpectedMoves) != expectedMoveSet.Len() {
				test.Fatalf(
					"board: %s\nexpectedMoves contains duplicates: %s",
					fen,
					board.MoveListToString(parsedExpectedMoves),
				)
			}

			boardState, err := board.ParseFen(fen)
			assertSuccess(test, err)

			err = boardState.Init()

			moves := boardState.GetLegalMoves()

			moveSet := utility.NewSet[board.Move]()
			for _, move := range moves {
				moveSet.Add(move)
			}
			if len(moves) != moveSet.Len() {
				test.Fatalf("board: %s\ncalculated moves contains duplicates: %v",
					fen, board.MoveListToString(moves))
			}

			if expectedMoveSet.Len() != moveSet.Len() {
				test.Fatalf(
					"board: %s\nexpected moves and received moves are not equal\nepxected: %s\ncalculated: %s\nin expected, not in calculated: %s\nvice verse: %s",
					fen,
					board.MoveListToString(parsedExpectedMoves),
					board.MoveListToString(moves),
					board.MoveListToString(expectedMoveSet.DiffArr(&moveSet)),
					board.MoveListToString(moveSet.DiffArr(&expectedMoveSet)),
				)
			}

			for move := range expectedMoveSet.Iter() {
				found := moveSet.Has(move)
				if !found {
					test.Fatalf("board: %s\nm%s was expected to be a legal move but was not\ncalculated: %v",
						fen, &move, board.MoveListToString(moves))
				}
			}

			return boardState
		}

		// king moves
		_ = helper(
			"k7/8/8/8/8/8/8/7K w 0",
			[]string{"H1:H2", "H1:G2", "H1:G1"},
		)
		_ = helper(
			"k7/P7/8/8/8/8/8/7K w 0",
			[]string{"H1:H2", "H1:G2", "H1:G1"},
		)
		_ = helper(
			"k7/1P6/8/8/8/8/8/7K w 0",
			[]string{"H1:G2"},
		)
		_ = helper(
			"kP6/8/8/8/8/8/8/7K w 0",
			[]string{"H1:H2", "H1:G2", "H1:G1"},
		)
		_ = helper(
			"k7/1P6/1P6/8/8/8/8/7K w 0",
			[]string{},
		)
		//

		// king + others
		_ = helper(
			"k7/1p6/8/8/8/8/8/7K w 0",
			[]string{"H1:H2", "H1:G1", "G2:F3"},
		)
		_ = helper(
			"kp6/1P6/8/8/8/8/8/7K w 0",
			[]string{"H1:G2", "G1:G2", "G1:F2"},
		)

		_ = helper(
			"kp6/1P6/8/8/8/8/8/7K w 0",
			[]string{"H1:G2", "G1:G2", "G1:F2"},
		)
		//

		// checks
		_ = helper(
			"kP6/pp6/8/8/8/1r6/8/7K w 0",
			[]string{"H1:G1"},
		)
		//

		// starting position
		boardState := helper(
			"krbpp3/rqnp4/nbp5/pp5P/p5PP/5PBN/4PNQR/3PPBRK w 0",
			[]string{
				// pawn moves
				"D1:C2",
				"E1:D2",
				"E2:D3",
				"F3:E4",
				"G4:F5",
				"H4:G5",
				"H5:G6",
				// knight moves
				"F2:D3",
				"F2:E4",
				"H3:F4",
				"H3:G5",
				// bishop moves
				"G3:F4",
				"G3:E5",
				"G3:D6",
				"G3:C7",
			},
		)

		move, err := board.DeserialiseMove("D1:C2")
		assertSuccess(test, err)
		err = boardState.MakeMove(move)
		assertSuccess(test, err)

		helperFromBoard(
			boardState,
			[]string{
				// pawn moves
				"E8:F7",
				"D8:E7",
				"D7:E6",
				"C6:D5",
				"B5:C4",
				"A5:B4",
				"A4:B3",
				// knight moves
				"C7:E6",
				"C7:D5",
				"A6:C5",
				"A6:B4",
				// bishop moves
				"B6:C5",
				"B6:D4",
				"B6:E3",
				"B6:F2",
			},
		)

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
