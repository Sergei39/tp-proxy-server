package main

import (
	"fmt"
	"github.com/jackc/pgx"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func middlewareSaveDb(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Executing middlewareSaveDb")
		next.ServeHTTP(w, r)
		log.Println("Executing middlewareSaveDb again")
	})
}

func (h handlers) saveRequest(r http.Request) {
	query := `INSERT INTO requests (method, path) VALUES ($1, $2) returning id`

	pathParams := strings.Split(r.RequestURI, "?")
	var id int
	err := h.DB.QueryRow(query, r.Method, pathParams[0]).Scan(&id)
	if err != nil {
		fmt.Println("don't save request")
	}

	if len(pathParams) > 1{
		query = `INSERT INTO params (request, name, value, type) VALUES `
		var queryParams []interface{}

		getParams := strings.Split(pathParams[1], "&")
		for i, params := range getParams {
			nameVal := strings.Split(params, "=")
			if len(nameVal) != 2 {
				continue
			}
			query += fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4)
			if i < len(getParams) - 1 {
				query += ", "
			}
			queryParams = append(queryParams, id, nameVal[0], nameVal[1], 0)
		}

		h.DB.Exec(query, queryParams...)
		if err != nil {
			fmt.Println("don't save request")
		}
	}


	query = `INSERT INTO params (request, name, value, type) VALUES `
	var queryParams []interface{}

	getParams := strings.Split(pathParams[1], "&")
	for i, params := range getParams {
		nameVal := strings.Split(params, "=")
		if len(nameVal) != 2 {
			continue
		}
		query += fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4)
		if i < len(getParams) - 1 {
			query += ", "
		}
		queryParams = append(queryParams, id, nameVal[0], nameVal[1], 0)
	}

	h.DB.Exec(query, queryParams...)
	if err != nil {
		fmt.Println("don't save request")
	}
}

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
	h.saveRequest(*r)
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
	DB *pgx.ConnPool
}

func NewHandlers(DB *pgx.ConnPool) *handlers {
	return &handlers{
		DB: DB,
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

	handlers := NewHandlers(db)

	server := &http.Server{
		Addr: ":8088",
		Handler: middlewareSaveDb(handlers.proxy),
	}
	log.Fatal(server.ListenAndServe())
}
