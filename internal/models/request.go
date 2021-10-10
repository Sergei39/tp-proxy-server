package models

type Request struct {
	Id int
	Method string
	Path string
	Host string
	Headers map[string][]string
	Cookies map[string][]string
	Params string
	Body string
	Schema string
}
