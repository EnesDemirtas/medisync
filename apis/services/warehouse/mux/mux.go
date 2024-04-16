// Package mux provides support to bind domain level routes
// to the application mux.
package mux

import (
	"net/http"
	"os"

	"github.com/EnesDemirtas/medisync/app/api/authsrv"
	midhttp "github.com/EnesDemirtas/medisync/app/api/mid/http"
	"github.com/EnesDemirtas/medisync/business/core/crud/delegate"
	"github.com/EnesDemirtas/medisync/business/core/crud/userbus"
	"github.com/EnesDemirtas/medisync/foundation/logger"
	"github.com/EnesDemirtas/medisync/foundation/web"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
)

// Options represent optional parameters.
type Options struct {
	corsOrigin []string
}

// WithCORS provides configuration options for CORS.
func WithCORS(origins []string) func(opts *Options) {
	return func(opts *Options) {
		opts.corsOrigin = origins
	}
}

// BusDomain represents the set of core business packages.
type BusDomain struct {
	Delegate *delegate.Delegate
	User     *userbus.Core
}

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Build     string
	Shutdown  chan os.Signal
	Log       *logger.Logger
	AuthSrv   *authsrv.AuthSrv
	DB        *sqlx.DB
	Tracer    trace.Tracer
	BusDomain BusDomain
}

// RouteAdder defines behavior that sets the routes to bind for an instance
// of the service.
type RouteAdder interface {
	Add(app *web.App, cfg Config)
}

// WebAPI constructs a http.Handler with all application routes bound.
func WebAPI(cfg Config, routeAdder RouteAdder, options ...func(opts *Options)) http.Handler {
	var opts Options
	for _, option := range options {
		option(&opts)
	}

	app := web.NewApp(
		cfg.Shutdown,
		cfg.Tracer,
		midhttp.Logger(cfg.Log),
		midhttp.Errors(cfg.Log),
		midhttp.Metrics(),
		midhttp.Panics(),
	)

	if len(opts.corsOrigin) > 0 {
		app.EnableCORS(midhttp.Cors(opts.corsOrigin))
	}

	routeAdder.Add(app, cfg)

	return app
}