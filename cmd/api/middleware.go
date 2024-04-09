package main

import (
	"fmt"
	"net/http"

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
	// 초당 평균 2개의 요청을 허용하고
	// 단일 '버스트'에서 최대 4개의 요청을 허용하는 새로운 속도 제한기를 초기화합니다.
	limiter := rate.NewLimiter(2, 4)

	// 우리가 반환하는 함수는 리미터 변수를 '닫는' 클로저입니다.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 요청이 허용되는지 확인하려면limiter.Allow()를 호출하고,
		// 허용되지 않으면 rateLimitExceededResponse() 도우미를 호출하여
		// 429 너무 많은 요청 응답을 반환합니다.
		if !limiter.Allow() {
			app.rateLimitExceedeResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
