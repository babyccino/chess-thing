package game_server

import (
	"testing"
	"time"

	"chess/auth"

	"github.com/google/uuid"
)

func TestGameClock(t *testing.T) {
	authServer := &auth.MockAuthServer{}

	server := NewGameServer(authServer)

	white := uuid.New()
	black := uuid.New()
	gameLength := 1 * time.Second
	increment := 0 * time.Second

	sessionId := server.NewSession(white, black, increment, gameLength)

	server.sessionsLock.Lock()
	session := server.sessions[sessionId]
	server.sessionsLock.Unlock()

	if session == nil {
		t.Fatal("Session not found")
	}

	whiteTime, blackTime := session.getClockState()
	if whiteTime < (gameLength - 50*time.Millisecond) {
		t.Errorf("Expected white time to be around %v, got %v", gameLength, whiteTime)
	}
	if blackTime != gameLength {
		t.Errorf("Expected black time to be %v, got %v", gameLength, blackTime)
	}

	// Wait for time to run out
	time.Sleep(1100 * time.Millisecond)

	// Wait longer for cleanup to complete
	time.Sleep(6 * time.Second)

	server.sessionsLock.Lock()
	_, exists := server.sessions[sessionId]
	server.sessionsLock.Unlock()

	if exists {
		t.Error("Session should have been removed due to time loss")
	}
}

func TestGameClockWithMoves(t *testing.T) {
	authServer := &auth.MockAuthServer{}

	server := NewGameServer(authServer)

	white := uuid.New()
	black := uuid.New()
	gameLength := 5 * time.Second
	increment := 1 * time.Second

	sessionId := server.NewSession(white, black, increment, gameLength)

	server.sessionsLock.Lock()
	session := server.sessions[sessionId]
	server.sessionsLock.Unlock()

	if session == nil {
		t.Fatal("Session not found")
	}

	initialWhiteTime, _ := session.getClockState()
	time.Sleep(100 * time.Millisecond)
	afterDelayWhiteTime, _ := session.getClockState()

	if afterDelayWhiteTime >= initialWhiteTime {
		t.Errorf("White's time should have decreased, but %v >= %v", afterDelayWhiteTime, initialWhiteTime)
	}

	// Test that increment is applied when clock is updated
	session.clockLock.Lock()
	session.whiteTime = 2 * time.Second                         // Set white's time to 2 seconds
	session.updatedAt = time.Now().Add(-500 * time.Millisecond) // Simulate 500ms elapsed
	session.clockLock.Unlock()

	session.updateClock()

	finalWhiteTime, _ := session.getClockState()
	expectedTime := 2*time.Second - 500*time.Millisecond + increment

	// Allow for some timing variance
	if finalWhiteTime < expectedTime-50*time.Millisecond || finalWhiteTime > expectedTime+50*time.Millisecond {
		t.Errorf("Expected white time to be around %v, got %v", expectedTime, finalWhiteTime)
	}

	// Clean up
	session.cleanup()
	server.RemoveSession(sessionId)
}
