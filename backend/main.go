package main

import (
	"context"
	"embed"

	"github.com/gin-contrib/cors"
	"github.com/gosoline-project/httpserver"
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
			return ProvideDeploymentWatcherModule(ctx, config, logger)
		}),
		application.WithModuleFactory("http", httpserver.NewServer("default", func(ctx context.Context, config cfg.Config, logger log.Logger, router *httpserver.Router) error {
			router.Use(cors.Default())
			router.UseFactory(httpserver.CreateEmbeddedStaticServe(publicFs, "public", "/api"))

			router.Group("/api/deployments").HandleWith(httpserver.With(NewHandlerDeployments, func(r *httpserver.Router, handler *HandlerDeployments) {
				r.GET("/watch", httpserver.BindSseN(handler.WatchDeployments))
			}))

			router.Group("/api/deployments/:namespace/:name").HandleWith(httpserver.With(NewHandlerCheckpoints, func(r *httpserver.Router, handler *HandlerCheckpoints) {
				r.GET("/checkpoints", httpserver.Bind(handler.GetCheckpoints))
			}))

			return nil
		})),
	).Run()
}
