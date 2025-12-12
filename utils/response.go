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

package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message,omitempty"`
    Data    interface{} `json:"data,omitempty"`
    Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
    Code    string `json:"code,omitempty"`
    Details string `json:"details,omitempty"`
}

func JSON(c *gin.Context, status int, message string, data interface{}) {
    c.JSON(status, APIResponse{
        Success: true,
        Message: message,
        Data:    data,
    })
}

func Error(c *gin.Context, status int, code string, details string) {
    c.JSON(status, APIResponse{
        Success: false,
        Error: &APIError{
            Code:    code,
            Details: details,
        },
    })
}

func HealthResponse(c *gin.Context) {
    JSON(c, http.StatusOK, "API-Core opérationnelle 🚀", gin.H{
        "service":   "api-core",
        "status":    "online",
        "timestamp": time.Now().UTC(),
    })
}
