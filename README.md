# `yiff` YAML diff tool

## Run as a Service

From the root of the repository:

### Start the Server
```shell
$ go run cmd/web/main.go
2025/07/30 05:25:48 Starting web server on :8080
```

### Validate Availability

Again, from the root of the repository:

```shell
$ curl -sSd @tests/testdata.json http://localhost:8080/diff | jq        

{
  "variables": [
    {
      "name": "key",
      "default": "value",
      "value": "newvalue",
      "status": "changed"
    }
  ]
}
```
