package matchmaking_server

import (
	"context"
	"encoding/json"
	"errors"
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
	"golang.org/x/exp/slices"
)

type Format struct {
	Increment  time.Duration
	GameLength time.Duration
}

type Queue struct {
	lock  sync.Mutex
	queue []*Player
}

func newQueue() *Queue {
	return &Queue{
		lock:  sync.Mutex{},
		queue: make([]*Player, 0),
	}
}

func (queue *Queue) push(player *Player) {
	queue.queue = append(queue.queue, player)
}

func (queue *Queue) pop() *Player {
	player := queue.queue[0]
	queue.queue = queue.queue[1:]
	return player
}

func (queue *Queue) removePlayer(player *Player) error {
	index := slices.Index(queue.queue, player)
	if index == -1 {
		return errors.New("player was not found in queue")
	}
	queue.queue = slices.Delete(queue.queue, index, index)
	return nil
}

type QueueMap map[Format]*Queue
type MatchmakingServer struct {
	ServeMux   *http.ServeMux
	gameServer *game_server.GameServer
	queueLock  sync.Mutex
	queues     QueueMap
	db         *model.Queries
	authServer *auth.AuthServer
}

type Player struct {
	id          uuid.UUID
	elo         int
	params      string
	Conn        *websocket.Conn
	closed      bool
	doneChannel chan struct{}
	queue       *Queue
}

func newPlayer(
	conn *websocket.Conn, queue *Queue, userId uuid.UUID,
) *Player {
	return &Player{
		id:          userId,
		elo:         0,
		params:      "",
		Conn:        conn,
		closed:      false,
		doneChannel: make(chan struct{}),
		queue:       queue,
	}
}

func NewMatchmakingServer(
	gameServer *game_server.GameServer,
	db *model.Queries,
	authServer *auth.AuthServer,
) *MatchmakingServer {
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

func (player *Player) write(ctx context.Context, bytes []byte) error {
	println("write")
	defer println("write done")
	return writeTimeout(ctx, 3*time.Second,
		player.Conn, bytes)
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
	}, nil
}

func (server *MatchmakingServer) getQueue(format *Format) *Queue {
	server.queueLock.Lock()
	queue, found := server.queues[*format]
	if !found {
		queue = newQueue()
		server.queues[*format] = queue
	}
	server.queueLock.Unlock()

	return queue
}

func found(gameId string) []byte {
	bytes, err := json.Marshal(QueueResponse{true, gameId})
	if err != nil {
		panic(err)
	}
	return bytes
}

func (server *MatchmakingServer) UnrankedHandler(
	writer http.ResponseWriter, req *http.Request,
) {
	slog.Info("UnrankedHandler")

	format, err := getFormat(req)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := req.Context()
	userSession, err := server.authServer.GetUserSession(ctx, writer, req)
	if err != nil {
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

		slog.Info("no match found",
			slog.String("http player", userSession.UserID.String()))

		writer.Header().Add("Content-Type", "application/json")
		writer.Write(bytes)
		return
	}

	player := queue.pop()
	queue.lock.Unlock()

	gameId := server.gameServer.NewSession(
		player.id,
		userSession.UserID,
		format.Increment,
		format.GameLength,
	)

	bytes := found(gameId.String())

	slog.Info("match found",
		slog.String("queue player", player.id.String()),
		slog.String("http player", userSession.UserID.String()))

	err = player.write(ctx, bytes)
	player.closeNow(ctx, err)

	println("returning")
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte("{\"found\":false}"))
	} else {
		writer.Header().Add("Content-Type", "application/json")
		writer.Write(bytes)
	}
}

func (server *MatchmakingServer) UnrankedQueueHandler(writer http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	session, err := server.authServer.GetUserSession(ctx, writer, req)
	if err != nil {
		return
	}

	err = server.Subscribe(ctx, writer, req, session.UserID)
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
func (server *MatchmakingServer) Subscribe(
	ctx context.Context,
	writer http.ResponseWriter,
	req *http.Request,
	userId uuid.UUID,
) error {
	format, err := getFormat(req)
	if err != nil {
		return err
	}

	// todo accept header
	conn, err := websocket.Accept(writer, req,
		&websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		return err
	}
	// todo make session id and add to context
	slog.InfoContext(ctx, "client subscribed to queue", slog.Any("format", format))

	// todo not sure why having this causes connection to be closed
	// ctx = conn.CloseRead(ctx)

	queue := server.getQueue(&format)
	queue.lock.Lock()
	// todo handle player joining queue more than once?
	player := newPlayer(conn, queue, userId)
	queue.push(player)
	queue.lock.Unlock()

	ctx = context.WithoutCancel(ctx)
	go player.initWrite(ctx)

	return nil
}

func writeTimeout(
	ctx context.Context, timeout time.Duration,
	wsConn *websocket.Conn, msg []byte,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wsConn.Write(ctx, websocket.MessageText, msg)
}

func (player *Player) closeNow(ctx context.Context, err error) {
	println("closeNow")
	defer println("closeNow done")
	if player.doneChannel != nil {
		player.doneChannel <- struct{}{}
	}

	slog.Info("closing player ws", slog.String("id", player.id.String()))
	if err != nil {
		logError(ctx, err)
	}

	player.Conn.CloseNow()
	player.queue.lock.Lock()
	err = player.queue.removePlayer(player)
	if err != nil {
		slog.Error("removing_player", slog.Any("error", err))
		return
	}
	player.queue.lock.Unlock()
}

const (
	pongWait     = 5 * time.Second
	pingInterval = (pongWait * 9) / 10
)

func (player *Player) initWrite(ctx context.Context) {
	pinger := time.NewTicker(pingInterval)
	defer pinger.Stop()

	for {
		select {
		case <-player.doneChannel:
			return
		case <-pinger.C:
			slog.DebugContext(ctx, "pinging")

			ctx, cancel := context.WithTimeout(ctx, pongWait)
			defer cancel()
			err := player.Conn.Ping(ctx)
			if err != nil {
				player.closeNow(ctx, nil)
				return
			}
		case <-ctx.Done():
			player.closeNow(ctx, nil)
			return
		}
	}
}

// func dumpMap(space string, m map[string]any) {
// 	for k, v := range m {
// 		if mv, ok := v.(map[string]any); ok {
// 			fmt.Printf("{ \"%v\": \n", k)
// 			dumpMap(space+"\t", mv)
// 			fmt.Printf("}\n")
// 		} else {
// 			fmt.Printf("%v %v : %v\n", space, k, v)
// 		}
// 	}
// }
