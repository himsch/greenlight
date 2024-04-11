package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		ErrorLog:     log.New(app.logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		// 이전처럼 서버에서 Shutdown()을 호출하지만
		// 이제는 오류가 반환되는 경우에만 shutdownError 채널로 보냅니다.
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		// 백그라운드 고루틴이 작업을 완료하기를 기다리고 있다는 메시지를 기록하세요.
		app.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		// WaitGroup 카운터가 0이 될 때까지 차단하려면 Wait()를 호출하세요.
		// 즉, 기본적으로 백그라운드 고루틴이 완료될 때까지 차단됩니다.
		// 그런 다음 shutdownError 채널에 nil을 반환하여 shutdown이 없이 완료되었음을 나타냅니다.
		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
