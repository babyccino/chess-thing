package game_server

import (
	"chess/board"
	"chess/utility"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type GameServer struct {
	ServeMux     *http.ServeMux
	sessionsLock sync.Mutex
	sessions     map[uuid.UUID]*Session
}

type Session struct {
	id    uuid.UUID
	board *board.BoardState

	subscriberLock sync.Mutex
	players        [2]*subscriber
	viewers        utility.Set[*subscriber]

	updatedAt time.Time
	createdAt time.Time
}

type subscriber struct {
	gameId      string
	events      chan Event
	doneChannel chan struct{}
	Conn        *websocket.Conn
	closed      bool
	session     *Session
}

func NewGameServer() (*GameServer, error) {
	server := &GameServer{
		ServeMux:     http.NewServeMux(),
		sessions:     make(map[uuid.UUID]*Session),
		sessionsLock: sync.Mutex{},
	}

	server.ServeMux.HandleFunc("/subscribe/", server.SubscribeHandler)

	return server, nil
}

func newSession() *Session {
	board := board.NewBoard()
	err := board.Init()
	if err != nil {
		panic(err)
	}
	return &Session{
		id:             uuid.New(),
		board:          board,
		subscriberLock: sync.Mutex{},
		viewers:        utility.NewSet[*subscriber](),
		createdAt:      time.Now(),
		updatedAt:      time.Now(),
	}
}

func (server *GameServer) NewSession() uuid.UUID {
	server.sessionsLock.Lock()
	defer server.sessionsLock.Unlock()

	session := newSession()
	server.sessions[session.id] = session
	return session.id
}

func (server *GameServer) OnShutdown() {
	// TODO
}

func (server *GameServer) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	server.ServeMux.ServeHTTP(writer, req)
}

func logError(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "error", slog.Any("error", err))
}

// subscribeHandler accepts the WebSocket connection and then subscribes
// it to all future messages.
func (server *GameServer) SubscribeHandler(writer http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	err := server.Subscribe(ctx, writer, req)
	if err == nil {
		return
	}
	logError(ctx, err)
	if errors.Is(err, context.Canceled) {
		return
	}
	closeStatus := websocket.CloseStatus(err)
	if closeStatus == websocket.StatusNormalClosure ||
		closeStatus == websocket.StatusGoingAway {
		return
	}
}

type eventType = string

const (
	connect       eventType = "connect"
	connectViewer           = "connectViewer"
	move                    = "move"
	end                     = "end"
	errorEvent              = "error"
)

/* TS
type ConnectEvent = {
  type: "connect"
  fen: string
  moveHistory?: string[]
  colour: "w" | "b"
  legalMoves?: string[]
}
type ConnectViewerEvent = {
  type: "connectViewer"
  fen: string
  moveHistory?: string[]
}
type MoveEvent = {
  type: "move"
  move: string
  fen: string
  legalMoves?: string[]
}
type SendMoveEvent = {
  type: "sendMove"
  move: string
}
type EndEvent = {
  type: "end"
  victor: "w" | "b"
}
type ChatEvent = {
  type: "chat"
  text: string
}
type ErrorEvent = {
  type: "error"
  text: string
}
*/

