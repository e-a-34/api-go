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
	"github.com/mitchellh/mapstructure"
	"gorm.io/gorm"
)

func RegisterBuilderRoutes(group *gin.RouterGroup, db *gorm.DB) {
	builder := group.Group("/builder")

	builder.GET("", func(c *gin.Context) {
		var pages []models.Page
		var tags []models.Tag
		var templates []models.Template

		if err := db.Preload("Template").Preload("Tags.Category").Find(&pages).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_PAGES_ERROR", err.Error())
			return
		}
		if err := db.Preload("Category").Find(&tags).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_TAGS_ERROR", err.Error())
			return
		}
		if err := db.Find(&templates).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_TEMPLATES_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": pages,
			"dependencies": gin.H{
				"tags":      tags,
				"templates": templates,
				"pages": pages,
			},
			"success": true,
		})
	})

	builder.POST("", func(c *gin.Context) {
		var payload models.Page
		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}
		if err := db.Create(&payload).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_CREATE_ERROR", err.Error())
			return
		}

		var created models.Page
		if err := db.Preload("Template").Preload("Tags.Category").First(&created, "id = ?", payload.ID).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": created, "success": true})
	})

	builder.PUT("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var payload models.Page

		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}
		var existing models.Page
		if err := db.Preload("Tags").First(&existing, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "Page not found")
			return
		}

		payload.ID = id
		if err := db.Model(&existing).Omit("Tags").Updates(&payload).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_UPDATE_ERROR", err.Error())
			return
		}

		if len(payload.Tags) > 0 {
			if err := db.Model(&existing).Association("Tags").Replace(payload.Tags); err != nil {
				utils.Error(c, http.StatusInternalServerError, "DB_ASSOCIATION_ERROR", err.Error())
				return
			}
		} else {
			if err := db.Model(&existing).Association("Tags").Clear(); err != nil {
				utils.Error(c, http.StatusInternalServerError, "DB_ASSOCIATION_CLEAR_ERROR", err.Error())
				return
			}
		}

		var updated models.Page
		if err := db.Preload("Template").Preload("Tags.Category").First(&updated, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": updated, "success": true})
	})

	builder.PATCH("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var updates map[string]interface{}

		if err := c.ShouldBindJSON(&updates); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}
		if tagsRaw, ok := updates["tags"]; ok {
			delete(updates, "tags")
			var page models.Page
			if err := db.Preload("Tags").First(&page, "id = ?", id).Error; err != nil {
				utils.Error(c, http.StatusNotFound, "NOT_FOUND", "Page not found")
				return
			}
			if tags, ok := tagsRaw.([]interface{}); ok {
				tagModels := make([]models.Tag, 0, len(tags))
				for _, t := range tags {
					if tagMap, ok := t.(map[string]interface{}); ok {
						if tagID, ok := tagMap["id"].(string); ok {
							tagModels = append(tagModels, models.Tag{ID: tagID})
						}
					}
				}
				if err := db.Model(&page).Association("Tags").Replace(tagModels); err != nil {
					utils.Error(c, http.StatusInternalServerError, "DB_ASSOCIATION_ERROR", err.Error())
					return
				}
			}
		}
		if len(updates) > 0 {
			if err := db.Model(&models.Page{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				utils.Error(c, http.StatusInternalServerError, "DB_PATCH_ERROR", err.Error())
				return
			}
		}
		var updated models.Page
		if err := db.Preload("Template").Preload("Tags.Category").First(&updated, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": updated, "success": true})
	})

	builder.POST("/deleteMany", func(c *gin.Context) {
		var ids []string
		if err := c.ShouldBindJSON(&ids); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}
		if len(ids) == 0 {
			utils.Error(c, http.StatusBadRequest, "NO_IDS_PROVIDED", "No IDs provided")
			return
		}
		if err := db.Delete(&models.Page{}, ids).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_DELETE_MANY_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Pages deleted successfully", "count": len(ids), "success": true})
	})

	builder.DELETE("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var page models.Page
		if err := db.First(&page, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				utils.Error(c, http.StatusNotFound, "NOT_FOUND", "Page not found")
				return
			}
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_ERROR", err.Error())
			return
		}
		if err := db.Delete(&page).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_DELETE_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Page deleted successfully", "id": id, "success": true})
	})

	builder.PATCH("/patchMany", func(c *gin.Context) {
		var payload struct {
			IDs     []string               `json:"ids"`
			Updates map[string]interface{} `json:"updates"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}
		if len(payload.IDs) == 0 {
			utils.Error(c, http.StatusBadRequest, "NO_IDS_PROVIDED", "No IDs provided")
			return
		}
		if len(payload.Updates) == 0 {
			utils.Error(c, http.StatusBadRequest, "NO_UPDATES_PROVIDED", "No updates provided")
			return
		}
		if tagsRaw, ok := payload.Updates["tags"]; ok {
			delete(payload.Updates, "tags")
			for _, id := range payload.IDs {
				var page models.Page
				if err := db.Preload("Tags").First(&page, "id = ?", id).Error; err != nil {
					continue
				}
				if tags, ok := tagsRaw.([]interface{}); ok {
					tagModels := make([]models.Tag, 0, len(tags))
					for _, t := range tags {
						if tagMap, ok := t.(map[string]interface{}); ok {
							if tagID, ok := tagMap["id"].(string); ok {
								tagModels = append(tagModels, models.Tag{ID: tagID})
							}
						}
					}
					db.Model(&page).Association("Tags").Replace(tagModels)
				}
			}
		}
		if len(payload.Updates) > 0 {
			var updates models.Page
			if err := mapstructure.Decode(payload.Updates, &updates); err != nil {
				utils.Error(c, http.StatusBadRequest, "DECODE_ERROR", err.Error())
				return
			}
			if err := db.Model(&models.Page{}).Where("id IN ?", payload.IDs).Updates(&updates).Error; err != nil {
				utils.Error(c, http.StatusInternalServerError, "DB_PATCH_MANY_ERROR", err.Error())
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"message": "Pages updated successfully", "count": len(payload.IDs), "success": true})
	})
}
