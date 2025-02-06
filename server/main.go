package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"chess/auth"
	"chess/env"
	"chess/game_server"
	"chess/matchmaking_server"
	"chess/model"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	log.SetFlags(0)

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func getAddr() string {
	if len(os.Args) < 2 {
		return "localhost:3000"
	}

	return os.Args[1]
}

type MiddlewareServer struct {
	ServeMux *http.ServeMux
}

//go:embed schema.sql
var ddl string

func getDb(ctx context.Context, environment *env.Env) (*sql.DB, error) {
	if environment.AppEnv == env.Dev {
		db, err := sql.Open("sqlite3", ":memory:")

		if err != nil {
			return nil, err
		}

		db.ExecContext(ctx, ddl)
		slog.Info("connected to in memory db")
		return db, err
	} else {
		slog.Info("connected to libsql db")
		return sql.Open(
			"libsql",
			fmt.Sprintf("%s?authToken=%s", environment.DbUrl, environment.DbAuthToken),
		)
	}
}

const CorsHeader = "Access-Control-Allow-Origin"
const AllowAll = "*"

func (server *MiddlewareServer) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add(CorsHeader, AllowAll)
	server.ServeMux.ServeHTTP(writer, req)
}

// run initializes the chatServer and then
// starts a http.Server for the passed in address.
func run() error {
	ctx := context.Background()

	environment, err := env.GetEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[fatal-error] %s", err)
		os.Exit(1)
	}

	db, err := getDb(ctx, environment)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[fatal-error] failed to open db %s: %s", environment.DbUrl, err)
		os.Exit(1)
	}

	queries := model.New(db)

	gameServer := game_server.NewGameServer()
	matchmakingServer := matchmaking_server.NewMatchmakingServer(gameServer)
	authServer := auth.NewAuthServer(queries, environment)

	mux := http.NewServeMux()
	mux.Handle("/game/", http.StripPrefix("/game", gameServer))
	mux.Handle("/matchmaking/", http.StripPrefix("/matchmaking", matchmakingServer))
	mux.Handle("/auth/", http.StripPrefix("/auth", authServer))

	middlewareServer := MiddlewareServer{ServeMux: mux}

	addr := getAddr()
	httpServer := &http.Server{
		Handler:      &middlewareServer,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
		Addr:         addr,
	}

	httpServer.RegisterOnShutdown(gameServer.OnShutdown)
	httpServer.RegisterOnShutdown(matchmakingServer.OnShutdown)

	errc := make(chan error, 1)
	go func() {
		log.Printf("listening on http://%v", addr)
		errc <- httpServer.ListenAndServe()
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	return httpServer.Shutdown(ctx)
}
