# fetch
Make a HTTP request to endpoints defined in a yaml file with optional parameters.

# Setup
```
go mod init fetch
go get gopkg.in/yaml.v3
```
# Usage

`go run fetch.go fetch.yaml`

or compile with:

```
go build fetch.go
./fetch fetch.yaml
```

# Additional notes
There is a very simple test HTTP server to visually verify HTTP requests locally in the `test-http-server/` directory.
Use the `test.yaml` file for localhost endpoints.
