package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"chess/game_server"
	"chess/matchmaking_server"
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

const CorsHeader = "Access-Control-Allow-Origin"
const AllowAll = "*"

func (server *MiddlewareServer) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add(CorsHeader, AllowAll)
	server.ServeMux.ServeHTTP(writer, req)
}

// run initializes the chatServer and then
// starts a http.Server for the passed in address.
func run() error {

	gameServer, err := game_server.NewGameServer()
	if err != nil {
		return err
	}

	matchmakingServer, err := matchmaking_server.NewMatchmakingServer(gameServer)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/game/", http.StripPrefix("/game", gameServer))
	mux.Handle("/matchmaking/", http.StripPrefix("/matchmaking", matchmakingServer))

	middlewareServer := MiddlewareServer{ServeMux: mux}

	addr := getAddr()
	// handler :=
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
