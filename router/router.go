package router

import (
	"bank/intern/http"
	auth "bank/intern/middleware"
	"bank/intern/repository"
	"bank/intern/service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	Pool       *pgxpool.Pool
	JWTSecret  string
	ExpireTime time.Duration
}

func NewRouter(config Config) *gin.Engine {
	repo := repository.CreateRepo(config.Pool)
	serv := service.CreateService(repo)
	jwtServ := service.NewJWTService(config.JWTSecret, config.ExpireTime)
	authServ := service.NewAuthService(repo, jwtServ)
	handlers := http.CreateHandlers(serv, authServ)
	router := gin.Default()
	router.StaticFile("/login.html", "./dist/login.html")
	router.StaticFile("/style.css", "./dist/style.css")
	router.Static("/assets", "./dist/assets")
	router.StaticFile("/", "./dist/index.html")
	router.NoRoute(func(c *gin.Context) {
		c.File("./dist/index.html")
	})
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.POST("/api/login", handlers.Login)         // логин +
	router.POST("/api/register", handlers.CreateUser) // рега +

	authorized := router.Group("/api")
	authorized.Use(auth.MetricsMiddleware())

	authorized.Use(auth.AuthMiddleware(jwtServ))

	authorized.GET("/cards/active", handlers.GetActiveCards)   // активные номера юзера
	authorized.GET("/cards/expired", handlers.GetExpiredCards) // история номеров юзера по айди
	authorized.POST("/change-password", handlers.UpdatePass)   // обновить пароль +

	admin := router.Group("/api/admin")
	admin.Use(auth.AuthMiddleware(jwtServ))
	admin.Use(auth.AdminMiddleware())

	admin.POST("/cards", handlers.CreateCard)    // создать карту +
	admin.POST("/cards/rent", handlers.RentCard) // арендовать +
	return router
}
