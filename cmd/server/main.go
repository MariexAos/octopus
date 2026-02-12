package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"octopus/internal/config"
	"octopus/internal/handler"
	"octopus/internal/model"
	"octopus/internal/mq"
	"octopus/internal/repository"
	"octopus/internal/service"
	"octopus/pkg/middleware"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Short Link Service API
// @version 1.0
// @description A short link service with analytics
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
func main() {
	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Setup logger
	setupLogger(cfg.Server.Mode)

	// Initialize repositories
	redisRepo := repository.NewRedisRepository(&cfg.Database.Redis)
	defer redisRepo.Close()

	mysqlRepo := repository.NewMySQLRepository(&cfg.Database.MySQL)
	defer mysqlRepo.Close()

	// Initialize services
	bloomSvc := service.NewBloomService(redisRepo.GetClient(), &cfg.Bloom)
	shortLinkSvc := service.NewShortLinkService(mysqlRepo, redisRepo, bloomSvc, getDomain(cfg))
	analyticsSvc := service.NewAnalyticsService(redisRepo)

	// Initialize MQ (optional, can be nil)
	var mqProducer *mq.Producer
	if cfg.RocketMQ.NameServer != "" {
		mqProducer, err = mq.NewProducer(&cfg.RocketMQ)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize RocketMQ producer, running without MQ")
		}
	}

	// Setup Gin
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()

	// Middleware
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(corsMiddleware())

	// Setup static files for 404 page
	router.LoadHTMLGlob("templates/*")

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		generateHandler := handler.NewGenerateHandler(shortLinkSvc)
		v1.POST("/shortlink/generate", generateHandler.Generate)
	}

	// Redirect handler (short codes)
	redirectHandler := handler.NewRedirectHandler(shortLinkSvc, analyticsSvc, mqProducer)
	router.GET("/:shortCode", redirectHandler.Redirect)

	// Analytics routes
	v1.GET("/analytics/:shortCode", redirectHandler.GetStats)

	// Swagger documentation
	setupSwagger(router)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Start MQ consumer if configured
	var mqConsumer *mq.Consumer
	if cfg.RocketMQ.NameServer != "" {
		// Create consumer with handler that saves to MySQL
		mqConsumer, err = mq.NewConsumer(&cfg.RocketMQ, func(ctx context.Context, msg *mq.AccessLogMessage) error {
			accessLog := &model.AccessLog{
				ShortCode:  msg.ShortCode,
				ClientIP:   msg.ClientIP,
				UserAgent:  msg.UserAgent,
				Referer:    msg.Referer,
				AccessTime: msg.AccessTime,
			}
			return mysqlRepo.SaveAccessLog(ctx, accessLog)
		})

		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize RocketMQ consumer")
		} else {
			go func() {
				if err := mqConsumer.Subscribe(); err != nil {
					log.Error().Err(err).Msg("Failed to subscribe to RocketMQ")
				}
			}()
			defer mqConsumer.Close()
		}
	}

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Info().Msgf("Starting server on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Close producer
	if mqProducer != nil {
		mqProducer.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}

// setupLogger configures the logger
func setupLogger(mode string) {
	if mode == "release" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// Use console writer for pretty output
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

// getDomain returns the domain for short links
func getDomain(cfg *config.Config) string {
	if port := cfg.Server.Port; port != 80 && port != 443 {
		return fmt.Sprintf("http://localhost:%d", port)
	}
	return "http://localhost"
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// setupSwagger sets up Swagger UI
func setupSwagger(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
