/*
 * Copyright (c) 2025 Enzo Amate
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"api-core-v2/middlewares"
	"api-core-v2/models"
	"api-core-v2/routes"
	"api-core-v2/services"
	"api-core-v2/workers"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
)

var db *gorm.DB
var rdb *redis.Client
var ctx = context.Background()

func main() {
	_ = godotenv.Load()
	debugMode := os.Getenv("DEBUG") == "true"

	if debugMode {
		gin.SetMode(gin.DebugMode)
		log.Println("🟢 Debug mode ON")
	} else {
		gin.SetMode(gin.ReleaseMode)
		log.Println("🔵 Debug mode OFF")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("❌ DATABASE_URL manquant")
	}

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Impossible de se connecter à Postgres: %v", err)
	}
	log.Println("✅ Connecté à Postgres")

	if err := models.AutoMigrateAll(db); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}
	log.Println("📦 Migrations OK")
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		log.Fatal("❌ REDIS_URL manquant")
	}

	rdb = redis.NewClient(&redis.Options{Addr: redisAddr, DB: 0})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Redis KO: %v", err)
	}
	log.Println("✅ Connecté à Redis")

	oidcService := services.InitOIDC()
	verifier := oidcService.Verifier

	if os.Getenv("TOKEN_VALIDATION_MODE") == "redis" {
		log.Println("🔵 Token validation mode: redis")
		workers.StartTokenRefresher(rdb)
	} else {
		log.Println("🔵 Token validation mode: live")
	}

	allowedOrigins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")
	r := gin.Default()

	if debugMode {
		r.Use(middlewares.DebugLogger())
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")
	api.Use(
		middlewares.AuthMiddleware(db, verifier, rdb),
	)
	routes.RegisterNavRoutes(api, db)
	routes.RegisterNavigationRoutes(api, db)
	routes.RegisterPublicPageItemRoutes(api, db)
	routes.RegisterUserRoutes(api, db)
	routes.RegisterPublicPageRoutes(api, db)
	routes.RegisterTagRoutes(api, db)
	routes.RegisterBuilderRoutes(api, db)
	routes.RegisterTagCategoryRoutes(api, db)
	r.Run(":8080")
}
