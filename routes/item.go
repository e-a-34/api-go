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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterPublicPageItemRoutes(r gin.IRoutes, db *gorm.DB) {

	r.GET("/page/:id/:itemId", func(c *gin.Context) {
		pageID := c.Param("id")
		itemID := c.Param("itemId")

		var page models.Page
		if err := db.
			Preload("Template").
			Preload("FicheTemplate").
			First(&page, "id = ?", pageID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Page introuvable"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var raw schemaRaw
		if page.SchemaRelationsDeployed != nil {
			_ = json.Unmarshal(page.SchemaRelationsDeployed, &raw.Relations)
		}
		if page.SchemaUiDeployed != nil {
			_ = json.Unmarshal(page.SchemaUiDeployed, &raw.UI)
		}

		if !Bool(page.Deploy) || page.TableName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cette page ne contient pas de table déployée"})
			return
		}

		sqlDB, _ := db.DB()
		query := fmt.Sprintf(`SELECT * FROM %s WHERE id = $1`, quoteIdent(page.TableName))
		row := sqlDB.QueryRow(query, itemID)

		cols, _ := getColumns(sqlDB, page.TableName)
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range cols {
			ptrs[i] = &values[i]
		}

		if err := row.Scan(ptrs...); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Item introuvable"})
			return
		}

		item := make(map[string]any)
		for i, col := range cols {
			item[col] = values[i]
		}

		fkByTable := make(map[string]map[string]struct{})
		for _, rel := range raw.Relations {
			if rel.Type == "one-to-one" || rel.Type == "one-to-many" {
				if fk, ok := item[rel.FromColumn]; ok && fk != nil {
					idStr := fmt.Sprintf("%v", fk)
					addFK(fkByTable, rel.ToTable, idStr)
				}
			}
		}
		pivotData := make(map[string][]string)
		for _, rel := range raw.Relations {
			if rel.Type != "many-to-many" {
				continue
			}
			pivot := pivotTableName(page.TableName, rel)

			q := fmt.Sprintf(`SELECT right_id FROM %s WHERE left_id = $1`, quoteIdent(pivot))
			rs, err := sqlDB.Query(q, itemID)
			if err != nil {
				continue
			}
			var rid string
			for rs.Next() {
				rs.Scan(&rid)
				pivotData[pivot] = append(pivotData[pivot], rid)
				addFK(fkByTable, rel.ToTable, rid)
			}
			rs.Close()
		}

		objCache := batchLoadRelated(sqlDB, fkByTable)
		for _, rel := range raw.Relations {
			switch rel.Type {
			case "one-to-one", "one-to-many":
				if fk, ok := item[rel.FromColumn]; ok && fk != nil {
					idStr := fmt.Sprintf("%v", fk)
					key := rel.ToTable + ":" + idStr
					if obj, ok := objCache[key]; ok {
						item[rel.FromColumn] = obj
					}
				}

			case "many-to-many":
				pivot := pivotTableName(page.TableName, rel)
				rightIDs := pivotData[pivot]
				list := make([]any, 0)
				for _, rid := range rightIDs {
					key := rel.ToTable + ":" + rid
					if obj, ok := objCache[key]; ok {
						list = append(list, obj)
					} else {
						list = append(list, rid)
					}
				}
				item[rel.FromColumn] = list
			}
		}


		dependencies := make(map[string]any)
		loaded := make(map[string]bool)

		for _, rel := range raw.Relations {
			if loaded[rel.ToTable] {
				continue
			}
			loaded[rel.ToTable] = true

			q := fmt.Sprintf(`SELECT * FROM %s`, quoteIdent(rel.ToTable))
			rs, err := sqlDB.Query(q)
			if err != nil {
				continue
			}

			cols, _ := rs.Columns()
			var arr []map[string]any

			for rs.Next() {
				vals := make([]interface{}, len(cols))
				ptrs := make([]interface{}, len(cols))
				for i := range cols {
					ptrs[i] = &vals[i]
				}
				if err := rs.Scan(ptrs...); err == nil {
					row := make(map[string]any, len(cols))
					for i, c := range cols {
						row[c] = vals[i]
					}
					arr = append(arr, row)
				}
			}

			rs.Close()
			dependencies[rel.FromColumn] = arr
		}

		c.JSON(http.StatusOK, gin.H{
			"id":        page.ID,
			"name":      page.Name,
			"template":  page.Template,
			"fiche":  page.FicheTemplate,
			"schema":    raw.UI,
			"relations": raw.Relations,
			"dependencies": dependencies,
			"item":      item,
		})
	})
}

func addFK(m map[string]map[string]struct{}, table string, id string) {
	if m[table] == nil {
		m[table] = make(map[string]struct{})
	}
	m[table][id] = struct{}{}
}

func getColumns(db *sql.DB, table string) ([]string, error) {
    q := `
        SELECT column_name 
        FROM information_schema.columns
        WHERE table_name = $1
        ORDER BY ordinal_position
    `
    rows, err := db.Query(q, table)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    cols := []string{}
    var col string
    for rows.Next() {
        if err := rows.Scan(&col); err != nil {
            continue
        }
        cols = append(cols, col)
    }

    return cols, nil
}
