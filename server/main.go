package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
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

const (
	CorsHeader = "Access-Control-Allow-Origin"
	AllowAll   = "*"
)

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

	// todo prod
	redirectPath := "http://localhost:3000/api"
	authServer := auth.NewAuthServer(queries, environment, redirectPath)
	gameServer := game_server.NewGameServer(authServer)
	matchmakingServer := matchmaking_server.NewMatchmakingServer(gameServer,
		queries, authServer)

	mux := http.NewServeMux()

	/**
	in dev this will be the main server and astro will be used for convenience
	in prod we're gonna statically generate html, serve that with nginx,
			and reverse proxy api requests to the go server

	prod: req -> nginx ->(api) go server
	                   ->(page) static html

	dev: req -> go server ->(api) go server
	                      ->(page) astro server

	dno why I did this this way tbh I could have just used the astro server as the proxy
	to the api server
	**/

	prefix := ""
	if environment.AppEnv == env.Dev {
		prefix = "/api"

		originServerURL, err := url.Parse("http://localhost:4321")
		if err != nil {
			log.Fatal("invalid origin server URL")
		}

		proxy := httputil.NewSingleHostReverseProxy(originServerURL)

		proxy.Director = func(req *http.Request) {
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
			req.URL.Host = originServerURL.Host
			req.URL.Scheme = originServerURL.Scheme
		}

		mux.HandleFunc("/", func(writer http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/api") {
				return
			}
			proxy.ServeHTTP(writer, req)
		})
	}

	gamePath := prefix + "/game"
	matchPath := prefix + "/matchmaking"
	authPath := prefix + "/auth"

	mux.Handle(gamePath+"/",
		http.StripPrefix(gamePath, gameServer))
	mux.Handle(matchPath+"/",
		http.StripPrefix(matchPath, matchmakingServer))
	mux.Handle(authPath+"/",
		http.StripPrefix(authPath, authServer))

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
