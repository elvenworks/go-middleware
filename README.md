
Package go-middleware is a collection of middlewares for the Gin web framework to be used in Elvenworks applications.

## Installation
Use go get.
```
go get github.com/elvenworks/go-middleware
```
Then import the validator package into your own code.
```
import "github.com/elvenworks/go-middleware"
```

## Usage logger
Sample code:
```go
import (
	middleware "github.com/elvenworks/go-middleware"
)

func InitRoutes() {
	skipPaths := []string{
		"/docs",
		"/api/private/v1/healthz",
		"/metrics",
	}

	logger := middleware.NewLogger(skipPaths, logs.GetLoggerLevel())
	logger.Use(httpServer.Router)
}
```
## Usage prometheus
Sample code:
```go
import (
	middleware "github.com/elvenworks/go-middleware"
)

func InitRoutes() {
	p := middleware.NewPrometheus("gin")
	p.Use(httpServer.Router)
}
```

## Usage auth_jwt
Sample code:
```go
import (
	middleware "github.com/elvenworks/go-middleware"
)

func New() *HTTP {
	router := gin.New()
	router.Use(middleware.NewAuthJWT())
}
```