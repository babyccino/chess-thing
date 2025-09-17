package game_server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"chess/auth"
	"chess/board"
	"chess/utility"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type SessionMap = map[uuid.UUID]*Session
type GameServer struct {
	ServeMux     *http.ServeMux
	sessionsLock sync.Mutex
	sessions     SessionMap
	authServer   auth.AuthStrategy
}

type Session struct {
	id             uuid.UUID
	boardStateLock sync.Mutex
	boardState     *board.BoardState

	subscriberLock sync.Mutex
	players        [2]*subscriber
	viewers        utility.Set[*subscriber]

	increment  time.Duration
	gameLength time.Duration

	clockLock  sync.Mutex
	whiteTime  time.Duration
	blackTime  time.Duration
	clockTimer *time.Timer

	server    *GameServer
	updatedAt time.Time
	createdAt time.Time
}

type ConnectionState int8

const (
	PreConnected ConnectionState = iota
	Connected
	Disconnected
	Closed
)

type subscriber struct {
	userId           uuid.UUID
	events           chan Event
	doneChannel      chan struct{}
	reconnectChannel chan struct{}
	Conn             *websocket.Conn
	state            ConnectionState
	session          *Session
	colour           board.Colour
}

func NewSubscriber(
	userId uuid.UUID,
	session *Session,
	colour board.Colour,
) *subscriber {
	return &subscriber{
		userId:           userId,
		events:           make(chan Event, 10),
		doneChannel:      make(chan struct{}),
		reconnectChannel: make(chan struct{}),
		session:          session,
		colour:           colour,
		state:            PreConnected,
	}
}
func (subscriber *subscriber) init(Conn *websocket.Conn) {
	subscriber.Conn = Conn
	subscriber.state = Connected
}

func NewGameServer(authServer auth.AuthStrategy) *GameServer {
	server := &GameServer{
		ServeMux:     http.NewServeMux(),
		sessions:     make(SessionMap),
		sessionsLock: sync.Mutex{},
		authServer:   authServer,
	}

	server.ServeMux.HandleFunc("/subscribe/", server.SubscribeHandler)

	return server
}

func newSession(
	white uuid.UUID,
	black uuid.UUID,
	increment time.Duration,
	gameLength time.Duration,
	server *GameServer,
) *Session {
	boardState := board.NewBoard()
	err := boardState.Init()
	if err != nil {
		panic(err)
	}

	session := &Session{
		id:             uuid.New(),
		boardState:     boardState,
		boardStateLock: sync.Mutex{},

		subscriberLock: sync.Mutex{},
		players:        [2]*subscriber{},
		viewers:        utility.NewSet[*subscriber](),

		increment:  increment,
		gameLength: gameLength,

		clockLock: sync.Mutex{},
		whiteTime: gameLength,
		blackTime: gameLength,

		server:    server,
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}

	session.players[0] = NewSubscriber(white, session, board.White)
	session.players[1] = NewSubscriber(black, session, board.Black)

	return session
}

func (server *GameServer) NewSession(
	white uuid.UUID,
	black uuid.UUID,
	increment time.Duration,
	gameLength time.Duration,
) uuid.UUID {
	server.sessionsLock.Lock()
	defer server.sessionsLock.Unlock()

	session := newSession(white, black, increment, gameLength, server)
	server.sessions[session.id] = session
	return session.id
}

func (server *GameServer) OnShutdown() {
	// todo
}

func (server *GameServer) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	authenticated, err := server.authServer.IsAuthenticated(ctx, writer, req)
	if err != nil {
		return
	}
	if !authenticated {
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}
	server.ServeMux.ServeHTTP(writer, req)
}

func logError(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "error", slog.Any("error", err))
}

type eventType = string

const (
	connect       eventType = "connect"
	reconnect               = "reconnect"
	disconnect              = "disconnect"
	connectViewer           = "connectViewer"
	move                    = "move"
	end                     = "end"
	errorEvent              = "error"
	abort                   = "abort"
)

