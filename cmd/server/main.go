package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"sniper/util"
	"sniper/util/auth"
	"sniper/util/conf"
	"sniper/util/ctxkit"
	"sniper/util/log"
	"sniper/util/mc"
	"sniper/util/trace"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var server *http.Server

type panicHandler struct {
	handler http.Handler
}

// 从 http 标准库搬来的
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

var logger = log.Get(context.Background())

func startSpan(r *http.Request) (*http.Request, opentracing.Span) {
	operation := "ServerHTTP"

	ctx := r.Context()
	var span opentracing.Span

	tracer := opentracing.GlobalTracer()
	carrier := opentracing.HTTPHeadersCarrier(r.Header)

	if spanCtx, err := tracer.Extract(opentracing.HTTPHeaders, carrier); err == nil {
		span = opentracing.StartSpan(operation, ext.RPCServerOption(spanCtx))
		ctx = opentracing.ContextWithSpan(ctx, span)
	} else {
		span, ctx = opentracing.StartSpanFromContext(ctx, operation)
	}

	ext.SpanKindRPCServer.Set(span)
	span.SetTag(string(ext.HTTPUrl), r.URL.Path)

	return r.WithContext(ctx), span
}

func (s panicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r, span := startSpan(r)

	defer func() {
		if rec := recover(); rec != nil {
			ctx := r.Context()
			ctx = ctxkit.WithTraceID(ctx, trace.GetTraceID(ctx))
			log.Get(ctx).Error(rec, string(debug.Stack()))
		}
		span.Finish()
	}()

	if r.Method == http.MethodOptions {
		origin := r.Header.Get("Origin")
		suffix := conf.Get("CORS_ORIGIN_SUFFIX")

		if suffix != "" && strings.HasSuffix(origin, suffix) {
			w.Header().Add("Access-Control-Allow-Origin", origin)
			w.Header().Add("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			w.Header().Add("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Access-Control-Allow-Headers", "Origin,No-Cache,X-Requested-With,If-Modified-Since,Pragma,Last-Modified,Cache-Control,Expires,Content-Type,Access-Control-Allow-Credentials,DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Cache-Webcdn,Content-Length")
			return
		}
	}

	s.handler.ServeHTTP(w, r)
}

const (
	//SecretKey key
	SecretKey string = "test"
)

// UserCredentials struct
type UserCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

//Token TODO:
type Token struct {
	Token string `json:"token"`
}

func fatal(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

//JSONResponse TODO:
func JSONResponse(response interface{}, w http.ResponseWriter) {

	json, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

// LoginHandler handler
func LoginHandler(w http.ResponseWriter, r *http.Request) {

	var user UserCredentials

	err := json.NewDecoder(r.Body).Decode(&user)

	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Error in request")
		return
	}

	if user.Username != "someone" || user.Password != "p@ssword" {
		// if user.Password != "p@ssword" {
		w.WriteHeader(http.StatusForbidden)
		fmt.Println("Error logging in")
		fmt.Fprint(w, "Invalid credentials")
		return
		// }
	}

	var response Token
	ctx := r.Context()
	c := mc.Get(ctx, "DEFAULT")
	item, err := c.Get(ctx, user.Username)
	fmt.Printf("item = %v", item)

	if mc.IsCacheMiss(err) {
		tokenString, err := auth.Generate(user.Username, time.Now(), conf.GetDuration("JWT_TOKEN_DURATION"))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Error extracting the key")
			fatal(err)
		}
		itemAdd := &mc.Item{Key: user.Username, Value: []byte(tokenString), Expiration: conf.GetInt32("JWT_TOKEN_EXPIRATION")}
		err = c.Add(ctx, itemAdd)
		if !mc.IsNotStored(err) {
			response = Token{tokenString}
		}
	} else {
		response = Token{string(item.Value)}
	}

	// token := jwt.New(jwt.SigningMethodHS256)
	// claims := make(jwt.MapClaims)
	// claims["exp"] = time.Now().Add(time.Hour * time.Duration(1)).Unix()
	// claims["iat"] = time.Now().Unix()
	// token.Claims = claims

	// tokenString, err := token.SignedString([]byte(SecretKey))
	// if err != nil {
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	fmt.Fprintln(w, "Error while signing the token")
	// 	fatal(err)
	// }

	JSONResponse(response, w)

}

func main() {
	reload := make(chan int, 1)
	stop := make(chan os.Signal, 1)

	conf.OnConfigChange(func() { reload <- 1 })
	conf.WatchConfig()
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	startServer()

	for {
		select {
		case <-reload:
			util.Reset()
		case sg := <-stop:
			stopServer()
			// 仿 nginx 使用 HUP 信号重载配置
			if sg == syscall.SIGHUP {
				startServer()
			} else {
				util.Stop()
				return
			}
		}
	}
}

func startServer() {
	logger.Info("start server")

	rand.Seed(int64(time.Now().Nanosecond()))

	mux := http.NewServeMux()

	timeout := 600 * time.Millisecond
	initMux(mux, isInternal)

	if isInternal {
		initInternalMux(mux)

		if d := conf.GetDuration("INTERNAL_API_TIMEOUT"); d > 0 {
			timeout = d
		}
	} else {
		if d := conf.GetDuration("OUTER_API_TIMEOUT"); d > 0 {
			timeout = d
		}
	}

	handler := http.TimeoutHandler(panicHandler{handler: mux}, timeout, "timeout")

	http.Handle("/", handler)

	http.HandleFunc("/login", LoginHandler)

	metricsHandler := promhttp.Handler()
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		util.GatherMetrics()

		metricsHandler.ServeHTTP(w, r)
	})

	http.HandleFunc("/monitor/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	addr := fmt.Sprintf(":%d", port)
	server = &http.Server{
		IdleTimeout: 60 * time.Second,
	}

	// 配置下发可能会多次触发重启，必须等待 Listen() 调用成功
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		// 本段代码基本搬自 http 标准库
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			panic(err)
		}
		wg.Done()

		err = server.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
		if err != http.ErrServerClosed {
			panic(err)
		}
	}()

	wg.Wait()
}

func stopServer() {
	logger.Info("stop server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal(err)
	}

	util.Reset()
}
