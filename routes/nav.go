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
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Bool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func RegisterNavigationRoutes(r *gin.RouterGroup, db *gorm.DB) {
	n := r.Group("/navigation")

	n.GET("", func(c *gin.Context) {
		var items []models.NavigationItem
		if err := db.Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		tree := map[string]*models.NavItem{}
		children := map[string][]*models.NavItem{}

		for _, item := range items {
			node := &models.NavItem{
				Title:     item.Title,
				Path:      item.Path,
				Icon:      item.Icon,
				Caption:   item.Caption,
				Disabled:  Bool(item.Disabled),
				DeepMatch: Bool(item.DeepMatch),
			}

			tree[item.ID] = node

			if item.ParentID != nil {
				children[*item.ParentID] = append(children[*item.ParentID], node)
			}
		}

		for id, node := range tree {
			for _, child := range children[id] {
				node.Children = append(node.Children, *child)
			}
		}

		var navSections []models.NavSection
		for _, item := range items {
			if item.IsHeader != nil && *item.IsHeader {
				section := models.NavSection{
					Subheader: item.Title,
					Items:     []models.NavItem{},
				}

				for _, child := range children[item.ID] {
					section.Items = append(section.Items, *child)
				}

				navSections = append(navSections, section)
			}
		}

		c.JSON(http.StatusOK, navSections)
	})


	n.POST("", func(c *gin.Context) {
		var input models.NavigationItem
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tx := db.Begin()

		if input.ParentID != nil {
			var parent models.NavigationItem
			if err := tx.First(&parent, "id = ?", *input.ParentID).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Parent not found"})
				return
			}

			if err := tx.Model(&models.NavigationItem{}).
				Where("rgt >= ?", parent.Rgt).
				Update("rgt", gorm.Expr("rgt + 2")).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			if err := tx.Model(&models.NavigationItem{}).
				Where("lft > ?", parent.Rgt).
				Update("lft", gorm.Expr("lft + 2")).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			input.Lft = parent.Rgt
			input.Rgt = parent.Rgt + 1
			input.Depth = parent.Depth + 1

		} else {
			var maxRgt sql.NullInt64
			if err := tx.Model(&models.NavigationItem{}).Select("MAX(rgt)").Scan(&maxRgt).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		tx.Commit()
		c.JSON(http.StatusCreated, input)
	})

	n.DELETE("/:id", func(c *gin.Context) {
		if err := db.Delete(&models.NavigationItem{}, "id = ?", c.Param("id")).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})
}
