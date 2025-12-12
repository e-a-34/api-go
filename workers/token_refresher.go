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

package workers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type IntrospectionResponse struct {
	Active bool  `json:"active"`
	Exp    int64 `json:"exp,omitempty"`
	Sub    string `json:"sub,omitempty"`
	Scope  string `json:"scope,omitempty"`
}


func IntrospectToken(ctx context.Context, token string) (bool, error) {

	introspectURL := os.Getenv("KEYCLOAK_INTROSPECTION_ENDPOINT")
	clientID := os.Getenv("OIDC_CLIENT_ID")
	clientSecret := os.Getenv("OIDC_CLIENT_SECRET")

	if introspectURL == "" || clientID == "" || clientSecret == "" {
		return false, fmt.Errorf("missing KEYCLOAK_* env vars")
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("token", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, introspectURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result IntrospectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Active, nil
}

func CleanTokenWithoutTTL(ctx context.Context, rdb *redis.Client, token string, debug bool) bool {

	ttl, _ := rdb.TTL(ctx, token).Result()

	if ttl == -1 {
		if debug {
			log.Printf("⚠️  [REFRESHER] Token sans TTL → suppression immédiate\n   Token: %s\n", token)
		}
		rdb.Del(ctx, token)
		return true
	}

	return false
}

func ProcessToken(ctx context.Context, rdb *redis.Client, token string, debug bool) {

	ttl, _ := rdb.TTL(ctx, token).Result()
	ttlHuman := ttl.String()

	if CleanTokenWithoutTTL(ctx, rdb, token, debug) {
		return
	}

	active, err := IntrospectToken(ctx, token)

	if err != nil {
		if debug {
			log.Printf("❌ [REFRESHER] Erreur introspection Keycloak: %s\n   Token: %s", err, token)
		}
		rdb.Del(ctx, token)
		return
	}

	if !active {
		if debug {
			log.Printf("🔴 [REFRESHER] Token inactif (Keycloak) → suppression\n   Token: %s\n   TTL: %s\n",
				token, ttlHuman)
		}
		rdb.Del(ctx, token)
		return
	}

	if debug {
		log.Printf("🟢 [REFRESHER] Token actif\n   Token: %s\n   TTL: %s\n", token, ttlHuman)
	}
}

func StartTokenRefresher(rdb *redis.Client) {

	debug := os.Getenv("DEBUG") == "true"

	intervalSec := 15
	if v := os.Getenv("TOKEN_CHECK_INTERVAL"); v != "" {
		fmt.Sscanf(v, "%d", &intervalSec)
	}

	go func() {

		ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
		ctx := context.Background()

		for range ticker.C {

			if debug {
				log.Printf("🟦 [REFRESHER] Début du check des tokens (interval: %ds)\n", intervalSec)
			}

			keys, _ := rdb.Keys(ctx, "*").Result()

			for _, token := range keys {
				ProcessToken(ctx, rdb, token, debug)
			}

			if debug && len(keys) == 0 {
				log.Println("ℹ️  [REFRESHER] Aucun token dans Redis.")
			}
		}
	}()
}


func GetTokenExp(token string) (int64, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid JWT format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, err
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, err
	}

	return claims.Exp, nil
}