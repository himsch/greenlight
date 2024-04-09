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

	// shutdownError 채널을 만듭니다. 우리는 이것을 사용하여
	// Shutdown() 함수에 의해 반환된 오류를 수신합니다.
	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		// "caught signal" 대신 "shutting down server"으로 로그 항목을 업데이트합니다.
		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		// 20초 제한 시간으로 컨텍스트를 생성합니다.
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		// 방금 만든 컨텍스트를 전달하여 서버에서 Shutdown()을 호출합니다.
		// Shutdown()은 단계적 종료가 성공했거나 오류(리스너를 닫는 데 문제가 있거나
		// 20초 컨텍스트 기한에 도달하기 전에 종료가 완료되지 않았기 때문에 발생할 수 있음)
		// 인 경우 nil을 반환합니다. 이 반환 값을 shutdownError 채널에 전달합니다.
		shutdownError <- srv.Shutdown(ctx)
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	// 서버에서 Shutdown()을 호출하면 ListenAndServe()가 즉시 http.ErrServerClosed 오류를 반환하게 됩니다.
	// 따라서 이 오류가 표시되면 이는 실제로 좋은 것이며 정상적인 종료가 시작되었음을 나타냅니다.
	// 따라서 우리는 이를 구체적으로 확인하고 http.ErrServerClosed가 아닌 경우에만 오류를 반환합니다.
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// 그렇지 않으면 Shutdown()에서 반환 값을 받기를 기다립니다.
	// shutdownError 채널. 반환 값이 오류인 경우 정상적인 종료에 문제가 있음을 알고 오류를 반환합니다.
	err = <-shutdownError
	if err != nil {
		return err
	}

	// 이 시점에서 우리는 정상적인 종료가 성공적으로 완료되었음을 확인하고 "서버 중지됨" 메시지를 기록합니다.
	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
