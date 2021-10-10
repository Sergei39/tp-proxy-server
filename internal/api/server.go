package api

import (
	"encoding/json"
	"fmt"
	models "github.com/Sergei39/tp-proxy-server/internal/models"
	proxy_server "github.com/Sergei39/tp-proxy-server/internal/proxy-server"
	"github.com/jackc/pgx"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type ApiServer interface {
	NewRouter() *http.ServeMux
}

type api struct {
	DB *pgx.ConnPool
	ps proxy_server.Server
	sc Scanner
}

func NewApi(DB *pgx.ConnPool, ps proxy_server.Server, sc Scanner) ApiServer {
	return &api{
		DB: DB,
		ps: ps,
		sc: sc,
	}
}

func (repo api) NewRouter() *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/requests", repo.getRequests)
	router.HandleFunc("/requests/", repo.getRequestById)
	router.HandleFunc("/repeat/", repo.repeatById)
	router.HandleFunc("/scan/", repo.scanById)

	return router
}

func (repo api) repeatById(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/repeat/"):]
	req, err := repo.getRequest(r, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	path := req.Path
	if req.Params != "" {
		path += "?" + req.Params
	}
	repo.ps.Handler(w, &http.Request{
		Method: req.Method,
		URL: &url.URL{
			Scheme: req.Schema,
			Host:   req.Host,
			Path:   path,
		},
		Header: req.Headers,
		Body:   ioutil.NopCloser(strings.NewReader(req.Body)),
		Host:   req.Host,
	})
}

func (repo api) scanById(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/scan/"):]
	req, err := repo.getRequest(r, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	answer, err := repo.sc.StartScan(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	result := strings.Join(answer, ", ")
	if len(answer) == 0 {
		result = "Not found"
	}
	_, err = io.Copy(w, strings.NewReader(result))
	if err != nil {
		fmt.Printf("write error: %s", err)
	}
}

func (repo api) getRequest(r *http.Request, id string) (*models.Request, error) {
	query := `
		select id, method, path, host, headers, params, body, cookies, schema from requests
		where id = $1
	`

	rows, err := repo.DB.Query(query, id)
	if err != nil {
		fmt.Printf("don't get requests: %s\n", err)
		return nil, err
	}

	req := new(models.Request)
	for rows.Next() {
		var headersJSON, cookiesJSON []byte
		err = rows.Scan(
			&req.Id,
			&req.Method,
			&req.Path,
			&req.Host,
			&headersJSON,
			&req.Params,
			&cookiesJSON,
			&req.Body,
			&req.Schema,
		)

		if err != nil {
			fmt.Printf("don't scan requests: %s\n", err)
			return nil, err
		}

		err = json.Unmarshal(headersJSON, &req.Headers)
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

func (repo api) getRequestById(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/requests/"):]
	req, err := repo.getRequest(r, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body := fmt.Sprintf("%d\n", req.Id)
	body += fmt.Sprintf("%s %s?%s HTTP/1.1\n", req.Method, req.Path, req.Params)
	body += fmt.Sprintf("Host: %s\n", req.Host)

	for key, val := range req.Headers {
		body += fmt.Sprintf("%s: %s\n", key, val)
	}

	if req.Method == "POST" {
		body += fmt.Sprintf("\n%s", req.Body)
	}

	_, err = io.Copy(w, strings.NewReader(body))
	if err != nil {
		fmt.Printf("write error: %s", err)
	}
}

func (repo api) getRequests(w http.ResponseWriter, r *http.Request) {
	query := "select id, method, path, host, headers, params, cookies, body from requests"

	rows, err := repo.DB.Query(query)
	if err != nil {
		fmt.Printf("don't get requests: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	requests := make([]models.Request, 0)
	for rows.Next() {
		request := new(models.Request)
		var headersJSON, cookiesJSON []byte

		err = rows.Scan(
			&request.Id,
			&request.Method,
			&request.Path,
			&request.Host,
			&headersJSON,
			&request.Params,
			&cookiesJSON,
			&request.Body,
		)

		if err != nil {
			fmt.Printf("don't scan requests: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		err = json.Unmarshal(headersJSON, &request.Headers)
		if err != nil {
			fmt.Printf("error Unmarshal headers: %s", err)
		}

		requests = append(requests, *request)
	}

	result := ""
	for _, req := range requests {
		body := fmt.Sprintf("%d\n", req.Id)
		body += fmt.Sprintf("%s %s?%s HTTP/1.1\n", req.Method, req.Path, req.Params)
		body += fmt.Sprintf("Host: %s\n", req.Host)

		result += body + "\n\n"
	}

	_, err = io.Copy(w, strings.NewReader(result))
	if err != nil {
		fmt.Printf("write error: %s", err)
	}
}
