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
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)


type RelationDefinition struct {
	Type       string `json:"type"`
	FromColumn string `json:"fromColumn"`
	ToTable    string `json:"toTable"`
	OnDelete   string `json:"onDelete"`
	PivotTable string `json:"pivotTable,omitempty"`
}

type schemaRaw struct {
	UI        []map[string]any     `json:"ui"`
	Relations []RelationDefinition `json:"relations"`
}

func RegisterPublicPageRoutes(r gin.IRoutes, db *gorm.DB) {
	r.GET("/page/:id", func(c *gin.Context) {
		id := c.Param("id")
		var page models.Page
		if err := db.Preload("Template").First(&page, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "❌ Page introuvable"})
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
		} else {
			raw.UI = []map[string]any{}
		}

		var menuDefs []map[string]any
		if page.SchemaMenuUiDeployed != nil {
			_ = json.Unmarshal(page.SchemaMenuUiDeployed, &menuDefs)
		}

		menus := make([]map[string]any, 0, len(menuDefs))
		for _, m := range menuDefs {
			menus = append(menus, map[string]any{
				"name":  m["name"],
				"order": m["order"],
				"refId": fmt.Sprintf("%v", m["refId"]),
			})
		}

		data := []map[string]any{}
		dependencies := make(map[string]any)

		if Bool(page.Deploy) && page.TableName != "" {
			sqlDB, _ := db.DB()
			rows, err := sqlDB.Query(fmt.Sprintf(`SELECT * FROM %s`, quoteIdent(page.TableName)))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			defer rows.Close()

			cols, _ := rows.Columns()
			rawRows := make([]map[string]any, 0)
			allIDs := make([]string, 0)

			for rows.Next() {
				values := make([]interface{}, len(cols))
				ptrs := make([]interface{}, len(cols))
				for i := range cols {
					ptrs[i] = &values[i]
				}
				if err := rows.Scan(ptrs...); err != nil {
					continue
				}

				entry := make(map[string]any, len(cols))
				for i, col := range cols {
					entry[col] = values[i]
				}

				if idv, ok := entry["id"]; ok && idv != nil {
					allIDs = append(allIDs, fmt.Sprintf("%v", idv))
				}

				rawRows = append(rawRows, entry)
			}

			if len(rawRows) == 0 {
				c.JSON(http.StatusOK, gin.H{
					"id":           page.ID,
					"name":         page.Name,
					"template":     page.Template,
					"schema":       raw.UI,
					"menus":        menus,
					"relations":    raw.Relations,
					"data":         data,
					"dependencies": dependencies,
				})
				return
			}


			pivotData := make(map[string]map[string][]string)

			for _, rel := range raw.Relations {
				if rel.Type != "many-to-many" || len(allIDs) == 0 {
					continue
				}
				pivot := pivotTableName(page.TableName, rel)
				in := "'" + strings.Join(allIDs, "','") + "'"
				query := fmt.Sprintf(
					`SELECT left_id, right_id FROM %s WHERE left_id IN (%s)`,
					quoteIdent(pivot), in,
				)

				rs, err := sqlDB.Query(query)
				if err != nil {
					continue
				}

				m := make(map[string][]string)
				for rs.Next() {
					var left, right string
					if err := rs.Scan(&left, &right); err == nil {
						m[left] = append(m[left], right)
					}
				}
				rs.Close()

				pivotData[pivot] = m
			}

			fkByTable := make(map[string]map[string]struct{})

			for _, rel := range raw.Relations {
				if rel.Type != "one-to-one" && rel.Type != "one-to-many" {
					continue
				}

				for _, entry := range rawRows {
					if fk, ok := entry[rel.FromColumn]; ok && fk != nil {
						idStr := fmt.Sprintf("%v", fk)
						if idStr == "" {
							continue
						}
						if fkByTable[rel.ToTable] == nil {
							fkByTable[rel.ToTable] = make(map[string]struct{})
						}
						fkByTable[rel.ToTable][idStr] = struct{}{}
					}
				}
			}

			for _, rel := range raw.Relations {
				if rel.Type != "many-to-many" {
					continue
				}
				pivot := pivotTableName(page.TableName, rel)
				pairs := pivotData[pivot]
				if pairs == nil {
					continue
				}

				for _, rights := range pairs {
					for _, rid := range rights {
						if fkByTable[rel.ToTable] == nil {
							fkByTable[rel.ToTable] = make(map[string]struct{})
						}
						fkByTable[rel.ToTable][rid] = struct{}{}
					}
				}
			}

			objCache := batchLoadRelated(sqlDB, fkByTable)

			for _, entry := range rawRows {
				for _, rel := range raw.Relations {

					switch rel.Type {

					case "one-to-one", "one-to-many":
						if fk, ok := entry[rel.FromColumn]; ok && fk != nil {
							idStr := fmt.Sprintf("%v", fk)
							if idStr == "" {
								continue
							}
							key := rel.ToTable + ":" + idStr

							if obj, ok := objCache[key]; ok {
								entry[rel.FromColumn] = obj
							}
						}

					case "many-to-many":
						pivot := pivotTableName(page.TableName, rel)
						entryID := fmt.Sprintf("%v", entry["id"])

						if pairs, ok := pivotData[pivot]; ok {
							rightIDs := pairs[entryID]
							if len(rightIDs) == 0 {
								entry[rel.FromColumn] = []any{}
								continue
							}

							list := make([]any, 0, len(rightIDs))
							for _, rid := range rightIDs {
								key := rel.ToTable + ":" + rid
								if obj, ok := objCache[key]; ok {
									list = append(list, obj)
								} else {
									list = append(list, rid)
								}
							}
							entry[rel.FromColumn] = list
						} else {
							entry[rel.FromColumn] = []any{}
						}
					}
				}

				data = append(data, entry)
			}

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
		}

		c.JSON(http.StatusOK, gin.H{
			"id":           page.ID,
			"name":         page.Name,
			"template":     page.Template,
			"schema":       raw.UI,
			"menus":        menus,
			"functions":	page.SchemaFunctionsDeployed,
			"conditions":	page.SchemaConditionsDeployed,
			"relations":    raw.Relations,
			"data":         data,
			"dependencies": dependencies,
		})
	})
	r.POST("/page/:id", func(c *gin.Context) {
		id := c.Param("id")

		var page models.Page
		if err := db.First(&page, "id = ?", id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Page introuvable"})
			return
		}

		if page.TableName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "TableName manquant"})
			return
		}

		var raw schemaRaw
		if page.SchemaRelationsDeployed != nil {
			_ = json.Unmarshal(page.SchemaRelationsDeployed, &raw.Relations)
		}

		var payload map[string]any
		if err := c.BindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		sqlDB, _ := db.DB()


		simpleFields := map[string]any{}
		m2mFields := map[string][]string{} 

		for _, rel := range raw.Relations {
			if rel.Type == "many-to-many" {
				if v, ok := payload[rel.FromColumn]; ok && v != nil {

					arr, ok := v.([]interface{})
					if !ok {
						fmt.Println("⚠️ Format M2M invalide pour", rel.FromColumn)
						delete(payload, rel.FromColumn)
						continue
					}

					ids := []string{}

					for _, a := range arr {
						switch val := a.(type) {

						case string:
							ids = append(ids, val)
						case map[string]interface{}:
							if idv, ok := val["id"]; ok {
								ids = append(ids, fmt.Sprintf("%v", idv))
							}

						default:
							fmt.Println("⚠️ Valeur M2M inconnue:", a)
						}
					}

					m2mFields[rel.FromColumn] = ids
				}

				delete(payload, rel.FromColumn)
			}
		}

		for k, v := range payload {
			simpleFields[k] = v
		}

		newID, err := InsertDynamic(sqlDB, page.TableName, simpleFields)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, rel := range raw.Relations {
			if rel.Type != "many-to-many" {
				continue
			}

			rightIDs := m2mFields[rel.FromColumn]
			pivotTable := pivotTableName(page.TableName, rel)

			if len(rightIDs) == 0 {
				continue
			}

			if err := InsertPivotM2M(sqlDB, pivotTable, newID, rightIDs); err != nil {
				fmt.Println("❌ Erreur pivot:", err)
			}
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Création OK",
			"id":      newID,
		})
	})


}


