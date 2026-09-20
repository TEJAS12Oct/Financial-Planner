package main

import (
	"FinancialPlannerGo/app/middleware"
	"FinancialPlannerGo/app/routes"
	"FinancialPlannerGo/config"
	envconfig "FinancialPlannerGo/config/env_config"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// @title BoltPe Financial Planner API
// @version 1.1.0
// @description BoltPe APIs for generating normalized, versioned financial-planning JSON and retrieving stored report metadata.
// @description The report-json endpoint uses the BoltPe-owned Planning Engine with no Policy Planner runtime dependency.
// // @description All protected endpoints require the X-API-Key header.
// @BasePath /
// @schemes http https
// @tag.name Financial Planner Reports
// @tag.description Generate normalized reports and retrieve stored report metadata.
// @tag.name Pages
// @tag.description Application pages and health endpoints.
// @tag.name Email
// @tag.description Email-service APIs.
// // @securityDefinitions.apikey ApiKeyAuth
// // @in header
// // @name X-API-Key
// // @description Enter the configured BoltPe API key.
//
//go:generate swag init -g main.go -o docs --parseInternal
func main() {
	if err := config.Init(); err != nil {
		log.Fatal("configuration initialization failed: ", err)
	}

	db := config.ConnectToDB()

	if err := config.RunDatabaseMigrations(db); err != nil {
		log.Fatal("database migration failed: ", err)
	}

	if envconfig.AppConfig.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	trustedProxies := splitCSV(
		envconfig.AppConfig.Network.TrustedProxies,
	)

	if err := router.SetTrustedProxies(trustedProxies); err != nil {
		log.Fatal(
			"invalid TRUSTED_PROXIES configuration: ",
			err,
		)
	}

	router.Use(
		gin.Logger(),
		middleware.RecoveryJSON(),
		middleware.SecurityHeaders(),
		middleware.LimitRequestBody(),
	)

	routes.RegisterFinancialPlannerRoutes(router)

	port := envconfig.AppConfig.App.Port

	log.Println(
		"Financial Planner service running on port:",
		port,
	)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       180 * time.Second,
		WriteTimeout:      180 * time.Second,
		IdleTimeout:       180 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		log.Fatal("server start failed: ", err)
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")

	result := make([]string, 0, len(parts))

	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
