package main

import (
	"context"
	"os"

	"github.com/clint456/edgex-go/internal/support/mappings"
	"github.com/labstack/echo/v4"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	mappings.Main(ctx, cancel, echo.New(), os.Args[1:])
}