type Event struct {
	Type        eventType `json:"type"`
	Fen         *string   `json:"fen,omitempty"`
	MoveHistory *[]string `json:"moveHistory,omitempty"`
	Colour      *string   `json:"colour,omitempty"`
	Move        *string   `json:"move,omitempty"`
	LegalMoves  *[]string `json:"legalMoves,omitempty"`
	Outcome     *string   `json:"outcome,omitempty"`
	Victor      *string   `json:"victor,omitempty"`
	Text        *string   `json:"text,omitempty"`
	WhiteTime   *int32    `json:"whiteTime,omitempty"` // Time in milliseconds
	BlackTime   *int32    `json:"blackTime,omitempty"` // Time in milliseconds
}

func moveList(moves []board.Move) []string {
	retMoves := make([]string, len(moves))
	for i, move := range moves {
		retMoves[i] = move.Serialise()
	}
	return retMoves
}

func serialiseColour(colour board.Colour) string {
	var retColour string
	if colour == board.White {
		retColour = "w"
	} else if colour == board.Black {
		retColour = "b"
	} else {
		retColour = "v"
	}
	return retColour
}

func (server *GameServer) SubscribeHandler(
	writer http.ResponseWriter,
	req *http.Request,
) {
	ctx := req.Context()
	gameId, err := getId(writer, req)
	if err != nil {
		logError(ctx, err)
		return
	}

	// todo getting back a lot of useless data
	authSession, err := server.authServer.GetUserSession(ctx, writer, req)
	if err != nil {
		logError(ctx, err)
		return
	}

	slog.InfoContext(ctx, "subscribing user",
		slog.String("email", authSession.UserEmail),
		slog.String("gameid", gameId.String()))

	server.sessionsLock.Lock()
	session, found := server.sessions[gameId]
	server.sessionsLock.Unlock()

	if !found {
		// todo accept header
		writer.WriteHeader(404)
		writer.Write([]byte(""))
		logError(ctx, errors.New("not found"))
		return
	}

	session.subscriberLock.Lock()
	sub, colour := session.getSubscriber(ctx, authSession.UserID)
	session.subscriberLock.Unlock()

	if colour >= board.White && sub.state == Connected {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte("already connected"))
		logError(ctx, errors.New("already connected"))
		return
	}

	state := sub.state
	if state == Disconnected {
		sub.reconnectChannel <- struct{}{}
	}

	// todo accept header
	conn, err := websocket.Accept(writer, req, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		logError(ctx, err)
		return
	}

	sub.init(conn)

	ctx = context.WithoutCancel(ctx)

	subEvent, eventForOthers := session.CreateConnectEvent(colour, state)

	err = sub.write(ctx, subEvent)
	if err != nil {
		sub.closeNow(ctx, err)
		logError(ctx, err)
		return
	}

	session.publish(sub, eventForOthers)

	if colour != board.None {
		go sub.initRead(ctx)
	}
	go sub.initWrite(ctx)
}

// Doesn't lock
func (session *Session) getSubscriber(
	ctx context.Context,
	userId uuid.UUID,
) (*subscriber, board.Colour) {
	if userId == session.players[0].userId {
		slog.InfoContext(ctx, "added client to session as white player", slog.String("id", userId.String()))
		return session.players[0], board.White
	} else if userId == session.players[1].userId {
		slog.InfoContext(ctx, "added client to session as black player", slog.String("id", userId.String()))
		return session.players[1], board.Black
	}

	slog.InfoContext(ctx, "added client to session as viewer", slog.String("id", userId.String()))
	sub := NewSubscriber(userId, session, board.None)
	session.viewers.Add(sub)
	return sub, board.None
}

