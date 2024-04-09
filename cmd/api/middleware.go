package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	// 각 클라이언트에 대한 속도 제한기와 마지막으로 확인된 시간을 보유하도록 클라이언트 구조체를 정의합니다.
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu sync.Mutex
		// 값이 클라이언트 구조체에 대한 포인터가 되도록 맵을 업데이트합니다.
		clients = make(map[string]*client)
	)

	// 1분마다 한 번씩 클라이언트 맵에서 오래된 항목을 제거하는 백그라운드 고루틴을 실행합니다.
	go func() {
		for {
			time.Sleep(time.Minute)

			// 정리가 진행되는 동안 속도 제한기 검사가 발생하지 않도록 뮤텍스를 잠급니다.
			mu.Lock()

			// 모든 클라이언트를 반복합니다. 지난 3분 동안 표시되지 않은 경우 지도에서 해당 항목을 삭제하세요.
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			// 중요한 것은 정리가 완료되면 뮤텍스를 잠금 해제하는 것입니다.
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		mu.Lock()

		if _, found := clients[ip]; !found {
			// 아직 존재하지 않는 경우 새 클라이언트 구조체를 생성하여 맵에 추가합니다.
			clients[ip] = &client{limiter: rate.NewLimiter(2, 4)}
		}

		// 클라이언트의 마지막 확인 시간을 업데이트합니다.
		clients[ip].lastSeen = time.Now()

		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.rateLimitExceedeResponse(w, r)
			return
		}

		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
