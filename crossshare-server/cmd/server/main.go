package main

import (
	"go.uber.org/fx"

	"crossshare-server/internal/config"
	"crossshare-server/internal/handler"
	"crossshare-server/internal/logger"
	"crossshare-server/internal/server"
	"crossshare-server/internal/service"
	"crossshare-server/internal/storage"
)

func main() {
	fx.New(
		config.Module,
		logger.Module,
		storage.Module,
		service.Module,
		handler.Module,
		server.Module,
	).Run()
}
