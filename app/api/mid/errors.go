package mid

import (
	"context"
	"net/http"

	"github.com/EnesDemirtas/medisync/app/api/errs"
	"github.com/EnesDemirtas/medisync/foundation/logger"
	"github.com/EnesDemirtas/medisync/foundation/web"
)

// Errors handles errors coming out of the call chain. It detects normal
// application errors which are used to respond to the client in a uniform way.
// Unexpected errors (status >= 500) are logged.
func Errors(log *logger.Logger) web.MidHandler {
	m := func(handler web.Handler) web.Handler {
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			err := handler(ctx, w, r)
			if err == nil {
				return nil
			}

			log.Error(ctx, "message", "msg", err)

			ctx, span := web.AddSpan(ctx, "business.web.request.mid.error")
			span.RecordError(err)
			span.End()

			var er errs.Error
			var status int

			switch {
			case errs.IsError(err):
				er = errs.GetError(err)
				status = er.Details.HTTPStatusCode

			default:
				er = errs.Error{
					Code:    errs.Unknown,
					Message: http.StatusText(http.StatusInternalServerError),
				}
				status = http.StatusInternalServerError
			}

			if err := web.Respond(ctx, w, er, status); err != nil {
				return err
			}

			// If we receive the shutdown err we need to return it
			// back to the base handler to shut down the service.
			if web.IsShutdown(err) {
				return err
			}

			return nil
		}

		return h
	}

	return m
}
