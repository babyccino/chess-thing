package matchmaking_server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"chess/auth"
	"chess/game_server"
	"chess/model"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type Format struct {
	Increment  time.Duration
	GameLength time.Duration
}

type Queue struct {
	lock  sync.Mutex
	queue []*player
}

func newQueue() *Queue {
	return &Queue{
		lock:  sync.Mutex{},
		queue: make([]*player, 0),
	}
}

func (queue *Queue) popQueue() *player {
	player := queue.queue[0]
	queue.queue = queue.queue[1:]
	return player
}

type QueueMap map[Format]*Queue
type MatchmakingServer struct {
	ServeMux   *http.ServeMux
	gameServer *game_server.GameServer
	queues     QueueMap
	db         *model.Queries
	authServer *auth.AuthServer
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

func NewMatchmakingServer(gameServer *game_server.GameServer, db *model.Queries, authServer *auth.AuthServer) *MatchmakingServer {
	serveMux := http.NewServeMux()
	server := &MatchmakingServer{
		ServeMux:   serveMux,
		queues:     make(QueueMap),
		gameServer: gameServer,
		db:         db,
		authServer: authServer,
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

func (player *player) write(ctx context.Context, bytes []byte) error {
	return player.Conn.Write(ctx,
		websocket.MessageText,
		bytes)
}

const formatQueryKey = "format"

func getFormat(req *http.Request) (Format, error) {
	format := req.URL.Query().Get(formatQueryKey)
	if format == "" {
		return Format{}, errors.New("no format found")
	}
	if format == "custom" {
		return Format{}, nil
	}
	before, after, found := strings.Cut(format, "+")
	if !found {
		return Format{}, errors.New("format in wrong format")
	}
	beforeNum, err := strconv.ParseInt(before, 10, 64) // 10 is base 10, 64 is bit size (int64)
	if err != nil {
		return Format{}, err
	}
	afterNum, err := strconv.ParseInt(after, 10, 64) // 10 is base 10, 64 is bit size (int64)
	if err != nil {
		return Format{}, err
	}

	return Format{
			GameLength: time.Minute * time.Duration(beforeNum),
			Increment:  time.Minute * time.Duration(afterNum),
		},
		nil
}

func (server *MatchmakingServer) getQueue(format *Format) *Queue {
	queue, found := server.queues[*format]
	if !found {
		queue = newQueue()
		server.queues[*format] = queue
	}
	return queue
}
func (server *MatchmakingServer) UnrankedHandler(writer http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	format, err := getFormat(req)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	if !server.authServer.IsAuthenticated(ctx, writer, req) {
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	queue := server.getQueue(&format)
	queue.lock.Lock()

	if len(queue.queue) == 0 {
		queue.lock.Unlock()

		bytes, err := json.Marshal(QueueResponse{Found: false})
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		writer.Header().Add("Content-Type", "application/json")
		writer.Write(bytes)
		return
	}

	player := queue.popQueue()
	queue.lock.Unlock()

	gameId := server.gameServer.NewSession(format.Increment, format.GameLength)

	bytes, err := json.Marshal(QueueResponse{true, gameId.String()})
	if err != nil {
		player.write(ctx, []byte("{\"found\":false,\"error\":\"ERROR\"}"))
		player.closeNow(ctx, nil)

		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte("{\"found\":false}"))

		panic("error marshalling json")
	}

	writer.Header().Add("Content-Type", "application/json")
	writer.Write(bytes)

	player.write(ctx, bytes)
	player.closeNow(ctx, nil)
}

func (server *MatchmakingServer) UnrankedQueueHandler(writer http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	session, err := server.authServer.GetUserSession(ctx, writer, req)
	if err != nil {
		return
	}

	err = server.Subscribe(ctx, writer, req, session.SessionUserID)
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
	// TODO
	return nil
}

// subscribeHandler accepts the WebSocket connection and then subscribes
// it to all future messages.
func (server *MatchmakingServer) Subscribe(ctx context.Context,
	writer http.ResponseWriter, req *http.Request, userId string) error {
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

	format, err := getFormat(req)
	if err != nil {
		return err
	}

	queue := server.getQueue(&format)
	queue.lock.Lock()
	queue.queue = append(queue.queue, player)
	queue.lock.Unlock()

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