func (session *Session) CreateConnectEvent(
	colour board.Colour,
	connectionState ConnectionState,
) (subEvent Event, otherEvent Event) {
	fen := session.boardState.Fen()
	whiteTime, blackTime := session.getClockState()
	whiteTimeMs := int32(whiteTime.Milliseconds())
	blackTimeMs := int32(blackTime.Milliseconds())

	if colour == board.None {
		list := moveList(session.boardState.MoveHistory)
		subEvent = Event{
			Type:        connectViewer,
			Fen:         &fen,
			MoveHistory: &list,
			WhiteTime:   &whiteTimeMs,
			BlackTime:   &blackTimeMs,
		}
		otherEvent = Event{
			Type: connectViewer,
		}
	} else {
		var connectionType string
		if connectionState == Disconnected {
			connectionType = reconnect
		} else {
			connectionType = connect
		}

		history := moveList(session.boardState.LegalMoves)
		colour := serialiseColour(colour)
		legalMoves := moveList(session.boardState.LegalMoves)
		subEvent = Event{
			Type:        connectionType,
			Fen:         &fen,
			MoveHistory: &history,
			Colour:      &colour,
			LegalMoves:  &legalMoves,
			WhiteTime:   &whiteTimeMs,
			BlackTime:   &blackTimeMs,
		}
		otherEvent = Event{
			Type:   connectionType,
			Colour: &colour,
		}
	}
	return subEvent, otherEvent
}

func (session *Session) DeleteSubscriber(sub *subscriber) {
	if session.players[0] == sub {
		session.handleWin(board.BlackWin)
		return
	} else if sub.session.players[1] == sub {
		session.handleWin(board.WhiteWin)
		return
	}

	// TODO concurrent map writes probably because this is being called twice?
	session.subscriberLock.Lock()
	session.viewers.Remove(sub)
	session.subscriberLock.Unlock()
}

func (session *Session) publishImpl(event Event, sub *subscriber) {
	if sub == nil || sub.events == nil {
		return
	}
	// if buffer is full the subscriber is closed
	select {
	case sub.events <- event:
	default:
		sub.closeSlow()
	}
}

func (session *Session) publish(sub *subscriber, event Event) {
	count := 0
	for _, player := range session.players {
		if player == sub {
			continue
		}
		count += 1
		session.publishImpl(event, player)
	}
	for viewer := range session.viewers.Iter() {
		if viewer == sub {
			continue
		}
		count += 1
		session.publishImpl(event, viewer)
	}

	slog.Info("subscribers were sent an event",
		slog.Int("count", count), slog.Any("event", event))
}

func (session *Session) handleError(err error) {
	text := err.Error()
	session.publish(nil, Event{Type: errorEvent, Text: &text})
	for _, player := range session.players {
		player.closeNow(nil, err)
	}
	for viewer := range session.viewers.Iter() {
		viewer.closeNow(nil, err)
	}
}

func (session *Session) handleMove(sub *subscriber, move board.Move) error {
	session.boardStateLock.Lock()
	defer session.boardStateLock.Unlock()

	moving := session.boardState.WhoseMove()

	whiteTime := session.whiteTime
	blackTime := session.blackTime

	// clock only starts after both players have made their first move
	startClock := session.boardState.MoveCounter > 1

	if startClock {
		session.clockLock.Lock()
		defer session.clockLock.Unlock()
		session.stopClockImpl()
		session.updateClockImpl()

		// shouldn't really happen but w/evs
		whiteTime, blackTime = session.getClockStateImpl()
		if moving == board.White && whiteTime <= 0 {
			session.handleTimeLossImpl(board.White)
		} else if moving == board.Black && blackTime <= 0 {
			session.handleTimeLossImpl(board.Black)
		}
	} else {

	}

	err := session.boardState.MakeMove(move)
	if err != nil {
		session.handleError(err)
		return err
	}

	serialisedLegalMoves := board.SerialiseMoveList(session.boardState.LegalMoves)
	moveStr := move.Serialise()
	fen := session.boardState.Fen()
	whiteTimeMs := int32(whiteTime.Milliseconds())
	blackTimeMs := int32(blackTime.Milliseconds())
	event := moveEvent(&moveStr, &fen, &serialisedLegalMoves,
		&whiteTimeMs, &blackTimeMs)
	session.publish(sub, event)

	if session.boardState.WinState > board.NoWin {
		err = errors.New("move sent after game end")
		session.handleError(err)
		return err
	}

	win := session.boardState.HasWinner()
	if win > board.NoWin {
		session.handleWinImpl(win)
		return nil
	}

	if startClock {
		session.startClockImpl(board.OppositeColour(moving))
	}
	return nil
}

