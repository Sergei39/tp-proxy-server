package repeater

import (
	"fmt"
	"github.com/jackc/pgx"
	"net/http"
	"strings"
)

type Repeater interface {
	SaveRequest(r http.Request)
}

type repeater struct {
	DB *pgx.ConnPool
}

func NewRepeater(DB *pgx.ConnPool) Repeater {
	return repeater{
		DB: DB,
	}
}

func (h repeater) SaveRequest(r http.Request) {
	id := h.saveMethod(r)
	h.saveParams(r, id)
	h.saveHeaders(r, id)
	h.saveCookies(r, id)
}

func (h repeater) saveMethod(r http.Request) int {
	query := `INSERT INTO requests (method, path) VALUES ($1, $2) returning id`

	pathParams := strings.Split(r.RequestURI, "?")
	var id int
	err := h.DB.QueryRow(query, r.Method, pathParams[0]).Scan(&id)
	if err != nil {
		fmt.Printf("don't save request: %s\n", err)
	}
	return id
}

func (h repeater) saveParams(r http.Request, id int) {
	pathParams := strings.Split(r.RequestURI, "?")
	if len(pathParams) > 1{
		query := `INSERT INTO params (request, name, value, type) VALUES `
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

		_, err := h.DB.Exec(query, queryParams...)
		if err != nil {
			fmt.Printf("don't save params: %s\n", err)
		}
		return
	}

	fmt.Printf("params not found\n")
}

func (h repeater) saveHeaders(r http.Request, id int) {
	query := `INSERT INTO headers (parent, name, value) VALUES `
	var queryParams []interface{}
	i := 0
	r.Cookies()
	for key, val := range r.Header {
		valStr := strings.Join(val, ",")
		query += fmt.Sprintf("(%d, $%d, $%d),", id, i*2+1, i*2+2)
		queryParams = append(queryParams, key, valStr)
		i++
	}
	query = query[:len(query)-1]
	_, err := h.DB.Exec(query, queryParams...)
	if err != nil {
		fmt.Printf("don't save headers: %s\n", err)
	}
}

func (h repeater) saveCookies(r http.Request, id int) {
	query := `INSERT INTO cookies (request, name, value) VALUES `
	var queryParams []interface{}
	for i, val := range r.Cookies() {
		query += fmt.Sprintf("(%d, $%d, $%d),", id, i*2+1, i*2+2)
		queryParams = append(queryParams, val.Name, val)
		i++
	}
	query = query[:len(query)-1]
	_, err := h.DB.Exec(query, queryParams...)
	if err != nil {
		fmt.Printf("don't save cookies: %s\n", err)
	}
}

