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

package routes

import (
	"api-core-v2/models"
	"api-core-v2/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterUserRoutes(group *gin.RouterGroup, db *gorm.DB) {
	users := group.Group("/users")
	users.GET("", func(c *gin.Context) {
		var users []models.User
		var tags []models.Tag

		if err := db.Preload("Tags.Category").Find(&users).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_USERS_ERROR", err.Error())
			return
		}

		if err := db.Preload("Category").Find(&tags).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_TAGS_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": users,
			"dependencies": gin.H{
				"tags": tags,
			},
			"success": true,
		})
	})

	users.POST("", func(c *gin.Context) {
		var payload models.User

		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}

		if err := db.Create(&payload).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_CREATE_ERROR", err.Error())
			return
		}

		var created models.User
		if err := db.Preload("Tags.Category").First(&created, "id = ?", payload.ID).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"data":    created,
			"success": true,
		})
	})

	users.PUT("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var payload models.User

		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}

		var existing models.User
		if err := db.Preload("Tags").First(&existing, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}

		payload.ID = id

		if err := db.Model(&existing).Omit("Tags").Updates(&payload).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_UPDATE_ERROR", err.Error())
			return
		}

		if payload.Tags != nil {
			if len(payload.Tags) > 0 {
				ids := make([]string, len(payload.Tags))
				for i, t := range payload.Tags {
					ids[i] = t.ID
				}

				var tags []models.Tag
				if err := db.Find(&tags, "id IN ?", ids).Error; err != nil {
					utils.Error(c, http.StatusInternalServerError, "DB_TAG_FETCH_ERROR", err.Error())
					return
				}

				if err := db.Model(&existing).Association("Tags").Replace(tags); err != nil {
					utils.Error(c, http.StatusInternalServerError, "DB_ASSOCIATION_ERROR", err.Error())
					return
				}
			} else {
				if err := db.Model(&existing).Association("Tags").Clear(); err != nil {
					utils.Error(c, http.StatusInternalServerError, "DB_ASSOCIATION_CLEAR_ERROR", err.Error())
					return
				}
			}
		}

		var updated models.User
		if err := db.Preload("Tags.Category").First(&updated, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":    updated,
			"success": true,
		})
	})
	users.PATCH("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var payload models.User

		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}

		var existing models.User
		if err := db.Preload("Tags").First(&existing, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}

		payload.ID = id

		if err := db.Model(&existing).Omit("Tags").Updates(&payload).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_UPDATE_ERROR", err.Error())
			return
		}

		if payload.Tags != nil {
			if len(payload.Tags) > 0 {
				ids := make([]string, len(payload.Tags))
				for i, t := range payload.Tags {
					ids[i] = t.ID
				}

				var tags []models.Tag
				if err := db.Find(&tags, "id IN ?", ids).Error; err != nil {
					utils.Error(c, http.StatusInternalServerError, "DB_TAG_FETCH_ERROR", err.Error())
					return
				}

				if err := db.Model(&existing).Association("Tags").Replace(tags); err != nil {
					utils.Error(c, http.StatusInternalServerError, "DB_ASSOCIATION_ERROR", err.Error())
					return
				}
			} else {
				if err := db.Model(&existing).Association("Tags").Clear(); err != nil {
					utils.Error(c, http.StatusInternalServerError, "DB_ASSOCIATION_CLEAR_ERROR", err.Error())
					return
				}
			}
		}

		var updated models.User
		if err := db.Preload("Tags.Category").First(&updated, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":    updated,
			"success": true,
		})
	})

	users.PATCH("/patchMany", func(c *gin.Context) {
		var payload struct {
			IDs     []string    `json:"ids"`
			Updates models.User `json:"updates"`
		}

		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}

		if len(payload.IDs) == 0 {
			utils.Error(c, http.StatusBadRequest, "NO_IDS_PROVIDED", "No IDs provided")
			return
		}

		if payload.Updates.Tags != nil {
			for _, id := range payload.IDs {
				var user models.User
				if err := db.Preload("Tags").First(&user, "id = ?", id).Error; err != nil {
					continue
				}

				if len(payload.Updates.Tags) > 0 {
					ids := make([]string, len(payload.Updates.Tags))
					for i, t := range payload.Updates.Tags {
						ids[i] = t.ID
					}

					var tags []models.Tag
					if err := db.Find(&tags, "id IN ?", ids).Error; err == nil {
						db.Model(&user).Association("Tags").Replace(tags)
					}
				} else {
					db.Model(&user).Association("Tags").Clear()
				}
			}
		}

		if err := db.Model(&models.User{}).
			Where("id IN ?", payload.IDs).
			Omit("Tags").
			Updates(&payload.Updates).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_PATCH_MANY_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Users updated successfully",
			"count":   len(payload.IDs),
			"success": true,
		})
	})
	users.POST("/deleteMany", func(c *gin.Context) {
		var ids []string

		if err := c.ShouldBindJSON(&ids); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}

		if len(ids) == 0 {
			utils.Error(c, http.StatusBadRequest, "NO_IDS_PROVIDED", "No IDs provided")
			return
		}

		if err := db.Delete(&models.User{}, ids).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_DELETE_MANY_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Users deleted successfully",
			"count":   len(ids),
			"success": true,
		})
	})

	users.DELETE("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var user models.User

		if err := db.First(&user, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				utils.Error(c, http.StatusNotFound, "NOT_FOUND", "User not found")
				return
			}
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_ERROR", err.Error())
			return
		}

		if err := db.Delete(&user).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_DELETE_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "User deleted successfully",
			"id":      id,
			"success": true,
		})
	})
}