func moveEvent(moveStr, fen *string, serialisedLegalMoves *[]string, whiteTimeMs, blackTimeMs *int32) Event {
	return Event{
		Type:       "move",
		Move:       moveStr,
		Fen:        fen,
		LegalMoves: serialisedLegalMoves,
		WhiteTime:  whiteTimeMs,
		BlackTime:  blackTimeMs,
	}
}

func (session *Session) handleWin(win board.WinState) {
	session.boardStateLock.Lock()
	session.handleWinImpl(win)
	session.boardStateLock.Unlock()
}
func (session *Session) handleWinImpl(win board.WinState) {
	slog.Info("win",
		slog.String("condition", board.WinStateToString(win)),
		slog.String("sessionId", session.id.String()))

	session.stopClock()

	var outcome string
	var victor string
	switch win {
	case board.BlackWin:
		outcome = "win"
		victor = "b"
	case board.WhiteWin:
		outcome = "win"
		victor = "w"
	case board.Stalemate:
		outcome = "stalemate"
		victor = "w"
	case board.MoveRuleDraw:
		outcome = "stalemate"
		victor = "w"
	case board.NoWin:
		fallthrough
	default:
	}
	session.publish(nil, Event{Type: "end", Outcome: &outcome, Victor: &victor})

	go func() {
		time.Sleep(5 * time.Second)
		session.cleanup()
	}()
}

func writeTimeout(ctx context.Context, timeout time.Duration, wsConn *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wsConn.Write(ctx, websocket.MessageText, msg)
}

func (sub *subscriber) closeNow(ctx context.Context, err error) {
	if sub.state == Closed {
		return
	}
	sub.state = Closed

	if sub.doneChannel != nil {
		sub.doneChannel <- struct{}{}
	}

	slog.Info("closing")
	if err != nil {
		logError(ctx, err)
	}
	sub.Conn.CloseNow()
	sub.session.DeleteSubscriber(sub)
}

func (sub *subscriber) closeSlow() {
	if sub.doneChannel != nil {
		sub.doneChannel <- struct{}{}
	}
	sub.state = Closed

	slog.Info("closing")
	if sub.Conn != nil {
		err := sub.Conn.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
		if err != nil {
			err = sub.Conn.CloseNow()
			panic(err)
		}
	}
	sub.session.DeleteSubscriber(sub)
}

var buffer = [1000]byte{}

func (sub *subscriber) initRead(ctx context.Context) {
	for {
		msgType, reader, err := sub.Conn.Reader(ctx)
		if err != nil {
			closeStatus := websocket.CloseStatus(err)
			slog.InfoContext(ctx, "close", slog.String("code", closeStatus.String()))

			if closeStatus == websocket.StatusGoingAway {
				sub.Disconnected(ctx, err)
				return
			}

			sub.closeNow(ctx, err)
			return
		}

		if msgType != websocket.MessageText {
			return
		}

		n, err := reader.Read(buffer[:])
		if err != nil {
			sub.closeNow(ctx, err)
			return
		}

		eventBuffer := Event{}
		err = json.Unmarshal(buffer[:n], &eventBuffer)
		if err != nil {
			sub.closeNow(ctx, err)
			return
		}
		if eventBuffer.Type != "sendMove" {
			sub.closeNow(ctx, errors.New("event sent is not \"sendMove\""))
			return
		}

		if sub.colour != board.White && sub.colour != board.Black {
			sub.closeNow(ctx, errors.New("invalid colour"))
			return
		}

		if sub.colour != sub.session.boardState.WhoseMove() {
			sub.closeNow(ctx, errors.New("not player to move"))
			colour := board.OppositeColour(sub.colour)
			sub.session.handleWin(board.ColourToWinState(colour))
			return
		}

		move, err := board.DeserialiseMove(*eventBuffer.Move)
		if err != nil {
			sub.closeNow(ctx, err)
			return
		}
		fmt.Printf("%+v\n", move)

		_ = sub.session.handleMove(sub, move)
	}
}

