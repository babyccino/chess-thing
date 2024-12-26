package board_test

import (
	"chess/board"
	"fmt"
	"testing"
)

func Test_fen(test *testing.T) {
	test.Parallel()

	test.Run("test fen creation", func(test *testing.T) {
		test.Parallel()

		boardState := board.NewBoard()
		assertStrEquality(test, "KRBPP3/RQNP4/NBP5/PP5p/P5pp/5pbn/4pnqr/3ppbrk w 0", boardState.Fen())

		move1, err := board.StringToPosition("A5")
		if err != nil {
			test.Fatal(err)
		}
		move2, err := board.StringToPosition("A6")
		if err != nil {
			test.Fatal(err)
		}
		boardState.Move(move1, move2)
		assertStrEquality(test, "KRBPP3/RQNP4/NBP5/PP5p/6pp/P4pbn/4pnqr/3ppbrk w 0", boardState.Fen())
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
	})
}

type PositionSet = map[board.Position]struct{}

func Test_check(test *testing.T) {
	test.Parallel()

	test.Run("test checks", func(test *testing.T) {
		test.Parallel()
		helper := func(
			fen string,
			endingCheck *board.CheckState,
			shouldError bool,
			pinnedPieces PositionSet,
		) {
			boardState, err := board.ParseFen(fen)
			assertSuccess(test, err)

			err = boardState.UpdateCheckState(shouldError)

			if shouldError {
				assertFailure(test, err)
				return
			} else {
				assertSuccess(test, err)
			}

			check := &boardState.Check
			assertCheckEquality(test, endingCheck, check)

			for i := range int8(64) {
				pos := board.IndexToPosition(i)
				_, found := pinnedPieces[pos]
				piece := boardState.GetSquare(pos)
				if found {
					if !piece.IsPinned() {
						test.Errorf("The %s at %s was expected to be pinned but was not",
							piece.StringDebug(), pos.String())
					}
				} else {
					if piece.IsPinned() {
						test.Errorf("The %s at %s was expected to not be pinned but was not",
							piece.StringDebug(), pos.String())
					}
				}
			}
		}

		helper("K6P/1ppppppp/8/8/8/8/8/7k w 0",
			&board.CheckState{board.NoCheck, board.Position{}},
			false,
			nil)

		// queen should be pinned
		helper("K7/1pp5/8/8/4B3/8/6q1/7k w 0",
			&board.CheckState{board.NoCheck, board.Position{}},
			false,
			PositionSet{{6, 6}: {}})

		// now a rook is between the queen and the bishop so the pin is broken
		helper("K7/p7/8/8/4B3/5r2/6q1/7k w 0",
			&board.CheckState{
				board.BlackCheck, board.Position{0, 1},
			},
			false,
			nil)

		helper("Kp6/1p6/8/8/8/8/8/7k w 0",
			&board.CheckState{
				board.BlackCheck, board.Position{1, 0},
			},
			false,
			nil)

		helper("KP6/P7/r1q5/8/8/8/8/7k w 0",
			&board.CheckState{
				board.BlackCheck, board.Position{2, 2},
			},
			false,
			PositionSet{{0, 1}: {}})

		helper("KP6/P7/2q5/8/8/8/8/7k w 0",
			&board.CheckState{
				board.BlackCheck, board.Position{2, 2},
			},
			false,
			nil)

		helper("KP6/P7/1nq5/8/8/8/8/7k w 0",
			&board.CheckState{
				board.BlackDoubleCheck, board.Position{2, 2},
			},
			false,
			nil)

		helper("KP6/P7/b7/r7/8/8/8/7k w 0",
			&board.CheckState{board.NoCheck, board.Position{}},
			false,
			nil)

		helper("KP6/P7/1nq5/8/8/8/8/6Rk w 0", nil, true,
			nil)
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
