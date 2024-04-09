package main

import (
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

	go func() {
		// os.Signal 값을 전달하는 종료 채널을 만듭니다.
		quit := make(chan os.Signal, 1)

		// signal.Notify()를 사용하여 들어오는 SIGINT 및 SIGTERM 신호를 수신하고 이를 종료 채널에 전달합니다.
		// 다른 신호는 signal.Notify()에 의해 포착되지 않으며 기본 동작을 유지합니다.
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// 종료 채널에서 신호를 읽으십시오. 이 코드는 신호가 수신될 때까지 Block 됩니다.
		s := <-quit

		// 신호가 포착되었음을 알리는 메시지를 기록합니다.
		// 신호 이름을 가져오고 이를 로그 항목 속성에 포함하기 위해 신호에 대해 String() 메서드도 호출합니다.
		app.logger.PrintInfo("caught signal", map[string]string{
			"signal": s.String(),
		})

		// 0(성공) 상태 코드로 애플리케이션을 종료합니다.
		os.Exit(0)
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})
	return srv.ListenAndServe()
}
