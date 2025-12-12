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
	"database/sql"
	"fmt"
	"strings"
)


func InsertDynamic(db *sql.DB, table string, fields map[string]any) (string, error) {
	if len(fields) == 0 {
		return "", fmt.Errorf("aucune donnée à insérer")
	}

	cols := []string{}
	params := []string{}
	args := []any{}
	i := 1

	for col, val := range fields {
		cols = append(cols, quoteIdent(col))
		params = append(params, fmt.Sprintf("$%d", i))
		args = append(args, val)
		i++
	}

	query := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s) RETURNING id`,
		quoteIdent(table),
		strings.Join(cols, ", "),
		strings.Join(params, ", "),
	)

	var newID string
	err := db.QueryRow(query, args...).Scan(&newID)
	return newID, err
}


func InsertPivotM2M(db *sql.DB, pivotTable string, leftID string, rightIDs []string) error {
	if len(rightIDs) == 0 {
		return nil
	}

	query := fmt.Sprintf(
		`INSERT INTO %s (left_id, right_id) VALUES ($1, $2)`,
		quoteIdent(pivotTable),
	)

	for _, r := range rightIDs {
		if _, err := db.Exec(query, leftID, r); err != nil {
			return err
		}
	}

	return nil
}


func ClearPivot(db *sql.DB, pivotTable, leftID string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE left_id = $1`, quoteIdent(pivotTable))
	_, err := db.Exec(q, leftID)
	return err
}

func UpdateDynamic(db *sql.DB, table string, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}

	sets := []string{}
	args := []any{}
	i := 1

	for col, val := range fields {
		sets = append(sets, fmt.Sprintf("%s = $%d", quoteIdent(col), i))
		args = append(args, val)
		i++
	}

	args = append(args, id)

	q := fmt.Sprintf(
		`UPDATE %s SET %s WHERE id = $%d`,
		quoteIdent(table),
		strings.Join(sets, ", "),
		len(args),
	)

	_, err := db.Exec(q, args...)
	return err
}
