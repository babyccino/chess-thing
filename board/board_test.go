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

func Test_check(test *testing.T) {
	test.Parallel()

	test.Run("test knight checks", func(test *testing.T) {
		test.Parallel()
		boardState, err := board.ParseFen(
			"K7/2n5/8/8/8/8/8/7k w 0")
		assertSuccess(test, err)

		wKing, bKing := boardState.GetKingPositions()
		check, err := boardState.CheckKnightChecks(wKing, bKing, true)
		assertSuccess(test, err)
		assertEq(test,
			&board.CheckState{board.BlackCheck, board.Position{2, 1}},
			check,
		)

		boardState, err = board.ParseFen(
			"K7/2N5/8/8/8/8/6n1/7k w 0")
		wKing, bKing = boardState.GetKingPositions()
		check, err = boardState.CheckKnightChecks(wKing, bKing, true)
		assertSuccess(test, err)
		assertEq(test,
			&board.CheckState{board.NoCheck, board.Position{}},
			check,
		)

		boardState, err = board.ParseFen(
			"K7/2n5/8/8/8/8/6n1/7k w 0")
		wKing, bKing = boardState.GetKingPositions()
		check, err = boardState.CheckKnightChecks(wKing, bKing, true)
		assertSuccess(test, err)
		assertEq(test,
			&board.CheckState{board.BlackCheck, board.Position{2, 1}},
			check,
		)

		boardState, err = board.ParseFen(
			"K7/2n5/8/8/8/8/5N2/7k w 0")
		wKing, bKing = boardState.GetKingPositions()
		_, err = boardState.CheckKnightChecks(wKing, bKing, true)
		assertFailure(test, err)

		boardState, err = board.ParseFen(
			"K7/2n5/1n6/8/8/8/5N2/7k w 0")
		wKing, bKing = boardState.GetKingPositions()
		_, err = boardState.CheckKnightChecks(wKing, bKing, true)
		assertFailure(test, err)
	})

	test.Run("test find piece in direction", func(test *testing.T) {
		test.Parallel()
		boardState, err := board.ParseFen(
			"K6P/pppppppp/8/8/8/8/8/7k w 0")
		assertSuccess(test, err)

		piece, pos := boardState.CheckInDirection(
			board.RightVec,
			&board.Position{0, 0},
		)
		assertSuccess(test, err)
		assertPieceInDirectionEquality(test,
			board.BPawn,
			piece,
			board.Position{7, 0},
			pos)

		boardState, err = board.ParseFen(
			"K7/pppppppp/8/8/8/8/8/7k w 0")
		assertSuccess(test, err)

		piece, pos = boardState.CheckInDirection(
			board.RightVec,
			&board.Position{0, 0},
		)
		assertSuccess(test, err)
		assertPieceInDirectionEquality(test,
			board.Clear,
			piece,
			board.Position{},
			pos)

		boardState, err = board.ParseFen(
			"K7/P7/8/8/8/5q2/7p/6pk w 0")
		assertSuccess(test, err)

		piece, pos = boardState.CheckInDirection(
			board.UpLeftVec,
			&board.Position{7, 7},
		)
		assertSuccess(test, err)
		assertPieceInDirectionEquality(test,
			board.WQueen,
			piece,
			board.Position{5, 5},
			pos)
	})
}

func assertEq(test *testing.T, expected, received fmt.Stringer) {
	test.Helper()
	if expected != received {
		test.Fatalf("expected: %s\nreceived: %s",
			expected.String(), received.String())
	}
}

func assertPieceInDirectionEquality(
	test *testing.T,
	expectedPiece,
	recievedPiece board.Piece,
	expectedPosition,
	recievedPosition board.Position,
) {
	test.Helper()
	if expectedPiece != recievedPiece || expectedPosition != recievedPosition {
		test.Fatalf("expected piece %s at pos: %s, got %s at %s",
			expectedPiece.String(),
			expectedPosition.String(),
			recievedPiece.String(),
			recievedPosition.String())
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
		test.Errorf("expected Check: %s, received: %v",
			expected.Check, received.Check)
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
