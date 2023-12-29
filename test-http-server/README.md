You'll need root privledges to fire off server on port 80 (or anything under 1080)

`go run test-server.go`

or 

`go build test-server.go`
`sudo ./test-server`


Fire off the fetch code in parent directory with `test.yaml` instead of `fetch.yaml`

You'll see output like the following:
```
GET / HTTP/1.1
Host: localhost
Accept-Encoding: gzip
User-Agent: fetch-synthetic-monitor


GET /careers HTTP/1.1
Host: localhost
Accept-Encoding: gzip
User-Agent: fetch-synthetic-monitor


POST /some/post/endpoint HTTP/1.1
Host: localhost
Accept-Encoding: gzip
Content-Length: 13
Content-Type: application/json
Foo: bar
User-Agent: fetch-synthetic-monitor

{"foo":"bar"}
```
