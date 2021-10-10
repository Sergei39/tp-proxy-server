package api

import (
	"fmt"
	"github.com/Sergei39/tp-proxy-server/internal/models"
	proxy_server "github.com/Sergei39/tp-proxy-server/internal/proxy-server"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

const checkString = "sdfkksvsd"

type Scanner interface {
	StartScan(req *models.Request) ([]string, error)
}

type scanner struct {
	ps proxy_server.Server
}

func NewScanner(ps proxy_server.Server) Scanner {
	return scanner{
		ps: ps,
	}
}

func (sc scanner) StartScan(req *models.Request) ([]string, error) {
	params, err := sc.getParamsFromFile()
	if err != nil {
		return nil, err
	}

	answer := make([]string, 0)
	var group sync.WaitGroup
	group.Add(100)
	in := make(chan string)
	out := make(chan string)
	for i := 0; i < 100; i++ {
		go sc.checkParam(req, in, out, &group)
	}

	go func(out chan string, answer *[]string) {
		for {
			val, ok := <- out
			if !ok {
				fmt.Println("Stop reader")
				break
			}

			*answer = append(*answer, val)
		}
	}(out, &answer)

	for _, val := range *params {
		in <- val
	}
	close(in)
	group.Wait()
	close(out)

	return answer, nil
}

func (sc scanner) checkParam(req *models.Request, in, out chan string, group *sync.WaitGroup) {
	for {
		val, ok := <- in
		if !ok {
			break
		}
		request := &http.Request{
			Method: req.Method,
			URL: &url.URL{
				Scheme: req.Schema,
				Host:   req.Host,
				Path:   fmt.Sprintf("%s?%s=%s", req.Path, val, checkString),
			},
			Header: req.Headers,
			Body:   ioutil.NopCloser(strings.NewReader(req.Body)),
			Host:   req.Host,
		}

		resp, err := http.DefaultTransport.RoundTrip(request)
		if err != nil {
			fmt.Printf("RoundTrip error: %s", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("ReadAll error: %s", err)
		}

		if strings.Contains(string(body), checkString) {
			out <- val
		}
	}

	group.Done()
}

func (sc scanner) getParamsFromFile() (*[]string, error) {
	file, err := os.Open("params.txt")
	if err != nil{
		fmt.Printf("fail open file: %s", err)
		return nil, err
	}
	defer file.Close()

	var chunk []byte
	buf := make([]byte, 1024)

	for {
		// Читаем из файла в buf
		n, err := file.Read(buf)
		if err != nil && err != io.EOF{
			fmt.Printf("read buf fail: %s", err)
			return nil, err
		}
		// Обозначим конец чтения
		if n == 0 {
			break
		}
		// Считываем в последний буфер
		chunk = append(chunk, buf[:n]...)
	}

	params := strings.Split(string(chunk), "\n")
	return &params, nil
}
