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
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterNavRoutes(group *gin.RouterGroup, db *gorm.DB) {
	navigation := group.Group("/nav")
	navigation.GET("", func(c *gin.Context) {
		var items []models.NavigationItem
		var pages []models.Page
		var tags []models.Tag
		var navDeps []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}

		if err := db.Preload("Parent").
			Preload("Page").
			Preload("Tags.Category").
			Order("lft ASC").
			Find(&items).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_NAVIGATION_ERROR", err.Error())
			return
		}
		if err := db.Model(&models.NavigationItem{}).
			Select("id", "title").
			Order("lft ASC").
			Find(&navDeps).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_NAV_DEPS_ERROR", err.Error())
			return
		}

		if err := db.Find(&pages).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_PAGES_ERROR", err.Error())
			return
		}

		if err := db.Preload("Category").Find(&tags).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_TAGS_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": items,
			"dependencies": gin.H{
				"navigation": navDeps,
				"pages":      pages,
				"tags":       tags,
			},
			"success": true,
		})
	})


	navigation.POST("", func(c *gin.Context) {
		var input models.NavigationItem
		if err := c.ShouldBindJSON(&input); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}

		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		if input.ParentID != nil {
			var parent models.NavigationItem
			if err := tx.First(&parent, "id = ?", *input.ParentID).Error; err != nil {
				tx.Rollback()
				utils.Error(c, http.StatusBadRequest, "PARENT_NOT_FOUND", "Parent not found")
				return
			}
			if err := tx.Model(&models.NavigationItem{}).
				Where("rgt >= ?", parent.Rgt).
				Update("rgt", gorm.Expr("rgt + 2")).Error; err != nil {
				tx.Rollback()
				utils.Error(c, http.StatusInternalServerError, "UPDATE_RGT_ERROR", err.Error())
				return
			}
			if err := tx.Model(&models.NavigationItem{}).
				Where("lft > ?", parent.Rgt).
				Update("lft", gorm.Expr("lft + 2")).Error; err != nil {
				tx.Rollback()
				utils.Error(c, http.StatusInternalServerError, "UPDATE_LFT_ERROR", err.Error())
				return
			}
			input.Lft = parent.Rgt
			input.Rgt = parent.Rgt + 1
			input.Depth = parent.Depth + 1

		} else {
			var maxRgt sql.NullInt64
			if err := tx.Model(&models.NavigationItem{}).Select("MAX(rgt)").Scan(&maxRgt).Error; err != nil {
				tx.Rollback()
				utils.Error(c, http.StatusInternalServerError, "MAX_RGT_ERROR", err.Error())
				return
			}
			start := 1
			if maxRgt.Valid {
				start = int(maxRgt.Int64) + 1
			}
			input.Lft = start
			input.Rgt = start + 1
			input.Depth = 0
		}
		if err := tx.Create(&input).Error; err != nil {
			tx.Rollback()
			utils.Error(c, http.StatusInternalServerError, "DB_CREATE_ERROR", err.Error())
			return
		}
		tx.Commit()
		var created models.NavigationItem
		if err := db.Preload("Parent").
			Preload("Page").
			Preload("Tags.Category").
			First(&created, "id = ?", input.ID).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"data":    created,
			"success": true,
		})
	})

	navigation.PUT("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var payload models.NavigationItem

		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}

		var existing models.NavigationItem
		if err := db.Preload("Tags").First(&existing, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "Navigation item not found")
			return
		}

		payload.ID = id

		if err := db.Model(&existing).Omit("Tags").Updates(&payload).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_UPDATE_ERROR", err.Error())
			return
		}

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

		var updated models.NavigationItem
		if err := db.Preload("Parent").
			Preload("Page").
			Preload("Tags.Category").
			First(&updated, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": updated, "success": true})
	})

	navigation.PATCH("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var payload models.NavigationItem

		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}

		var existing models.NavigationItem
		if err := db.Preload("Tags").First(&existing, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusNotFound, "NOT_FOUND", "Navigation item not found")
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

		var updated models.NavigationItem
		if err := db.Preload("Parent").
			Preload("Page").
			Preload("Tags.Category").
			First(&updated, "id = ?", id).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_RELOAD_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": updated, "success": true})
	})

	navigation.PATCH("/patchMany", func(c *gin.Context) {
		var payload struct {
			IDs     []string              `json:"ids"`
			Updates models.NavigationItem `json:"updates"`
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
				var item models.NavigationItem
				if err := db.Preload("Tags").First(&item, "id = ?", id).Error; err != nil {
					continue
				}

				if len(payload.Updates.Tags) > 0 {
					ids := make([]string, len(payload.Updates.Tags))
					for i, t := range payload.Updates.Tags {
						ids[i] = t.ID
					}
					var tags []models.Tag
					if err := db.Find(&tags, "id IN ?", ids).Error; err == nil {
						db.Model(&item).Association("Tags").Replace(tags)
					}
				} else {
					db.Model(&item).Association("Tags").Clear()
				}
			}
		}

		if err := db.Model(&models.NavigationItem{}).
			Where("id IN ?", payload.IDs).
			Omit("Tags").
			Updates(&payload.Updates).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_PATCH_MANY_ERROR", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Navigation items updated successfully",
			"count":   len(payload.IDs),
			"success": true,
		})
	})

	navigation.POST("/deleteMany", func(c *gin.Context) {
		var ids []string
		if err := c.ShouldBindJSON(&ids); err != nil {
			utils.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
			return
		}
		if len(ids) == 0 {
			utils.Error(c, http.StatusBadRequest, "NO_IDS_PROVIDED", "No IDs provided")
			return
		}
		if err := db.Delete(&models.NavigationItem{}, ids).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_DELETE_MANY_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Navigation items deleted successfully", "count": len(ids), "success": true})
	})

	navigation.DELETE("/:id", func(c *gin.Context) {
		id := c.Param("id")
		var item models.NavigationItem
		if err := db.First(&item, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				utils.Error(c, http.StatusNotFound, "NOT_FOUND", "Navigation item not found")
				return
			}
			utils.Error(c, http.StatusInternalServerError, "DB_FETCH_ERROR", err.Error())
			return
		}
		if err := db.Delete(&item).Error; err != nil {
			utils.Error(c, http.StatusInternalServerError, "DB_DELETE_ERROR", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Navigation item deleted successfully", "id": id, "success": true})
	})
}
