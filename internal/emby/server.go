package emby

import (
	"bytes"
	"context"
	"fntv-proxy/internal/cache"
	"fntv-proxy/internal/config"
	"fntv-proxy/internal/logger"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Server Emby 302 代理服务器
type Server struct {
	config          *config.Config
	logger          *logger.Logger
	cache           *cache.Cache
	playbackHandler *PlaybackHandler
	streamHandler   *StreamHandler
	proxy           *httputil.ReverseProxy
	targetURL       *url.URL
	httpServer      *http.Server
}

// NewServer 创建 Emby 代理服务器
func NewServer(cfg *config.Config) (*Server, error) {
	log := logger.New(cfg.GetLogLevel(), cfg.LogDir)

	targetURL, err := url.Parse(cfg.Emby.GetTargetAddr())
	if err != nil {
		return nil, err
	}

	ttl := cfg.Emby.GetCacheTTL(cfg.GetCacheTTL())
	c := cache.NewWithStreamTTL(ttl)

	ph := NewPlaybackHandler(c, log, &cfg.Emby)
	sh := NewStreamHandler(c, log, &cfg.Emby, targetURL)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	return &Server{
		config:          cfg,
		logger:          log,
		cache:           c,
		playbackHandler: ph,
		streamHandler:   sh,
		proxy:           proxy,
		targetURL:       targetURL,
	}, nil
}

// Start 启动 Emby 代理
func (s *Server) Start() error {
	originalDirector := s.proxy.Director
	s.proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = s.targetURL.Host
	}
	s.proxy.ModifyResponse = s.handleResponse

	s.httpServer = &http.Server{
		Addr:    s.config.Emby.GetListenAddr(),
		Handler: s.loggingMiddleware(s.proxy),
	}

	s.logger.Info("🚀 [Emby] 代理启动: %s -> %s", s.config.Emby.GetListenAddr(), s.config.Emby.GetTargetAddr())
	return s.httpServer.ListenAndServe()
}

// Stop 优雅关闭
func (s *Server) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleResponse(resp *http.Response) error {
	if !isPlaybackInfoRequest(resp.Request) {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	newBody, modified, err := s.playbackHandler.Handle(resp, body)
	if err != nil {
		s.logger.Error("[Emby] 处理 PlaybackInfo 失败: %v", err)
		newBody = body
	}

	if modified {
		resp.Header.Del("Content-Encoding")
		resp.Header.Del("Content-Length")
		resp.ContentLength = int64(len(newBody))
		resp.Header.Set("Content-Length", strconv.Itoa(len(newBody)))
	}

	resp.Body = io.NopCloser(bytes.NewBuffer(newBody))
	return nil
}

func isPlaybackInfoRequest(req *http.Request) bool {
	if req.Method != http.MethodPost && req.Method != http.MethodGet {
		return false
	}
	return strings.Contains(strings.ToLower(req.URL.Path), "/playbackinfo")
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logger.Debug("[Emby] 请求: %s %s", r.Method, r.URL.Path)

		if s.streamHandler.Handle(w, r) {
			return
		}

		next.ServeHTTP(w, r)
	})
}
