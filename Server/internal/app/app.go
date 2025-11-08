package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	authinterface "github.com/inzarubin80/Server/internal/app/authinterface"
	appHttp "github.com/inzarubin80/Server/internal/app/http"
	middleware "github.com/inzarubin80/Server/internal/app/http/middleware"
	ws "github.com/inzarubin80/Server/internal/app/ws"
	service "github.com/inzarubin80/Server/internal/service"

	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/cors"
	"golang.org/x/oauth2"
)

const (
	readHeaderTimeoutSeconds = 3
)

type (
	mux interface {
		Handle(pattern string, handler http.Handler)
	}
	server interface {
		ListenAndServe() error
		Close() error
	}

	App struct {
		mux           mux
		server        server
		pokerService  *service.PokerService
		config        config
		hub           *ws.Hub
		oauthConfig   *oauth2.Config
		store         *sessions.CookieStore
		provadersConf authinterface.MapProviderOauthConf
	}
)

func (a *App) ListenAndServe() error {
	go a.hub.Run()

	a.mux.Handle(a.config.path.ping, appHttp.NewPingHandlerHandler(a.config.path.ping))
	a.mux.Handle(a.config.path.session, appHttp.NewGetSessionHandler(a.store, a.config.path.session))
	a.mux.Handle(a.config.path.getProviders, appHttp.NewProvadersHandler(a.provadersConf, a.config.path.getProviders))
	a.mux.Handle(a.config.path.login, appHttp.NewLoginHandler(a.provadersConf, a.config.path.login, a.store))	
	fmt.Println("start server")
	return a.server.ListenAndServe()
}

func NewApp(ctx context.Context, config config, dbConn *pgxpool.Pool) (*App, error) {

	var (
		mux   = http.NewServeMux()
		hub   = ws.NewHub()
		store = sessions.NewCookieStore([]byte(config.sectrets.storeSecret))
	)

	// Do not instantiate full pokerService in skeleton; leave nil to avoid linking missing implementations.
	var pokerService *service.PokerService = nil

	// Создаем CORS middleware
	corsMiddleware := cors.New(cors.Options{
		// Явно разрешаем оба домена (без точки в начале)
		AllowedOrigins: []string{
			"http://localhost:3000",
		},
		// Добавляем все необходимые методы
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		// Разрешаем все стандартные заголовки + кастомные
		AllowedHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization",
			"X-Requested-With", "X-CSRF-Token", "Custom-Header",
		},
		// Разрешаем куки и авторизацию
		AllowCredentials: true,
		// Опционально: максимальное время кеширования preflight-запросов
		MaxAge: 86400,
	})

	// Обертываем основной обработчик
	handler := corsMiddleware.Handler(middleware.NewLogMux(mux))

	return &App{
		mux:           mux,
		server:        &http.Server{Addr: config.addr, Handler: handler, ReadHeaderTimeout: readHeaderTimeoutSeconds * time.Second},
		pokerService:  pokerService,
		config:        config,
		hub:           hub,
		store:         store,
		provadersConf: config.provadersConf,
	}, nil

}
