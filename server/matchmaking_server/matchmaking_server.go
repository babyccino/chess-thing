package matchmaking_server

import (
	"chess/auth"
	"chess/game_server"
	"chess/model"
	"context"
	"database/sql"
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

type MatchmakingServer struct {
	ServeMux   *http.ServeMux
	gameServer *game_server.GameServer
	queueLock  sync.Mutex
	queue      []*player
	db         *model.Queries
}

type player struct {
	id          uuid.UUID
	elo         int
	params      string
	Conn        *websocket.Conn
	closed      bool
	doneChannel chan struct{}
	server      *MatchmakingServer
}

func newPlayer(conn *websocket.Conn, server *MatchmakingServer) *player {
	return &player{
		id:          uuid.New(),
		elo:         0,
		params:      "",
		Conn:        conn,
		closed:      false,
		doneChannel: make(chan struct{}),
		server:      server,
	}
}

func NewMatchmakingServer(gameServer *game_server.GameServer, db *model.Queries) *MatchmakingServer {
	serveMux := http.NewServeMux()
	server := &MatchmakingServer{
		ServeMux:   serveMux,
		queue:      make([]*player, 0),
		queueLock:  sync.Mutex{},
		gameServer: gameServer,
		db:         db,
	}

	serveMux.HandleFunc("/unranked", server.UnrankedHandler)
	serveMux.HandleFunc("/unranked/subscribe", server.UnrankedQueueHandler)

	return server
}

func (server *MatchmakingServer) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	server.ServeMux.ServeHTTP(writer, req)
}

func (server *MatchmakingServer) OnShutdown() {
	// TODO
}

func logError(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "error", slog.Any("error", err))
}

type QueueResponse struct {
	Found  bool   `json:"found"`
	GameId string `json:"gameId,omitempty"`
}

func (server *MatchmakingServer) UnrankedHandler(writer http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	sessionId, err := req.Cookie(auth.CookieKeySession)
	if err != nil {
		http.Error(writer, "State cookie not found", http.StatusBadRequest)
		return
	}

	_, err = server.db.GetSessionById(ctx, sessionId.Value)
	if err == sql.ErrNoRows {
		http.Error(writer, "No db session found", http.StatusUnauthorized)
	} else if err != nil {
		slog.Error(
			"error retrieving session",
			slog.Any("error", err),
		)
		http.Error(writer, "Failed querying db", http.StatusInternalServerError)
		return
	}

	server.queueLock.Lock()

	if len(server.queue) > 0 {
		player := server.queue[0]
		server.queue = server.queue[1:]
		server.queueLock.Unlock()

		gameId := server.gameServer.NewSession()

		bytes, err := json.Marshal(QueueResponse{true, gameId.String()})
		if err != nil {
			server.queueLock.Lock()
			server.queue = append(server.queue, player)
			server.queueLock.Unlock()
			writer.WriteHeader(500)
			writer.Write([]byte("{\"ok\":true}"))
			return
		}

		writer.Header().Add("Content-Type", "application/json")
		writer.Write(bytes)

		player.Conn.Write(ctx, websocket.MessageText, bytes)
		player.closeNow(ctx, nil)
		return
	}
	server.queueLock.Unlock()

	bytes, err := json.Marshal(QueueResponse{Found: false})
	if err != nil {
		writer.WriteHeader(500)
		return
	}

	writer.Header().Add("Content-Type", "application/json")
	writer.Write(bytes)
}

func (server *MatchmakingServer) UnrankedQueueHandler(writer http.ResponseWriter, req *http.Request) {
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

func (server *MatchmakingServer) MarkDelete(id uuid.UUID) error {
	//TODO
	return nil
}

// subscribeHandler accepts the WebSocket connection and then subscribes
// it to all future messages.
func (server *MatchmakingServer) Subscribe(ctx context.Context, writer http.ResponseWriter, req *http.Request) error {
	// todo accept header
	conn, err := websocket.Accept(writer, req, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		return err
	}

	// todo make session id and add to context
	slog.InfoContext(ctx, "client subscribed to unranked queue")

	// todo not sure why having this causes connection to be closed
	// ctx = conn.CloseRead(ctx)
	player := newPlayer(conn, server)

	server.queueLock.Lock()
	server.queue = append(server.queue, player)
	server.queueLock.Unlock()

	ctx = context.WithoutCancel(ctx)
	go player.initWrite(ctx)

	return nil
}

func writeTimeout(ctx context.Context, timeout time.Duration, wsConn *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wsConn.Write(ctx, websocket.MessageText, msg)
}

func (player *player) closeNow(ctx context.Context, err error) {
	if player.doneChannel != nil {
		player.doneChannel <- struct{}{}
	}

	slog.Info("closing")
	if err != nil {
		logError(ctx, err)
	}
	player.Conn.CloseNow()
	player.server.MarkDelete(player.id)
}

func (player *player) closeSlow() {
	player.closed = true
	if player.Conn != nil {
		player.Conn.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
	}
	player.server.MarkDelete(player.id)
}

const (
	pongWait     = 5 * time.Second
	pingInterval = (pongWait * 9) / 10
)

func (player *player) initWrite(ctx context.Context) {
	pinger := time.NewTicker(pingInterval)
	var err error
	defer pinger.Stop()
	defer player.closeNow(ctx, err)

	for {
		select {
		case <-player.doneChannel:
			return
		case <-pinger.C:
			slog.DebugContext(ctx, "pinging")
			ctx, cancel := context.WithTimeout(ctx, pongWait)
			defer cancel()
			err2 := player.Conn.Ping(ctx)
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
