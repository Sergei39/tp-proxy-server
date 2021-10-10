package db_proxy

import (
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx"
	"net/http"
	"strings"
)

type Repeater interface {
	SaveRequest(r http.Request, schema string)
}

type repeater struct {
	DB *pgx.ConnPool
}

func NewRepeater(DB *pgx.ConnPool) Repeater {
	return repeater{
		DB: DB,
	}
}

func (h repeater) SaveRequest(r http.Request, schema string) {
	query := `INSERT INTO requests (host, path, method, headers, params, cookies, body, schema) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	pathParams := strings.Split(r.RequestURI, "?")
	path, getParams := pathParams[0], ""
	if len(pathParams) > 1 {
		getParams = pathParams[1]
	}

	body := ""
	if _, ok := r.Header["Content-Length"]; ok {
		fmt.Println("Save body")
		var b []byte
		if _, err := r.Body.Read(b); err != nil {
			fmt.Printf("read body error: %s\n", err)
		}
		body = string(b)
	}

	_, err := h.DB.Exec(query, r.Host, path, r.Method, parseHeaders(r), getParams, parseCookies(r), body, schema)
	if err != nil {
		fmt.Printf("don't save request: %s\n", err)
	}
}

func parseHeaders(r http.Request) []byte {
	result, err := json.Marshal(r.Header)
	if err != nil {
		fmt.Printf("error Marshal: %s", err)
	}

	return result
}

func parseCookies(r http.Request) []byte {
	result, err := json.Marshal(r.Cookies())
	if err != nil {
		fmt.Printf("error Marshal: %s", err)
	}

	return result
}
