package proxy_server

import (
	"fmt"
	db_proxy "github.com/Sergei39/tp-proxy-server/internal/db-proxy"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type Server interface {
	Handler(w http.ResponseWriter, r *http.Request)
	SaveMiddleware(next http.HandlerFunc) http.HandlerFunc
}

type server struct {
	db db_proxy.Repeater
}

func NewProxyServer(db db_proxy.Repeater) Server {
	return &server{
		db,
	}
}

func (h server) proxyHTTPS(w http.ResponseWriter, r *http.Request) {
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

func (h server) proxyHTTP(w http.ResponseWriter, r *http.Request) {
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

func (h server) SaveMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("Proxy-Connection")

		if r.Method == http.MethodConnect {
			h.db.SaveRequest(*r, "https")
		} else {
			host := strings.Split(r.Host, ":")[0]
			if len(strings.Split(r.RequestURI, host)) > 1 {
				r.RequestURI = strings.Split(r.RequestURI, host)[1]
			}
			h.db.SaveRequest(*r, "http")
		}

		next.ServeHTTP(w, r)
	})
}

func (h server) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		fmt.Println("Connect https")
		h.proxyHTTPS(w, r)
	} else {
		fmt.Println("Connect http")
		h.proxyHTTP(w, r)
	}
}
