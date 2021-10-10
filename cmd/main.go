package main

import (
	"fmt"
	api_server "github.com/Sergei39/tp-proxy-server/internal/api"
	db_proxy "github.com/Sergei39/tp-proxy-server/internal/db-proxy"
	proxy_server "github.com/Sergei39/tp-proxy-server/internal/proxy-server"
	"github.com/jackc/pgx"
	"log"
	"net/http"
)

func main() {
	connectionString := "postgres://seshishkin:postgres@localhost/proxy?sslmode=disable"

	configDB, err := pgx.ParseURI(connectionString)
	if err != nil {
		fmt.Println(err)
		return
	}

	db, err := pgx.NewConnPool(
		pgx.ConnPoolConfig{
			ConnConfig:     configDB,
			MaxConnections: 16,
			AfterConnect:   nil,
			AcquireTimeout: 0,
		})

	if err != nil {
		fmt.Println(err)
		return
	}

	rp := db_proxy.NewRepeater(db)
	proxy := proxy_server.NewProxyServer(rp)

	serverProxy := &http.Server{
		Addr:    ":" + "8080",
		Handler: proxy.SaveMiddleware(proxy.Handler),
	}
	go func() {
		log.Fatal(serverProxy.ListenAndServe())
	}()

	scanner := api_server.NewScanner(proxy)
	apiServer := api_server.NewApi(db, proxy, scanner)
	router := apiServer.NewRouter()

	serverApi := &http.Server{
		Addr: ":8000",
		Handler: router,
	}
	log.Fatal(serverApi.ListenAndServe())
}
