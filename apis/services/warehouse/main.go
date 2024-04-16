package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/EnesDemirtas/medisync/apis/services/warehouse/build/crud"
	"github.com/EnesDemirtas/medisync/apis/services/warehouse/mux"
	"github.com/EnesDemirtas/medisync/app/api/authsrv"
	"github.com/EnesDemirtas/medisync/app/api/debug"
	"github.com/EnesDemirtas/medisync/business/core/crud/delegate"
	"github.com/EnesDemirtas/medisync/business/core/crud/userbus"
	"github.com/EnesDemirtas/medisync/business/core/crud/userbus/stores/userdb"
	"github.com/EnesDemirtas/medisync/business/data/sqldb"
	"github.com/EnesDemirtas/medisync/foundation/logger"
	"github.com/EnesDemirtas/medisync/foundation/web"
	"github.com/ardanlabs/conf/v3"
)

var build = "develop"
var routes = "crud"

func main() {
	var log *logger.Logger

	events := logger.Events{
		Error: func(ctx context.Context, r logger.Record) {
			log.Info(ctx, "******* SEND ALERT *******")
		},
	}

	traceIDFn := func(ctx context.Context) string {
		return web.GetTraceID(ctx)
	}

	log = logger.NewWithEvents(os.Stdout, logger.LevelInfo, "WAREHOUSE", traceIDFn, events)

	// ----------------------------------------------------------

	ctx := context.Background()

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "msg", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {
	// ----------------------------------------------------------------
	// GOMAXPROCS

	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// ----------------------------------------------------------------
	// Configuration

	cfg := struct {
		conf.Version
		Web struct {
			ReadTimeout        time.Duration `conf:"default:5s"`
			WriteTimeout       time.Duration `conf:"default:10s"`
			IdleTimeout        time.Duration `conf:"default:120s"`
			ShutdownTimeout    time.Duration `conf:"default:20s"`
			APIHost            string        `conf:"default:0.0.0.0:3000"`
			DebugHost          string        `conf:"default:0.0.0.0:3010"`
			CORSAllowedOrigins []string      `conf:"default:*"`
		}
		Auth struct {
			Host string `conf:"default:http://auth-service.warehouse-system.svc.cluster.local:6000"`
		}
		DB struct {
			User         string `conf:"default:postgres"`
			Password     string `conf:"default:postgres,mask"`
			HostPort     string `conf:"default:database-service.warehouse-system.svc.cluster.local"`
			Name         string `conf:"default:postgres"`
			MaxIdleConns int    `conf:"default:2"`
			MaxOpenConns int    `conf:"default:0"`
			DisableTLS   bool   `conf:"default:true"`
		}
		// Tempo struct {
		// 	ReporterURI string  `conf:"default:tempo.sales-system.svc.cluster.local:4317"`
		// 	ServiceName string  `conf:"default:sales"`
		// 	Probability float64 `conf:"default:0.05"`
		// 	// Shouldn't use a high Probability value in non-developer systems.
		// 	// 0.05 should be enough for most systems. Some might want to have
		// 	// this even lower.
		// }
	}{
		Version: conf.Version{
			Build: build,
			Desc:  "Warehouse",
		},
	}

	const prefix = "WAREHOUSE"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// ----------------------------------------------------------------
	// App Starting

	log.Info(ctx, "starting service", "version", cfg.Version.Build)
	defer log.Info(ctx, "shutdown complete")

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output: %w", err)
	}
	log.Info(ctx, "startup", "config", out)

	expvar.NewString("build").Set(cfg.Version.Build)

	// -----------------------------------------------------------
	// Database Support

	log.Info(ctx, "startup", "status", "initializing database support", "hostport", cfg.DB.HostPort)

	db, err := sqldb.Open(sqldb.Config{
		User:         cfg.DB.User,
		Password:     cfg.DB.Password,
		HostPort:     cfg.DB.HostPort,
		Name:         cfg.DB.Name,
		MaxIdleConns: cfg.DB.MaxIdleConns,
		MaxOpenConns: cfg.DB.MaxOpenConns,
		DisableTLS:   cfg.DB.DisableTLS,
	})
	if err != nil {
		return fmt.Errorf("connecting to db: %w", err)
	}

	defer db.Close()

	// --------------------------------------------------------------
	// Initialize authentication support

	log.Info(ctx, "startup", "status", "initializing authentication support")

	logFunc := func(ctx context.Context, msg string) {
		log.Info(ctx, "authapi", "message", msg)
	}
	authSrv := authsrv.New(cfg.Auth.Host, logFunc)

	// --------------------------------------------------------------
	// Start Tracing Support


	// --------------------------------------------------------------
	// Build Core APIs

	log.Info(ctx, "startup", "status", "initializing business support")

	delegate := delegate.New(log)
	userBus  := userbus.NewCore(log, delegate, userdb.NewStore(log, db))

	// ---------------------------------------------------------------
	// Start Debug Service

	go func() {
		log.Info(ctx, "startup", "status", "debug v1 router started", "host", cfg.Web.DebugHost)

		if err := http.ListenAndServe(cfg.Web.DebugHost, debug.Mux()); err != nil {
			log.Error(ctx, "shutdown", "status", "debug v1 router closed", "host", cfg.Web.DebugHost, "msg", err)
		}
	}()

	// ----------------------------------------------------------------
	// Start API Service

	log.Info(ctx, "startup", "status", "initializing V1 API support")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	cfgMux := mux.Config{
		Build: 		build,
		Shutdown:	shutdown,
		Log:		log,
		AuthSrv: 	authSrv,
		DB:			db,
		BusDomain:	mux.BusDomain{
			Delegate: 	delegate,
			User:		userBus,
		},
	}

	api := http.Server{
		Addr: 		  cfg.Web.APIHost,
		Handler:	  mux.WebAPI(cfgMux, buildRoutes(), mux.WithCORS(cfg.Web.CORSAllowedOrigins)),
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     logger.NewStdLogger(log, logger.LevelError),	
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info(ctx, "startup", "status", "api router started", "host", api.Addr)

		serverErrors <- api.ListenAndServe()
	}()

	// ------------------------------------------------------------------------
	// Shutdown

	select {
	case err := <- serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <- shutdown:
		log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(ctx, cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}

func buildRoutes() mux.RouteAdder {

		// The idea here is that we can build different versions of the binary
	// with different sets of exposed web APIs. By default we build a single
	// an instance with all the web APIs.
	//
	// Here is the scenario. It would be nice to build two binaries, one for the
	// transactional APIs (CRUD) and one for the reporting APIs. This would allow
	// the system to run two instances of the database. One instance tuned for the
	// transactional database calls and the other tuned for the reporting calls.
	// Tuning meaning indexing and memory requirements. The two databases can be
	// kept in sync with replication.

	// switch routes {
	// case "crud":
	// 	return crud.Routes()

	// case "reporting":
	// 	return reporting.Routes()
	// }

	// return all.Routes()

	return crud.Routes()
}