package board_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"chess/board"
	"chess/utility"
)

func Test_piece_functions(test *testing.T) {
	test.Run("test piece is", func(test *testing.T) {
		test.Parallel()
		assertBoolEq(test, true, board.WKing.Is(board.King))
		assertBoolEq(test, false, board.WKing.Is(board.Queen))
	})
}

func Test_fen(test *testing.T) {
	test.Run("test fen creation", func(test *testing.T) {
		test.Parallel()
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

		wKing, bKing, err := boardState.GetKingPositions()
		assertSuccess(test, err)
		board.AssertPositionsEqual(test, *bKing, board.Position{0, 0})
		board.AssertPositionsEqual(test, *wKing, board.Position{7, 7})

		// need both kings
		received, err = board.ParseFen("8/8/8/8/8/8/8/7k w 0")
		assertFailure(test, err)

		boardState, err = board.ParseFen(
			"kB6/4p3/2b2r2/5R2/P1R1PnP1/2pP3Q/4Bn2/7K w 98")
		assertSuccess(test, err)
	})
}

type PinMap = map[board.Position]board.PinDirection

func Test_check(test *testing.T) {
	test.Run("test checks", func(test *testing.T) {
		test.Parallel()
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

func legalMovesHelper(
	test *testing.T,
	fen string,
	expectedMoves []string,
) *board.BoardState {
	test.Helper()
	boardState, err := board.ParseFen(fen)
	assertSuccess(test, err)

	err = boardState.Init()

	return helperFromBoardInner(test, boardState, expectedMoves, fen)
}

func legalMovesHelperFromBoard(
	test *testing.T,
	boardState *board.BoardState,
	expectedMoves []string,
) *board.BoardState {
	test.Helper()
	fen := boardState.Fen()
	return helperFromBoardInner(test, boardState, expectedMoves, fen)
}

func helperFromBoardInner(
	test *testing.T,
	boardState *board.BoardState,
	expectedMoves []string,
	fen string,
) *board.BoardState {
	parsedExpectedMoves := getExpectedMoves(test, expectedMoves)

	expectedMoveSet := utility.NewSet[board.Move]()
	for _, expectedMove := range parsedExpectedMoves {
		expectedMoveSet.Add(expectedMove)
	}
	if len(parsedExpectedMoves) != expectedMoveSet.Len() {
		test.Log(boardState.String())
		test.Fatalf(
			"board: %s\nexpectedMoves contains duplicates: %s",
			fen,
			board.MoveListToString(parsedExpectedMoves),
		)
	}

	moves := boardState.LegalMoves

	moveSet := utility.NewSet[board.Move]()
	for _, move := range moves {
		moveSet.Add(move)
	}
	if len(moves) != moveSet.Len() {
		test.Log(boardState.String())
		test.Fatalf("board: %s\ncalculated moves contains duplicates: %v",
			fen, board.MoveListToString(moves))
	}

	if expectedMoveSet.Len() != moveSet.Len() {
		test.Fatalf(
			"board: \n%s\nexpected moves and received moves are not equal\nepxected: %s\ncalculated: %s\nin expected, not in calculated: %s\nvice versa: %s",
			boardState.String(),
			board.MoveListToString(parsedExpectedMoves),
			board.MoveListToString(moves),
			board.MoveListToString(expectedMoveSet.DiffArr(&moveSet)),
			board.MoveListToString(moveSet.DiffArr(&expectedMoveSet)),
		)
	}
	for move := range expectedMoveSet.Iter() {
		found := moveSet.Has(move)
		if !found {
			test.Log(boardState.String())
			test.Fatalf("board: %s\nm%s was expected to be a legal move but was not\ncalculated: %v",
				fen, &move, board.MoveListToString(moves))
		}
	}

	return boardState
}

func findIllegalMove(
	test *testing.T,
	fen string,
	illegalMove string,
) *board.BoardState {
	test.Helper()
	boardState, err := board.ParseFen(fen)
	assertSuccess(test, err)

	err = boardState.Init()
	moves := boardState.LegalMoves

	parsedMove, err := board.DeserialiseMove(illegalMove)
	assertSuccess(test, err)

	for _, move := range moves {
		if move == parsedMove {
			test.Fatalf(
				"board: %s\nillegal move found: %s in %s",
				boardState.String(),
				illegalMove,
				board.MoveListToString(moves),
			)
		}
	}
	return boardState
}

const errStr = `err: %v
fen: %s
board failed making move %s then %s after %d moves
prev move list: %s
then move list: %s
%s
%s
%s`

func Test_legal_moves(test *testing.T) {
	test.Run("test legal moves", func(test *testing.T) {
		test.Parallel()
		// king moves
		_ = legalMovesHelper(
			test,
			"k7/8/8/8/8/8/8/7K w 0",
			[]string{"H1:H2", "H1:G2", "H1:G1"},
		)
		_ = legalMovesHelper(
			test,
			"k7/P7/8/8/8/8/8/7K w 0",
			[]string{"H1:H2", "H1:G2", "H1:G1"},
		)
		_ = legalMovesHelper(
			test,
			"k7/1P6/8/8/8/8/8/7K w 0",
			[]string{"H1:G2"},
		)
		_ = legalMovesHelper(
			test,
			"kP6/8/8/8/8/8/8/7K w 0",
			[]string{"H1:H2", "H1:G2", "H1:G1"},
		)
		_ = legalMovesHelper(
			test,
			"k7/1P6/1P6/8/8/8/8/7K w 0",
			[]string{},
		)
		//

		// king + others
		_ = legalMovesHelper(
			test,
			"k7/1p6/8/8/8/8/8/7K w 0",
			[]string{"H1:H2", "H1:G1", "G2:F3"},
		)
		_ = legalMovesHelper(
			test,
			"kp6/1P6/8/8/8/8/8/7K w 0",
			[]string{"H1:G2", "G1:G2", "G1:F2"},
		)

		_ = legalMovesHelper(
			test,
			"kp6/1P6/8/8/8/8/8/7K w 0",
			[]string{"H1:G2", "G1:G2", "G1:F2"},
		)
		//

		// checks
		_ = legalMovesHelper(
			test,
			"kP6/nn6/8/8/8/1r6/8/7K w 0",
			[]string{"H1:G1"},
		)

		boardState := legalMovesHelper(
			test,
			"k6R/pp6/8/8/8/1r6/8/7K w 0",
			[]string{},
		)
		winner := boardState.HasWinner()
		if winner != board.BlackWin {
			test.Fatalf("expected black winner\nreceived: %d",
				winner)
		}

		// x-ray attack on king
		legalMovesHelper(
			test,
			"1rb5/5N2/1Q1P2p1/ppk4P/p2R1n2/1P5n/2B1PN1R/4P2K w 92",
			[]string{"F4:G3"},
		)

		//     . ♔ .          1
		//     .   .          2
		//                    3
		//   ♙ ♙              4
		//   ♙   .            5
		//       .            6
		//       ♜            7
		//                 ♚  8
		// long pawn move to block check
		legalMovesHelper(
			test,
			"2k5/8/8/pp6/p7/8/2R5/7K w 0",
			[]string{
				"F1:G1",
				"F1:G2",
				"F1:E1",
				"F1:E2",
				"G4:F5",
				"H4:F6",
			},
		)
		//

		// pinned piece
		findIllegalMove(test, "kq3R2/r2p4/1rR2P2/p1n5/3bp1B1/2p2P2/2p1P3/3P3K w 84", "G1:A7")

		findIllegalMove(test, "k5b1/qB2P3/4p1p1/2P2p1P/6PR/2p5/1B6/R1n4K w 70", "H2:G2")
		//

		// starting position
		boardState = legalMovesHelper(
			test,
			"krbpp3/rqnp4/nbp5/pp5P/p5PP/5PBN/4PNQR/3PPBRK w 0",
			[]string{
				// pawn moves
				"D1:C2",
				"D1:B3",
				"E1:D2",
				"E1:C3",
				"E2:D3",
				"E2:C4",
				"F3:E4",
				"F3:D5",
				"G4:F5",
				"G4:E6",
				"H4:G5",
				"H4:F6",
				"H5:G6",
				"H5:F7",
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

		_ = legalMovesHelperFromBoard(
			test,
			boardState,
			[]string{
				// pawn moves
				"E8:F7",
				"E8:G6",
				"D8:E7",
				"D8:F6",
				"D7:E6",
				"D7:F5",
				"C6:D5",
				"C6:E4",
				"B5:C4",
				"B5:D3",
				"A5:B4",
				"A5:C3",
				"A4:B3",
				"A4:C2",
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
		//
	})

	test.Run("test random legal moves from start position", func(test *testing.T) {
		test.Parallel()
		boardState := board.NewBoard()

		err := boardState.Init()
		assertSuccess(test, err)

		whoseMove := boardState.WhoseMove()
		if whoseMove != board.White {
			test.Fatalf("expected white\nreceived: %s", board.ColourString(whoseMove))
		}

		previousState := boardState.String()
		previousLegalMoves := board.MoveListToString(boardState.LegalMoves)
		previousMove := board.Move{}
		drawCount := 0
		bWinCount := 0
		wWinCount := 0
		for i := range 100000 {
			state := boardState.String()
			moves := boardState.LegalMoves
			move := moves[rand.IntN(len(moves))]

			legalMovesStr := board.MoveListToString(moves)

			// todo show board image instead of fen
			err := boardState.MakeMove(move)
			if err != nil {
				afterErr := boardState.String()
				test.Fatalf(
					errStr,
					err,
					boardState.Fen(),
					previousMove.Serialise(),
					move.Serialise(),
					i,
					previousLegalMoves,
					legalMovesStr,
					previousState,
					state,
					afterErr,
				)
			}

			previousMove = move
			previousState = state
			previousLegalMoves = legalMovesStr

			win := boardState.HasWinner()
			if win != board.NoWin {
				test.Logf("winner was found resetting board after %d moves", i)
				boardState = board.NewBoard()
				err := boardState.Init()
				assertSuccess(test, err)

				whoseMove = board.None

				switch win {
				case board.WhiteWin:
					wWinCount += 1
				case board.BlackWin:
					bWinCount += 1
				case board.MoveRuleDraw:
					fallthrough
				case board.Stalemate:
					drawCount += 1
				}
			}

			newMove := boardState.WhoseMove()
			if (whoseMove == board.White || newMove == board.Black) &&
				(whoseMove == board.Black || newMove == board.White) {
				test.Fatal("Whose move it is did not change")
			}
			whoseMove = newMove
		}

		// logging
		// test.Fatalf("white wins: %d, black wins: %d, draws: %d",
		// 	wWinCount, bWinCount, drawCount)
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