type Event struct {
	Type        eventType `json:"type"`
	Fen         string    `json:"fen,omitempty"`
	MoveHistory []string  `json:"moveHistory,omitempty"`
	Colour      string    `json:"colour,omitempty"`
	Move        string    `json:"move,omitempty"`
	LegalMoves  []string  `json:"legalMoves,omitempty"`
	Outcome     string    `json:"outcome,omitempty"`
	Victor      string    `json:"victor,omitempty"`
	Text        string    `json:"text,omitempty"`
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

func (session *Session) CreateConnectEvent(colour board.Colour) Event {
	fen := session.board.Fen()
	println(fen)
	if colour == board.None {
		return Event{
			Type:        connectViewer,
			Fen:         fen,
			MoveHistory: moveList(session.board.MoveHistory),
		}
	} else {
		return Event{
			Type:        connect,
			Fen:         fen,
			MoveHistory: moveList(session.board.MoveHistory),
			Colour:      serialiseColour(colour),
			LegalMoves:  moveList(session.board.GetLegalMoves()),
		}
	}
}

func (server *GameServer) Subscribe(ctx context.Context, writer http.ResponseWriter, req *http.Request) error {
	id, err := getId(writer, req)
	if err != nil {
		return err
	}

	server.sessionsLock.Lock()
	session, found := server.sessions[id]
	server.sessionsLock.Unlock()

	if !found {
		// todo accept header
		writer.WriteHeader(404)
		writer.Write([]byte(""))
		return errors.New("not found")
	}

	// todo accept header
	Conn, err := websocket.Accept(writer, req, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		return err
	}

	sub := &subscriber{
		events:      make(chan Event, 10),
		doneChannel: make(chan struct{}),
		Conn:        Conn,
		session:     session,
	}

	// just make any connection either one of the players
	// todo: obvs this is just for testing
	session.subscriberLock.Lock()
	colour := board.None
	if session.players[0] == nil {
		colour = board.White
		session.players[0] = sub
		slog.InfoContext(ctx, "added client to session as white player", slog.String("id", id.String()))
	} else if session.players[1] == nil {
		colour = board.Black
		session.players[1] = sub
		slog.InfoContext(ctx, "added client to session as black player", slog.String("id", id.String()))
	} else {
		colour = board.None
		session.viewers.Add(sub)
		slog.InfoContext(ctx, "added client to session as viewer", slog.String("id", id.String()))
	}
	session.subscriberLock.Unlock()

	ctx = context.WithoutCancel(ctx)

	// todo send info about other player
	event := session.CreateConnectEvent(colour)
	err = sub.write(ctx, event)
	if err != nil {
		sub.closeNow(ctx, err)
		return err
	}

	if colour != board.None {
		go sub.initRead(ctx)
	}
	go sub.initWrite(ctx)

	return nil
}

func (session *Session) DeleteSubscriber(sub *subscriber) {
	if session.players[0] == sub || sub.session.players[1] == sub {
		session.end()
		return
	}

	// TODO concurrent map writes probably because this is being called twice?
	session.subscriberLock.Lock()
	defer session.subscriberLock.Unlock()
	session.viewers.Remove(sub)
}

func (session *Session) end() {

}

func (session *Session) publishImpl(event Event, sub *subscriber) {
	if sub == nil || sub.events == nil {
		return
	}
	// if buffer is full the subscriber is closed
	count := 0
	select {
	case sub.events <- event:
		count++
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
		session.publishImpl(event, player)
	}
	for viewer := range session.viewers.Iter() {
		if viewer == sub {
			continue
		}
		session.publishImpl(event, viewer)
	}
	slog.Debug("subscribers were sent an event",
		slog.Int("count", count))
}

func writeTimeout(ctx context.Context, timeout time.Duration, wsConn *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wsConn.Write(ctx, websocket.MessageText, msg)
}

func (sub *subscriber) closeNow(ctx context.Context, err error) {
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
	sub.closed = true
	if sub.Conn != nil {
		sub.Conn.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
	}
	sub.session.DeleteSubscriber(sub)
}

var buffer = [1000]byte{}

func (sub *subscriber) initRead(ctx context.Context) {
	for {
		msgType, reader, err := sub.Conn.Reader(ctx)
		if err != nil {
			closeStatus := websocket.CloseStatus(err)
			slog.InfoContext(ctx, "close ", slog.Int("code", int(closeStatus)))

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

		var eventBuffer = Event{}
		err = json.Unmarshal(buffer[:n], &eventBuffer)
		if err != nil {
			sub.closeNow(ctx, err)
			return
		}
		if eventBuffer.Type != "sendMove" {
			sub.closeNow(ctx, errors.New("event sent is not \"sendMove\""))
			return
		}

		move, err := board.DeserialiseMove(eventBuffer.Move)
		if err != nil {
			sub.closeNow(ctx, err)
			return
		}
		fmt.Printf("%+v\n", move)

		sub.session.handleMove(sub, move)
	}
}

func (session *Session) handleError(err error) {
	session.publish(nil, Event{Type: errorEvent, Text: err.Error()})
	for _, player := range session.players {
		player.closeNow(nil, err)
	}
	for viewer := range session.viewers.Iter() {
		viewer.closeNow(nil, err)
	}
}
func (session *Session) handleMove(sub *subscriber, move board.Move) {
	err := session.board.MakeMove(move)
	if err != nil {
		session.handleError(err)
	}

	serialisedLegalMoves := board.SerialiseMoveList(session.board.LegalMoves)
	event := Event{
		Type:       "move",
		Move:       move.Serialise(),
		Fen:        session.board.Fen(),
		LegalMoves: serialisedLegalMoves,
	}
	session.publish(sub, event)

	win := session.board.HasWinner()
	switch win {
	case board.BlackWin:
		session.publish(nil, Event{Type: "end", Outcome: "win", Victor: "b"})
	case board.WhiteWin:
		session.publish(nil, Event{Type: "end", Outcome: "win", Victor: "w"})
	case board.Stalemate:
		session.publish(nil, Event{Type: "end", Outcome: "stalemate", Victor: "w"})
	case board.MoveRuleDraw:
		session.publish(nil, Event{Type: "end", Outcome: "stalemate", Victor: "w"})
	case board.NoWin:
		fallthrough
	default:
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
	var err error
	defer pinger.Stop()
	defer sub.closeNow(ctx, err)

	for {
		select {
		case <-sub.doneChannel:
			return
		case event := <-sub.events:
			err2 := sub.write(ctx, event)
			if err2 != nil {
				err = err2
				return
			}
		case <-pinger.C:
			slog.DebugContext(ctx, "pinging")
			ctx, cancel := context.WithTimeout(ctx, pongWait)
			defer cancel()
			err2 := sub.Conn.Ping(ctx)
			if err2 != nil {
				err = err2
				return
			}
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
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
