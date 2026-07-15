package router

import (
	"path/filepath"

	"warehousecore/admin"
	adminmw "warehousecore/admin/middleware"
	"warehousecore/internal/config"
	jwtmgr "warehousecore/internal/pkg/jwt"
	"warehousecore/internal/repo"
	"warehousecore/internal/service"
	"warehousecore/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware(cfg))

	if cfg.Storage.Driver == "local" || cfg.Storage.Driver == "" {
		uploadDir := filepath.Join(cfg.Storage.LocalPath, cfg.Storage.Prefix)
		r.Static("/uploads", uploadDir)
	}

	if _, err := storage.New(&cfg.Storage); err != nil {
		panic(err)
	}

	repos := repo.New(db)
	masterSvc := service.NewMasterService(repos)
	docSvc := service.NewDocumentService(repos)
	querySvc := service.NewQueryService(repos)
	integSvc := service.NewIntegrationService(repos)
	h := admin.NewHandlers(masterSvc, docSvc, querySvc, integSvc)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "warehousecore"})
	})

	v1 := r.Group("/api/v1")
	adminGroup := v1.Group("/admin")
	jwtMgr := jwtmgr.NewManager(cfg.Auth.JWTSecret)
	adminGroup.Use(adminmw.AdminAuth(&cfg.Auth, jwtMgr))
	admin.RegisterRoutes(adminGroup, h)

	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.CORS.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin == ""
		for _, o := range origins {
			if o == origin || o == "*" {
				allowed = true
				break
			}
		}
		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
