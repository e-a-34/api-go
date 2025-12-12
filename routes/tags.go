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

func RegisterTagRoutes(group *gin.RouterGroup, db *gorm.DB) {
	tags := group.Group("/tags")
	tags.GET("", func(c *gin.Context) {
		var tags []models.Tag
		var categories []models.TagCategory

		if err := db.Preload("Category").Find(&tags).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_TAGS_ERROR", err.Error())
			return
		}
		if err := db.Find(&categories).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_CATEGORIES_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": tags,
			"dependencies": gin.H{
				"categories": categories,
			},
			"success": true,
		})
	})
	tags.POST("", func(c *gin.Context) {
		var payload models.Tag
		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}
		if err := db.Create(&payload).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_CREATE_ERROR", err.Error())
			return
		}

		var created models.Tag
		if err := db.Preload("Category").First(&created, "id = ?", payload.ID).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": created, "success": true})
	})
	tags.PUT("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var payload models.Tag

		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}

		var existing models.Tag
		if err := db.Preload("Category").First(&existing, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "Tag not found")
			return
		}

		payload.ID = id
		if err := db.Model(&existing).Updates(&payload).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_UPDATE_ERROR", err.Error())
			return
		}

		var updated models.Tag
		if err := db.Preload("Category").First(&updated, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": updated, "success": true})
	})
	tags.PATCH("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var updates map[string]interface{}

		if err := c.ShouldBindJSON(&updates); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}

		if len(updates) == 0 {
			utils.Error(c, http.StatusBadRequest, "NO_UPDATES_PROVIDED", "No updates provided")
			return
		}

		if err := db.Model(&models.Tag{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_PATCH_ERROR", err.Error())
			return
		}

		var updated models.Tag
		if err := db.Preload("Category").First(&updated, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": updated, "success": true})
	})
	tags.PATCH("/patchMany", func(c *gin.Context) {
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

		var updates models.Tag
		if err := mapstructure.Decode(payload.Updates, &updates); err != nil {
			utils.Error(c, http.StatusBadRequest, "DECODE_ERROR", err.Error())
			return
		}

		if err := db.Model(&models.Tag{}).Where("id IN ?", payload.IDs).Updates(&updates).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_PATCH_MANY_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Tags updated successfully",
			"count":   len(payload.IDs),
			"success": true,
		})
	})
	tags.POST("/deleteMany", func(c *gin.Context) {
		var ids []string
		if err := c.ShouldBindJSON(&ids); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}
		if len(ids) == 0 {
			utils.Error(c, http.StatusBadRequest, "NO_IDS_PROVIDED", "No IDs provided")
			return
		}
		if err := db.Delete(&models.Tag{}, ids).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_DELETE_MANY_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Tags deleted successfully",
			"count":   len(ids),
			"success": true,
		})
	})

	tags.DELETE("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var tag models.Tag

		if err := db.First(&tag, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				utils.Error(c, http.StatusNotFound, "NOT_FOUND", "Tag not found")
				return
			}
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_ERROR", err.Error())
			return
		}

		if err := db.Delete(&tag).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_DELETE_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Tag deleted successfully",
			"id":      id,
			"success": true,
		})
	})
}
