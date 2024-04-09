package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"

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
	// 클라이언트의 IP 주소와 속도 제한기를 보관할 뮤텍스와 맵을 선언합니다.
	var (
		mu      sync.Mutex
		clients = make(map[string]*rate.Limiter)
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 요청에서 클라이언트의 IP 주소를 추출합니다.
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// 이 코드가 동시에 실행되지 않도록 뮤텍스를 잠급니다.
		mu.Lock()

		// 해당 IP 주소가 이미 맵에 존재하는지 확인하세요.
		// 그렇지 않은 경우 새 속도 제한기를 초기화하고 IP 주소와 제한기를 맵에 추가합니다.
		if _, found := clients[ip]; !found {
			clients[ip] = rate.NewLimiter(2, 4)
		}

		// 현재 IP 주소에 대한 속도 제한기에서 Allow() 메서드를 호출합니다.
		// 요청이 허용되지 않으면 이전과 마찬가지로 뮤텍스를 잠금 해제하고 429 Too Many Requests 응답을 보냅니다.
		if !clients[ip].Allow() {
			mu.Unlock()
			app.rateLimitExceedeResponse(w, r)
			return
		}

		// 매우 중요한 것은 체인의 다음 핸들러를 호출하기 전에 뮤텍스를 잠금 해제하는 것입니다.
		// 뮤텍스를 잠금 해제하기 위해 *defer*를 사용하지 않는다는 점에 유의하십시오.
		// 이는 이 미들웨어의 다운스트림 모든 핸들러도 반환될 때까지 뮤텍스가 잠금 해제되지 않음을 의미하기 때문입니다.
		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
