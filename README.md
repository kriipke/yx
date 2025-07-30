# `yiff` YAML diff tool

## Run as a Service

From the root of the repository:

### Start the Server
```shell
$ go run cmd/api/main.go
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

## Run as a Container

From the root of the repository:

```shell
$ docker build -t yiff-api .
```

```shell
$ docker run --rm -p "8080:8080" yiff-api
```
