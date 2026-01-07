package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

func (app *application) serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		ErrorLog:     log.New(app.logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownError := make(chan error)
	go func(ctx context.Context) {
		<-ctx.Done()
		app.logger.Info().Str("component", "server").Msg("stopping")

		serverCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(serverCtx)
		if err != nil {
			app.logger.Error().Err(err).Str("component", "server").Msg("error stopping")
			shutdownError <- err
		}

		app.logger.Info().Str("component", "server").Msg("completing background tasks")

		app.wg.Wait()
		shutdownError <- nil
	}(ctx)

	app.logger.Info().Str("component", "server").Str("address", srv.Addr).Str("env", app.config.env).Str("node", app.config.node).Msg("starting")

	// http.ErrServerClosed is expected during a graceful shutdown, so we'll filter it out
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server listen and serve: %w", err)
	}

	err = <-shutdownError
	if err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	app.logger.Info().Str("component", "server").Msg("stopped")

	return nil
}