const (
	pongWait     = 5 * time.Second
	pingInterval = (pongWait * 9) / 10
)

func (sub *subscriber) write(ctx context.Context, event Event) error {
	resp, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = writeTimeout(ctx, time.Second*5, sub.Conn, resp)
	if err != nil {
		return err
	}

	return nil
}

func (sub *subscriber) initWrite(ctx context.Context) {
	pinger := time.NewTicker(pingInterval)
	defer pinger.Stop()

	for {
		select {
		case <-sub.doneChannel:
			return
		case event := <-sub.events:
			err := sub.write(ctx, event)

			if err != nil {
				sub.closeNow(ctx, err)
				return
			}
		case <-pinger.C:
			slog.InfoContext(ctx, "pinging")
			ctx, cancel := context.WithTimeout(ctx, pongWait)
			defer cancel()

			err := sub.Conn.Ping(ctx)

			if err != nil {
				slog.Info("ping failed",
					slog.String("userId", sub.userId.String()),
					slog.String("gameid", sub.session.id.String()))
				sub.Disconnected(ctx, err)
				return
			}

			slog.Info("ping succeeded",
				slog.String("userId", sub.userId.String()),
				slog.String("gameid", sub.session.id.String()))
		case <-ctx.Done():
			sub.closeNow(ctx, nil)
			return
		}
	}
}

func (sub *subscriber) Disconnected(ctx context.Context, err error) {
	if sub.state == Disconnected {
		return
	}
	sub.state = Disconnected

	colour := serialiseColour(sub.colour)
	sub.session.publish(nil, Event{
		Type:   disconnect,
		Colour: &colour,
	})

	duration := sub.session.gameLength / 10
	timer := time.NewTimer(duration)
	defer timer.Stop()

	slog.Info("user disconnected",
		slog.String("userId", sub.userId.String()),
		slog.String("gameId", sub.session.id.String()),
		slog.String("waiting", duration.String()),
	)

	select {
	case <-timer.C:
		slog.Info("game ended due to timeout",
			slog.String("userId", sub.userId.String()),
			slog.String("gameId", sub.session.id.String()),
		)

		colour := board.OppositeColour(sub.colour)
		sub.session.handleWin(board.ColourToWinState(colour))

		sub.closeNow(ctx, err)
	case <-ctx.Done():
		sub.closeNow(ctx, ctx.Err())
	case <-sub.reconnectChannel:
		return
	}
}

func getId(writer http.ResponseWriter, req *http.Request) (uuid.UUID, error) {
	id := strings.TrimPrefix(req.URL.Path, "/subscribe/")
	if id == "" {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return uuid.UUID{}, errors.New("no campaign id in request")
	}

	return uuid.Parse(id)
}

func dumpMap(space string, m map[string]interface{}) {
	for k, v := range m {
		if mv, ok := v.(map[string]interface{}); ok {
			fmt.Printf("{ \"%v\": \n", k)
			dumpMap(space+"\t", mv)
			fmt.Printf("}\n")
		} else {
			fmt.Printf("%v %v : %v\n", space, k, v)
		}
	}
}

func (session *Session) startClock(colour board.Colour) {
	session.clockLock.Lock()
	session.startClockImpl(colour)
	session.clockLock.Unlock()
}
func (session *Session) startClockImpl(colour board.Colour) {
	if session.clockTimer != nil {
		session.clockTimer.Stop()
	}

	var remainingTime time.Duration
	if session.boardState.WhoseMove() == board.White {
		remainingTime = session.whiteTime
	} else {
		remainingTime = session.blackTime
	}

	session.clockTimer = time.AfterFunc(remainingTime, func() {
		session.handleTimeLoss(colour)
	})
}

