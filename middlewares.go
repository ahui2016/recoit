package main

import (
	"fmt"
	"net/http"
	"strings"
)

// func handlerToFunc(h http.Handler) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		h.ServeHTTP(w, r)
// 	}
// }

func checkLoginForFileServer(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if passwordTry >= passwordMaxTry {
			db.Reset()
			http.NotFound(w, r)
			return
		}
		if isLoggedOut(r) {
			http.NotFound(w, r)
			return
		}
		h.ServeHTTP(w, r)
	}
}

func checkLogin(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isLoggedOut(r) {
			// 凡是以 "/api/" 开头的请求都返回 json 消息。
			if strings.HasPrefix(r.URL.Path, "/api/") {
				jsonRequireLogin(w)
				return
			}
			// 不是以 "/api/" 开头的都是页面。
			fmt.Fprint(w, HTML["login"])
			return
		}
		fn(w, r)
	}
}

// 限制从前端传输过来的数据大小。
func setMaxBytes(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		fn(w, r)
	}
}

func isLoggedIn(r *http.Request) bool {
	return db.IsReady() && db.Sess.Check(r)
}

func isLoggedOut(r *http.Request) bool {
	return !isLoggedIn(r)
}

func checkPasswordTry(w http.ResponseWriter) bool {
	if passwordTry >= passwordMaxTry {
		// log.Fatal()
		db.Reset()
		jsonMessage(w, "No more try. Input wrong password too many times.", 403)
		return true
	}
	return false
}
