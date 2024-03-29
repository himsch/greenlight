package main

import (
	"fmt"
	"net/http"
)

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	// 이스케이프할 필요 없이 JSON의 문자를 사용합니까?
	// %q 동사를 사용하여 보간된 값을 큰따옴표로 묶습니다.
	js := `{"status": "available", "environment": %q, "version": %q}`
	js = fmt.Sprintf(js, app.config.env, version)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte(js))
}
