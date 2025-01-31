package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"chess/auth"
	"chess/game_server"
	"chess/matchmaking_server"
	"chess/model"

	"github.com/joho/godotenv"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	log.SetFlags(0)

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

type AppEnv = uint8

const (
	dev AppEnv = iota
	prod
)

type Env struct {
	dbUrl       string
	dbAuthToken string
	appEnv      AppEnv
}

func getEnv() (env *Env, err error) {
	dbUrl, dbUrlExists := os.LookupEnv("LIB_SQL_DB_URL")
	dbAuthToken, dbAuthTokenExists := os.LookupEnv("LIB_SQL_AUTH_TOKEN")
	appEnvStr, appEnvExists := os.LookupEnv("APP_ENV")

	appEnv := dev
	if appEnvExists && appEnvStr == "prod" {
		appEnv = prod
	}

	if appEnv == prod && (!dbUrlExists || !dbAuthTokenExists) {
		err := godotenv.Load("./.env")
		if err != nil {
			log.Fatal("env variables not found and .env file not found")
		}

		dbUrl, dbUrlExists = os.LookupEnv("LIB_SQL_DB_URL")
		dbAuthToken, dbAuthTokenExists = os.LookupEnv("LIB_SQL_AUTH_TOKEN")
		if !dbUrlExists || !dbAuthTokenExists {
			return nil, errors.New("env variables not found in .env file")
		}
	}

	return &Env{dbUrl, dbAuthToken, appEnv}, nil
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

func getDb(ctx context.Context, env *Env) (*sql.DB, error) {
	if env.appEnv == dev {
		db, err := sql.Open("sqlite", ":memory:")

		if err != nil {
			return nil, err
		}

		db.ExecContext(ctx, ddl)
		return db, err
	} else {
		return sql.Open(
			"libsql",
			fmt.Sprintf("%s?authToken=%s", env.dbUrl, env.dbAuthToken),
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

	env, err := getEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[fatal-error] %s", err)
		os.Exit(1)
	}

	db, err := getDb(ctx, env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[fatal-error] failed to open db %s: %s", env.dbUrl, err)
		os.Exit(1)
	}

	queries := model.New(db)

	gameServer := game_server.NewGameServer()
	matchmakingServer := matchmaking_server.NewMatchmakingServer(gameServer)
	authServer := auth.NewAuthServer(queries)

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