func quoteIdent(ident string) string {
	safe := strings.ReplaceAll(ident, `"`, `""`)
	return `"` + safe + `"`
}

func pivotTableName(pageTable string, rel RelationDefinition) string {
	if rel.PivotTable != "" {
		return rel.PivotTable
	}
	return strings.ToLower(fmt.Sprintf("%s_%s_%s", pageTable, rel.FromColumn, rel.ToTable))
}
func batchLoadRelated(db *sql.DB, fkByTable map[string]map[string]struct{}) map[string]map[string]any {
	cache := make(map[string]map[string]any)

	for table, idSet := range fkByTable {
		if len(idSet) == 0 {
			continue
		}

		ids := make([]string, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
		}

		placeholders := make([]string, len(ids))
		args := make([]interface{}, len(ids))
		for i, id := range ids {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = id
		}

		q := fmt.Sprintf(
			`SELECT * FROM %s WHERE id IN (%s)`,
			quoteIdent(table),
			strings.Join(placeholders, ","),
		)

		rs, err := db.Query(q, args...)
		if err != nil {
			continue
		}

		cols, _ := rs.Columns()

		for rs.Next() {
			vals := make([]interface{}, len(cols))
			ptrs := make([]interface{}, len(cols))
			for i := range cols {
				ptrs[i] = &vals[i]
			}
			if err := rs.Scan(ptrs...); err != nil {
				continue
			}

			row := make(map[string]any, len(cols))
			var idVal string

			for i, c := range cols {
				v := vals[i]
				row[c] = v
				if c == "id" && v != nil {
					idVal = fmt.Sprintf("%v", v)
				}
			}

			if idVal != "" {
				key := table + ":" + idVal
				cache[key] = row
			}
		}

		rs.Close()
	}
	return cache
}