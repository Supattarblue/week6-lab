package mw

import (
    "encoding/json"
    "log"
    "net/http"
    "time"
)

func Logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s ใช้เวลา %v", r.Method, r.URL.Path,
            time.Since(start).Round(time.Millisecond))
    })
}

func Recovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("เชฟทำจานหล่น %v", err)
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusInternalServerError)
                json.NewEncoder(w).Encode(map[string]any{
                    "error": map[string]string{
                        "code":    "INTERNAL_ERROR",
                        "message": "ครัวมีปัญหา ลองใหม่อีกครั้ง",
                    },
                })
            }
        }()
        next.ServeHTTP(w, r)
    })
}

// allowedOrigins คือรายชื่อเว็บ (frontend) ที่อนุญาตให้เรียก API นี้ได้
var allowedOrigins = map[string]bool{
    "http://localhost:5173":                 true, // Vite
    "http://localhost:3000":                 true, // MovieHub (localhost)
    "http://127.0.0.1:3000":                 true, // MovieHub (127.0.0.1)
    "https://moviehub-650710728.vercel.app": true, // MovieHub บน Vercel
}

func CORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        if allowedOrigins[origin] {
            w.Header().Set("Access-Control-Allow-Origin", origin)
            w.Header().Set("Vary", "Origin")
        }
        w.Header().Set("Access-Control-Allow-Methods",
            "GET, POST, PUT, PATCH, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return // ใบสั่งถามทาง จบตรงนี้ ไม่ต้องเข้าครัว
        }
        next.ServeHTTP(w, r)
    })
}