func (session *Session) startAbortClockImpl(colour board.Colour) {
	if session.clockTimer != nil {
		session.clockTimer.Stop()
	}

	abortTimer := session.whiteTime / 10

	session.clockTimer = time.AfterFunc(abortTimer, func() {
		session.handleAbort(colour)
	})
}

func (session *Session) stopClock() {
	session.clockLock.Lock()
	session.stopClockImpl()
	session.clockLock.Unlock()
}
func (session *Session) stopClockImpl() {
	if session.clockTimer != nil {
		session.clockTimer.Stop()
		session.clockTimer = nil
	}
}

func (session *Session) updateClock() {
	session.clockLock.Lock()
	session.updateClockImpl()
	session.clockLock.Unlock()
}
func (session *Session) updateClockImpl() {
	now := time.Now()
	elapsed := now.Sub(session.updatedAt)

	if session.boardState.WhoseMove() == board.Black {
		session.whiteTime = session.whiteTime - elapsed + session.increment
		if session.whiteTime < 0 {
			session.whiteTime = 0
		}
	} else {
		session.blackTime = session.blackTime - elapsed + session.increment
		if session.blackTime < 0 {
			session.blackTime = 0
		}
	}

	session.updatedAt = now
}

func (session *Session) handleTimeLoss(losingColour board.Colour) {
	session.clockLock.Lock()
	session.handleTimeLossImpl(losingColour)
	session.clockLock.Unlock()
}
func (session *Session) handleTimeLossImpl(losingColour board.Colour) {
	winningColour := board.OppositeColour(losingColour)
	winState := board.ColourToWinState(winningColour)

	var outcome string
	var victor string
	switch winState {
	case board.BlackWin:
		outcome = "win"
		victor = "b"
	case board.WhiteWin:
		outcome = "win"
		victor = "w"
	}

	session.publish(nil, Event{Type: "end", Outcome: &outcome, Victor: &victor})

	go func() {
		time.Sleep(5 * time.Second)
		session.cleanup()
		session.server.RemoveSession(session.id)
	}()
}

func (session *Session) handleAbort(colour board.Colour) {
	session.clockLock.Lock()
	session.handleTimeLossImpl(colour)
	session.clockLock.Unlock()
}
func (session *Session) handleAbortImpl(colour board.Colour) {
	colourStr := serialiseColour(colour)
	session.publish(nil, Event{Type: abort, Colour: &colourStr})

	go func() {
		time.Sleep(5 * time.Second)
		session.cleanup()
		session.server.RemoveSession(session.id)
	}()
}

func (session *Session) getClockState() (whiteTime, blackTime time.Duration) {
	session.clockLock.Lock()
	defer session.clockLock.Unlock()
	return session.getClockStateImpl()
}
func (session *Session) getClockStateImpl() (whiteTime, blackTime time.Duration) {
	now := time.Now()
	elapsed := now.Sub(session.updatedAt)

	whiteTime = session.whiteTime
	blackTime = session.blackTime

	if session.boardState.WhoseMove() == board.White {
		whiteTime -= elapsed
		if whiteTime < 0 {
			whiteTime = 0
		}
	} else {
		blackTime -= elapsed
		if blackTime < 0 {
			blackTime = 0
		}
	}

	return whiteTime, blackTime
}

func (session *Session) cleanup() {
	session.server.RemoveSession(session.id)

	session.clockLock.Lock()
	defer session.clockLock.Unlock()

	if session.clockTimer != nil {
		session.clockTimer.Stop()
		session.clockTimer = nil
	}
}

func (server *GameServer) RemoveSession(sessionId uuid.UUID) {
	server.sessionsLock.Lock()
	defer server.sessionsLock.Unlock()

	if session, exists := server.sessions[sessionId]; exists {
		session.cleanup()
		delete(server.sessions, sessionId)
	}
}
