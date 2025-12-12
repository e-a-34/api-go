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
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func DebugLogger() gin.HandlerFunc {
	return func(c *gin.Context) {

		log.Println("────────────────────────────────────────")
		log.Printf("➡️  %s %s", c.Request.Method, c.Request.URL.Path)

		log.Println("📌 Headers:")
		for k, v := range c.Request.Header {

			value := strings.Join(v, ", ")
			if strings.ToLower(k) == "cookie" {
				value = maskCookies(value)
			}

			log.Printf("   %s: %s", k, value)
		}

		if c.Request.Method == http.MethodPost ||
			c.Request.Method == http.MethodPut ||
			c.Request.Method == http.MethodPatch {

			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

			if len(bodyBytes) > 0 {
				log.Println("📦 Body:")
				log.Println(string(bodyBytes))
			}
		}
		c.Next()
		log.Printf("⬅️  Response status: %d", c.Writer.Status())
	}
}

func maskCookies(raw string) string {
	cookies := strings.Split(raw, "; ")
	masked := []string{}

	sensitive := []string{
		"authjs.session-token",
		"authjs.csrf-token",
		"pga4_session",
		"session",
	}

	for _, c := range cookies {
		for _, s := range sensitive {
			if strings.HasPrefix(strings.ToLower(c), strings.ToLower(s)+"=") {
				masked = append(masked, s+"=****MASKED****")
				goto nextCookie
			}
		}
		masked = append(masked, c)

	nextCookie:
	}

	return strings.Join(masked, "; ")
}
