package main

import (
	"context"
	"database/sql"
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

func getEnv() (dbUrl string, dbAuthToken string, err error) {
	dbUrl, dbUrlExists := os.LookupEnv("LIB_SQL_DB_URL")
	dbAuthToken, dbAuthTokenExists := os.LookupEnv("LIB_SQL_AUTH_TOKEN")
	if !dbUrlExists || !dbAuthTokenExists {
		err := godotenv.Load("./.env")
		if err != nil {
			log.Fatal("env variables not found and .env file not found")
		}

		dbUrl, dbUrlExists = os.LookupEnv("LIB_SQL_DB_URL")
		dbAuthToken, dbAuthTokenExists = os.LookupEnv("LIB_SQL_AUTH_TOKEN")
		if !dbUrlExists || !dbAuthTokenExists {
			return "", "", errors.New("env variables not found in .env file")
		}
	}

	return dbUrl, dbAuthToken, nil
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

const CorsHeader = "Access-Control-Allow-Origin"
const AllowAll = "*"

func (server *MiddlewareServer) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add(CorsHeader, AllowAll)
	server.ServeMux.ServeHTTP(writer, req)
}

// run initializes the chatServer and then
// starts a http.Server for the passed in address.
func run() error {
	dbUrl, dbAuthToken, err := getEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[fatal-error] %s", err)
		os.Exit(1)
	}

	db, err := sql.Open("libsql", fmt.Sprintf("%s?authToken=%s", dbUrl, dbAuthToken))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[fatal-error] failed to open db %s: %s", dbUrl, err)
		os.Exit(1)
	}

	gameServer := game_server.NewGameServer()
	matchmakingServer := matchmaking_server.NewMatchmakingServer(gameServer)
	authServer := auth.NewAuthServer(db)

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
