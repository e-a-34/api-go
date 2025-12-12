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

package middlewares

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"api-core-v2/services"
	"api-core-v2/workers"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func AuthMiddleware(db *gorm.DB, verifier *oidc.IDTokenVerifier, rdb *redis.Client) gin.HandlerFunc {

	mode := strings.ToLower(os.Getenv("TOKEN_VALIDATION_MODE"))
	ctx := context.Background()

	return func(c *gin.Context) {

		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(401, gin.H{"error": "Missing Bearer token"})
			c.Abort()
			return
		}

		rawToken := strings.TrimPrefix(auth, "Bearer ")

		tokenParsed, _, err := new(jwt.Parser).ParseUnverified(rawToken, jwt.MapClaims{})
		if err != nil {
			log.Println("❌ Unable to decode JWT:", err)
			c.JSON(401, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		claims := tokenParsed.Claims.(jwt.MapClaims)

		if mode == "live" {
			if _, err := verifier.Verify(ctx, rawToken); err != nil {
				log.Println("❌ Token invalid (live mode):", err)
				c.JSON(401, gin.H{"error": "Invalid token"})
				c.Abort()
				return
			}

			services.SyncUserFromClaims(db, claims)

			c.Next()
			return
		}

		if mode == "introspection" {
			active, err := workers.IntrospectToken(ctx, rawToken)
			if err != nil || !active {
				log.Println("❌ Token rejected (introspection):", err)
				c.JSON(401, gin.H{"error": "Invalid token"})
				c.Abort()
				return
			}

			services.SyncUserFromClaims(db, claims)

			c.Next()
			return
		}
		if mode == "redis" {

			exists, _ := rdb.Exists(ctx, rawToken).Result()
			if exists == 1 {
				services.SyncUserFromClaims(db, claims)

				c.Next()
				return
			}

			active, err := workers.IntrospectToken(ctx, rawToken)
			if err != nil || !active {
				log.Println("❌ Token invalid (redis introspection):", err)
				c.JSON(401, gin.H{"error": "Invalid token"})
				c.Abort()
				return
			}

			exp, err := workers.GetTokenExp(rawToken)
			if err != nil {
				log.Println("❌ Unable to read exp claim:", err)
				c.JSON(401, gin.H{"error": "Invalid token claims"})
				c.Abort()
				return
			}
			expTime := time.Unix(exp, 0)
			ttl := time.Until(expTime)

			if ttl > 0 {
				rdb.Set(ctx, rawToken, "valid", ttl)
			}

			services.SyncUserFromClaims(db, claims)

			c.Next()
			return
		}
		log.Println("❌ Unknown TOKEN_VALIDATION_MODE:", mode)
		c.JSON(500, gin.H{"error": "Server misconfigured"})
		c.Abort()
	}
}
