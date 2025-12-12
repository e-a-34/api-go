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

package models

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	ID                string          `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Sub               string          `gorm:"unique;not null" json:"sub"`
	Email             string          `gorm:"uniqueIndex;not null" json:"email"`
	GivenName         string          `json:"givenName"`
	FamilyName        string          `json:"familyName"`
	Name              string          `json:"name"`
	PreferredUsername string          `gorm:"index" json:"preferredUsername"`
	Groups            json.RawMessage `gorm:"type:jsonb" json:"groups"`
	IsAdmin           *bool            `gorm:"default:false" json:"isAdmin"`
	FirstLogin        time.Time       `gorm:"autoCreateTime" json:"firstLogin"`
	LastLogin         *time.Time      `json:"lastLogin"`
	LoginCount        int             `gorm:"default:0" json:"loginCount"`
	Iss               string          `json:"iss"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	Tags []Tag `gorm:"many2many:user_tags;constraint:OnDelete:CASCADE;" json:"tags,omitempty" crud:"dependency"`
}

type TagCategory struct {
	ID        string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name 	  string `gorm:"not null" json:"name"`
	Tags      []Tag     `gorm:"foreignKey:CategoryID;references:ID" json:"tags,omitempty" crud:"dependency"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

type Tag struct {
	ID         string       `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name       string       `gorm:"not null" json:"name"`
	Color      string       `gorm:"type:varchar(7)" json:"color"`
	CategoryID *string      `gorm:"type:uuid" json:"categoryId,omitempty"`
	Category   *TagCategory `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:CategoryID;references:ID" json:"category,omitempty" crud:"dependency"`
	CreatedAt  time.Time    `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time    `gorm:"autoUpdateTime" json:"updatedAt"`
}


type Template struct {
    ID          string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
    Name        string    `gorm:"unique;not null" json:"name"`
    Description string    `gorm:"type:text" json:"description,omitempty"`
    IsFiche     *bool     `gorm:"default:false" json:"isFiche"`
    CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}


type Page struct {
	ID          string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name        string         `gorm:"unique;not null" json:"name"`
	TemplateID  *string        `gorm:"type:uuid" json:"templateId,omitempty"`
	Template    *Template      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"template,omitempty" crud:"dependency"`

	FicheTemplateID *string   `gorm:"type:uuid" json:"ficheTemplateId,omitempty"`
    FicheTemplate   *Template `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"ficheTemplate,omitempty" crud:"dependency"`

	SchemaColumns    datatypes.JSON `gorm:"type:jsonb;column:schema_columns" json:"schemaColumns,omitempty"`
	SchemaRelations  datatypes.JSON `gorm:"type:jsonb;column:schema_relations" json:"schemaRelations,omitempty"`
	SchemaUi         datatypes.JSON `gorm:"type:jsonb;column:schema_ui" json:"schemaUi,omitempty"`
	SchemaMenuUi     datatypes.JSON `gorm:"type:jsonb;column:schema_menu_ui" json:"schemaMenuUi,omitempty"`
	SchemaConditions datatypes.JSON `gorm:"type:jsonb;column:schema_conditions" json:"schemaConditions,omitempty"`
	SchemaFunctions datatypes.JSON `gorm:"type:jsonb;column:schema_functions" json:"schemaFunctions,omitempty"`

	SchemaColumnsDeployed    datatypes.JSON `gorm:"type:jsonb;column:schema_columns_deployed" json:"schemaColumnsDeployed,omitempty"`
	SchemaRelationsDeployed  datatypes.JSON `gorm:"type:jsonb;column:schema_relations_deployed" json:"schemaRelationsDeployed,omitempty"`
	SchemaUiDeployed         datatypes.JSON `gorm:"type:jsonb;column:schema_ui_deployed" json:"schemaUiDeployed,omitempty"`
	SchemaMenuUiDeployed     datatypes.JSON `gorm:"type:jsonb;column:schema_menu_ui_deployed" json:"schemaMenuUiDeployed,omitempty"`
	SchemaConditionsDeployed datatypes.JSON `gorm:"type:jsonb;column:schema_conditions_deployed" json:"schemaConditionsDeployed,omitempty"`
	SchemaFunctionsDeployed datatypes.JSON `gorm:"type:jsonb;column:schema_functions_deployed" json:"schemaFunctionsDeployed,omitempty"`
	

	TableName string `gorm:"type:varchar(255)" json:"tableName"`
	Deploy    *bool   `gorm:"default:false" json:"deploy"`

	Tags []Tag `gorm:"many2many:page_tags;constraint:OnDelete:CASCADE;" json:"tags,omitempty" crud:"dependency"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

type NavigationItem struct {
	ID       string          `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ParentID *string         `gorm:"type:uuid;index" json:"parentId,omitempty"`
	Parent   *NavigationItem `gorm:"foreignKey:ParentID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"parent,omitempty" crud:"dependency"`
	Lft   int `gorm:"index" json:"lft"`
	Rgt   int `gorm:"index" json:"rgt"`
	Depth int `gorm:"default:0" json:"depth"`
	Title     string `gorm:"not null" json:"title"`
	Icon      string `json:"icon,omitempty"`
	Path      string `gorm:"index" json:"path,omitempty"`
	Order     int    `gorm:"default:0" json:"order"`
	Disabled  *bool   `gorm:"default:false" json:"disabled"`
	Caption   string `json:"caption,omitempty"`
	DeepMatch *bool   `gorm:"default:false" json:"deepMatch"`
	IsHeader *bool   `gorm:"default:false" json:"isHeader"`
	IsAdmin   *bool   `gorm:"default:false" json:"isAdmin"`
	PageID *string `gorm:"type:uuid;index" json:"pageId,omitempty"`
	Page   *Page   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"page,omitempty" crud:"dependency"`
	Tags   []Tag             `gorm:"many2many:navigation_item_tags;constraint:OnDelete:CASCADE;" json:"tags,omitempty" crud:"dependency"`
	Extras datatypes.JSONMap `gorm:"type:jsonb" json:"extras,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

type AuditLog struct {
	ID         string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID     *string        `gorm:"type:uuid;index" json:"userId,omitempty"`
	User       *User          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"user,omitempty" crud:"dependency"`
	Action     string         `gorm:"not null" json:"action"`
	Resource   string         `gorm:"index" json:"resource"`
	ResourceID *string        `gorm:"type:uuid;index" json:"resourceId,omitempty"`
	Status     string         `gorm:"not null" json:"status"`
	IP         string         `json:"ip,omitempty"`
	UserAgent  string         `json:"userAgent,omitempty"`
	Metadata   datatypes.JSON `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"createdAt"`
}

func AutoMigrateAll(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&AuditLog{},
		&TagCategory{},
		&Tag{},
		&Template{},
		&Page{},
		&NavigationItem{},
	)
}
