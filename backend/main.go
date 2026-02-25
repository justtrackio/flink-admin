package main

import (
	"context"
	"embed"

	"github.com/gin-contrib/cors"
	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/flink-admin/internal"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:embed config.dist.yml
var configDist []byte

//go:embed public
var publicFs embed.FS

func main() {
	application.New(
		application.WithConfigDebug,
		application.WithConfigBytes(configDist, "yml"),
		application.WithConfigEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		application.WithConfigFileFlag,
		application.WithConfigSanitizers(cfg.TimeSanitizer),
		application.WithLoggerHandlersFromConfig,
		application.WithUTCClock(true),
		application.WithModuleFactory("k8s-watcher", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return internal.ProvideDeploymentWatcherModule(ctx, config, logger)
		}),
		application.WithModuleFactory("http", httpserver.NewServer("default", func(ctx context.Context, config cfg.Config, logger log.Logger, router *httpserver.Router) error {
			router.Use(cors.Default())
			router.UseFactory(httpserver.CreateEmbeddedStaticServe(publicFs, "public", "/api"))

			router.Group("/api/deployments").HandleWith(httpserver.With(internal.NewHandlerDeployments, func(r *httpserver.Router, handler *internal.HandlerDeployments) {
				r.GET("/watch", httpserver.BindSseN(handler.WatchDeployments))
			}))

			deploymentGroup := router.Group("/api/deployments/:namespace/:name")
			deploymentGroup.HandleWith(httpserver.With(internal.NewHandlerCheckpoints, func(r *httpserver.Router, handler *internal.HandlerCheckpoints) {
				r.GET("/checkpoints", httpserver.Bind(handler.GetCheckpoints))
			}))
			deploymentGroup.HandleWith(httpserver.With(internal.NewHandlerStorageCheckpoints, func(r *httpserver.Router, handler *internal.HandlerStorageCheckpoints) {
				r.GET("/storage-checkpoints", httpserver.Bind(handler.GetStorageCheckpoints))
			}))
			deploymentGroup.HandleWith(httpserver.With(internal.NewHandlerEvents, func(r *httpserver.Router, handler *internal.HandlerEvents) {
				r.GET("/events", httpserver.Bind(handler.GetEvents))
			}))
			deploymentGroup.HandleWith(httpserver.With(internal.NewHandlerExceptions, func(r *httpserver.Router, handler *internal.HandlerExceptions) {
				r.GET("/exceptions", httpserver.Bind(handler.GetExceptions))
			}))

			return nil
		})),
	).Run()
}
