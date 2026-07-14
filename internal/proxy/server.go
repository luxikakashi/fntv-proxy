package proxy

import (
"bytes"
"context"
"fntv-proxy/internal/cache"
"fntv-proxy/internal/config"
"fntv-proxy/internal/handler"
"fntv-proxy/internal/logger"
"io"
"net/http"
"net/http/httputil"
"net/url"
"strings"
"time"
)

type Server struct {
config          *config.Config
logger          *logger.Logger
cache           *cache.Cache
playbackHandler *handler.PlaybackHandler
streamHandler   *handler.StreamHandler
proxy           *httputil.ReverseProxy
httpServer      *http.Server
}

func NewServer(cfg *config.Config) (*Server, error) {
log := logger.New(cfg.GetLogLevel(), cfg.LogDir)
targetURL, err := url.Parse(cfg.GetTargetAddr())
if err != nil { return nil, err }
c := cache.NewWithStreamTTL(cfg.GetCacheTTL())
ph := handler.NewPlaybackHandler(c, log)
sh := handler.NewStreamHandler(c, log, cfg.GetTargetAddr())
proxy := httputil.NewSingleHostReverseProxy(targetURL)
return &Server{config: cfg, logger: log, cache: c, playbackHandler: ph, streamHandler: sh, proxy: proxy}, nil
}

func (s *Server) Start() error {
originalDirector := s.proxy.Director
s.proxy.Director = func(req *http.Request) { originalDirector(req); req.Host = s.config.GetTargetAddr() }
s.proxy.ModifyResponse = s.handleResponse
s.httpServer = &http.Server{Addr: s.config.GetListenAddr(), Handler: s.loggingMiddleware(s.proxy)}
return s.httpServer.ListenAndServe()
}

func (s *Server) Stop() error {
if s.httpServer != nil {
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
return s.httpServer.Shutdown(ctx)
}
return nil
}

func (s *Server) Reload() {
s.logger.SetLevel(s.config.GetLogLevel())
s.logger.Info("配置已重载，新日志级别: %s", s.config.GetLogLevel())
}

func (s *Server) handleResponse(resp *http.Response) error {
if !s.isPlaybackInfoRequest(resp.Request) { return nil }
body, err := io.ReadAll(resp.Body)
if err != nil { return err }
resp.Body.Close()
newBody, _ := s.playbackHandler.Handle(resp, body)
resp.Body = io.NopCloser(bytes.NewBuffer(newBody))
return nil
}

func (s *Server) isPlaybackInfoRequest(req *http.Request) bool {
if req.Method != "POST" && req.Method != "GET" { return false }
return len(req.URL.Path) > 0 && strings.Contains(req.URL.Path, "/PlaybackInfo")
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
wrapped := &responseRecorder{ResponseWriter: w, statusCode: 200}
if s.streamHandler.Handle(wrapped, r) { return }
next.ServeHTTP(wrapped, r)
})
}

type responseRecorder struct {
http.ResponseWriter
statusCode int
written    bool
}

func (rec *responseRecorder) WriteHeader(code int) {
if !rec.written { rec.statusCode = code; rec.written = true; rec.ResponseWriter.WriteHeader(code) }
}
