package main

import (
	"fmt"
	"github.com/Sergei39/tp-proxy-server/repeater"
	"github.com/jackc/pgx"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func (h handlers) proxyHTTPS(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Host: ", r.Host)
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func changeHTTPRequest(r *http.Request) {
	r.Header.Del("Proxy-Connection")
	host := strings.Split(r.Host, ":")[0]
	r.RequestURI = strings.Split(r.RequestURI, host)[1]
}

func (h handlers) proxyHTTP(w http.ResponseWriter, r *http.Request) {
	changeHTTPRequest(r)
	h.repeater.SaveRequest(*r)
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

type handlers struct {
	repeater repeater.Repeater
}

func NewHandlers(rp repeater.Repeater) *handlers {
	return &handlers{
		repeater: rp,
	}
}

func (h handlers) proxy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		fmt.Println("Connect https")
		h.proxyHTTPS(w, r)
	} else {
		h.proxyHTTP(w, r)
	}
}

func main() {
	connectionString := "postgres://" + "seshishkin" + ":" + "5432" +
		"@localhost/" + "postgres" + "?sslmode=disable"

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

	rp := repeater.NewRepeater(db)

	handlers := NewHandlers(rp)

	server := &http.Server{
		Addr: ":8088",
		Handler: http.HandlerFunc(handlers.proxy),
	}
	log.Fatal(server.ListenAndServe())
}
