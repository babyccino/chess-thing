package game_server

import (
	"chess/board"
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
	subscriberLock sync.Mutex
	players        [2]*subscriber
	viewers        map[*subscriber]struct{}

	game      *Game
	id        string
	updatedAt time.Time
	createdAt time.Time
}
type Game struct {
	board *board.BoardState
}

type subscriber struct {
	gameId      string
	events      chan string
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

	server.ServeMux.HandleFunc("/", func(writer http.ResponseWriter, req *http.Request) {
		writer.Write([]byte("Go to wss:*/subscribe/gameId to connect"))
	})
	server.ServeMux.HandleFunc("/subscribe/", server.SubscribeHandler)

	return server, nil
}

func newSession() *Session {
	return &Session{
		subscriberLock: sync.Mutex{},
		viewers:        make(map[*subscriber]struct{}),
		createdAt:      time.Now(),
		updatedAt:      time.Now(),
	}
}

func (server *GameServer) NewGame() uuid.UUID {
	server.sessionsLock.Lock()
	defer server.sessionsLock.Unlock()

	session := newSession()
	id := uuid.New()
	server.sessions[id] = session
	return id
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

func (server *GameServer) Subscribe(ctx context.Context, writer http.ResponseWriter, req *http.Request) error {
	id, err := getId(writer, req)
	if err != nil {
		return err
	}

	// todo accept header
	Conn, err := websocket.Accept(writer, req, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		return err
	}

	// todo make session id and add to context
	slog.InfoContext(ctx, "client subscribed to events from campaign", slog.String("id", id.String()))

	server.sessionsLock.Lock()

	session, found := server.sessions[id]
	if !found {
		session = newSession()
		server.sessions[id] = session
	}

	sub := &subscriber{
		events:      make(chan string, 10),
		doneChannel: make(chan struct{}, 1),
		Conn:        Conn,
		session:     session,
	}

	// just make any connection either one of the players
	// todo: obvs this is just for testing
	session.subscriberLock.Lock()
	if session.players[0] != nil {
		session.players[0] = sub
	} else if session.players[1] != nil {
		session.players[1] = sub
	} else {
		session.viewers[sub] = struct{}{}
	}
	session.subscriberLock.Unlock()

	server.sessionsLock.Unlock()

	ctx = context.WithoutCancel(ctx)
	go sub.initWrite(ctx)
	go sub.initRead(ctx)

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
	delete(session.viewers, sub)
}

func (session *Session) end() {

}

func (session *Session) publishImpl(str string, sub *subscriber) {
	if sub == nil || sub.events == nil {
		return
	}
	// if buffer is full the subscriber is closed
	count := 0
	select {
	case sub.events <- str:
		count++
	default:
		sub.closeSlow()
	}
}
func (session *Session) publish(sub *subscriber, str string) {
	count := 0
	for _, player := range session.players {
		if player == sub {
			continue
		}
		session.publishImpl(str, player)
	}
	for viewer := range session.viewers {
		if viewer == sub {
			continue
		}
		session.publishImpl(str, viewer)
	}
	fmt.Printf("[debug] %d subscribers were sent an event \n", count)
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

		buf := make([]byte, 15)
		n, err := reader.Read(buf)
		if err != nil {
			sub.closeNow(ctx, err)
			return
		}

		sub.session.publish(sub, string(buf[:n]))
	}
}

const (
	pongWait     = 5 * time.Second
	pingInterval = (pongWait * 9) / 10
)

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
			resp, err2 := json.Marshal(event)
			if err2 != nil {
				err = err2
				return
			}

			err2 = writeTimeout(ctx, time.Second*5, sub.Conn, resp)
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

	return uuid.FromBytes([]byte(id))
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
