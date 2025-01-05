package game_server

import (
	"chess/board"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type GameServer struct {
	serveMux     http.ServeMux
	sessionsLock sync.Mutex
	sessions     map[string]*Session
	snsArn       string
	maxEventAge  time.Duration
}

type Session struct {
	players [2]*subscriber
	viewers map[*subscriber]struct{}

	game      *Game
	id        string
	updatedAt time.Time
	createdAt time.Time
}
type Game struct {
	board *board.BoardState
}

type subscriber struct {
	gameId  string
	events  chan string
	Conn    *websocket.Conn
	closed  bool
	session *Session
}

func NewGameServer(maxEventAge time.Duration) (*GameServer, error) {
	server := &GameServer{
		sessions:    make(map[string]*Session),
		maxEventAge: maxEventAge,
	}
	server.serveMux.HandleFunc("/", func(writer http.ResponseWriter, req *http.Request) {
		writer.Write([]byte("Go to wss:*/subscribe/gameId to connect"))
	})
	server.serveMux.HandleFunc("/subscribe/", server.SubscribeHandler)
	server.serveMux.HandleFunc("/ping", server.PingHandler)

	return server, nil
}

func newSession() *Session {
	return &Session{
		viewers:   make(map[*subscriber]struct{}),
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

func (server *GameServer) OnShutdown() {
}

func (server *GameServer) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	server.serveMux.ServeHTTP(writer, req)
}

func (server *GameServer) PingHandler(writer http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	body := http.MaxBytesReader(writer, req.Body, 1024)
	msg, err := io.ReadAll(body)
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	println(string(msg))

	var parsedBody struct {
		Ping bool `json:"ping"`
	}
	err2 := json.Unmarshal(msg, &parsedBody)
	fmt.Println(parsedBody.Ping)
	if err2 != nil || !parsedBody.Ping {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	writer.WriteHeader(http.StatusAccepted)
	writer.Header().Add("Content-Type", "application/json")
	pong := struct {
		Pong bool `json:"pong"`
	}{Pong: true}
	response, err := json.Marshal(pong)
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	writer.Write(response)
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
	Conn, err := websocket.Accept(writer, req, nil)
	if err != nil {
		return err
	}

	// todo make session id and add to context
	slog.InfoContext(ctx, "client subscribed to events from campaign", slog.String("id", id))

	server.sessionsLock.Lock()

	session, found := server.sessions[id]
	if !found {
		session = newSession()
		server.sessions[id] = session
	}

	sub := &subscriber{
		events:  make(chan string, 10),
		Conn:    Conn,
		session: session,
	}

	// just make any connection either one of the players
	// todo: obvs this is just for testing
	if session.players[0] != nil {
		session.players[0] = sub
	} else if session.players[1] != nil {
		session.players[1] = sub
	} else {
		session.viewers[sub] = struct{}{}
	}
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
func (session *Session) publish(str string) {
	count := 0
	for _, sub := range session.players {
		session.publishImpl(str, sub)
	}
	for sub := range session.viewers {
		session.publishImpl(str, sub)
	}
	fmt.Printf("[debug] %d subscribers were sent an event \n", count)
}

func writeTimeout(ctx context.Context, timeout time.Duration, wsConn *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wsConn.Write(ctx, websocket.MessageText, msg)
}

func (sub *subscriber) closeNow(ctx context.Context, err error) {
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

		sub.session.publish(string(buf[:n]))
	}
}

const (
	pongWait     = 5 * time.Second
	pingInterval = (pongWait * 9) / 10
)

func (sub *subscriber) initWrite(ctx context.Context) {
	pinger := time.NewTicker(pingInterval)
	defer pinger.Stop()

	for {
		select {
		case event := <-sub.events:
			resp, err := json.Marshal(event)
			if err != nil {
				return
			}

			err = writeTimeout(ctx, time.Second*5, sub.Conn, resp)
			if err != nil {
				sub.closeNow(ctx, err)
				return
			}
		case <-pinger.C:
			slog.DebugContext(ctx, "pinging")
			ctx, cancel := context.WithTimeout(ctx, pongWait)

			err := sub.Conn.Ping(ctx)
			if err != nil {
				sub.closeNow(ctx, err)
			}

			cancel()
		case <-ctx.Done():
			err := ctx.Err()
			if err != nil {
				logError(ctx, err)
			} else {
				slog.DebugContext(ctx, "context done")
			}
			sub.closeNow(ctx, err)
			return
		}
	}
}

func getId(writer http.ResponseWriter, req *http.Request) (string, error) {
	id := strings.TrimPrefix(req.URL.Path, "/subscribe/")
	if id == "" {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return "", errors.New("no campaign id in request")
	}

	return id, nil
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
