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
	"errors"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
    btrue  = true
    bfalse = false
)

func InitDefaultData(db *gorm.DB) {
    var cNav, cCat, cTag, cPerm, cTemp int64
    db.Model(&models.NavigationItem{}).Count(&cNav)
    db.Model(&models.TagCategory{}).Count(&cCat)
    db.Model(&models.Tag{}).Count(&cTag)
    db.Model(&models.Template{}).Count(&cTemp)

    if cNav+cCat+cTag+cPerm+cTemp > 0 {
        return
    }

    if err := db.Transaction(func(tx *gorm.DB) error {
        if err := seedNavigation(tx); err != nil {
            return err
        }
        if err := seedTagCategories(tx); err != nil {
            return err
        }
        if err := seedTags(tx); err != nil {
            return err
        }
        if err := seedTemplate(tx); err != nil {
            return err
        }
        return nil
    }); err != nil {
        log.Printf("❌ Erreur init: %v", err)
        return
    }

    log.Println("✅ Données de base initialisées.")
}

func seedTemplate(db *gorm.DB) error {
    var count int64
    if err := db.Model(&models.Template{}).Count(&count).Error; err != nil {
        return err
    }
    if count > 0 {
        return nil
    }

    templates := []models.Template{
        {Name: "List"},
        {Name: "Fiche"},
        {Name: "Lens"},
        {Name: "Base"},
        {Name: "Default", IsFiche: &btrue},
        {Name: "Kubevirt", IsFiche: &btrue},
    }

    return db.Create(&templates).Error
}

func seedNavigation(db *gorm.DB) error {
    var count int64
    db.Model(&models.NavigationItem{}).Count(&count)
    if count > 0 {
        return nil
    }

    admin := models.NavigationItem{
        Title:    "Administration",
        IsHeader: &btrue,
        IsAdmin:  &btrue,
        Lft:      1,
        Rgt:      16,
        Depth:    0,
    }
    if err := db.Create(&admin).Error; err != nil {
        return err
    }

    settings := models.NavigationItem{
        Title:    "Settings",
        Icon:     "mdi:settings",
        ParentID: &admin.ID,
        Lft:      2,
        Rgt:      15,
        Depth:    1,
        IsAdmin:  &btrue,
    }
    if err := db.Create(&settings).Error; err != nil {
        return err
    }

    children := []models.NavigationItem{
        {Title: "Navigation", Path: "/dashboard/settings/navigation", ParentID: &settings.ID, Lft: 3, Rgt: 4, Depth: 2, IsAdmin: &btrue},
        {Title: "Users", Path: "/dashboard/settings/users", ParentID: &settings.ID, Lft: 5, Rgt: 6, Depth: 2, IsAdmin: &btrue},
        {Title: "Tags", Path: "/dashboard/settings/tags", ParentID: &settings.ID, Lft: 7, Rgt: 8, Depth: 2, IsAdmin: &btrue},
        {Title: "Tag Categories", Path: "/dashboard/settings/tag-categories", ParentID: &settings.ID, Lft: 9, Rgt: 10, Depth: 2, IsAdmin: &btrue},
        {Title: "Permissions", Path: "/dashboard/settings/permissions", ParentID: &settings.ID, Lft: 11, Rgt: 12, Depth: 2, IsAdmin: &btrue},
        {Title: "Builder", Path: "/dashboard/settings/builder", ParentID: &settings.ID, Lft: 13, Rgt: 14, Depth: 2, IsAdmin: &btrue},
    }

    return db.Create(&children).Error
}

func seedTagCategories(db *gorm.DB) error {
    var count int64
    if err := db.Model(&models.TagCategory{}).Count(&count).Error; err != nil {
        return err
    }
    if count > 0 {
        return nil
    }

    categories := []models.TagCategory{
        {Name: "Environnement"},
        {Name: "Application"},
        {Name: "Filiale"},
    }

    return db.Create(&categories).Error
}

func seedTags(db *gorm.DB) error {
    findOrCreateCat := func(name string) (models.TagCategory, error) {
        var cat models.TagCategory
        if err := db.Where("name = ?", name).First(&cat).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                cat = models.TagCategory{Name: name}
                if err := db.Create(&cat).Error; err != nil {
                    return models.TagCategory{}, err
                }
            } else {
                return models.TagCategory{}, err
            }
        }
        return cat, nil
    }

    envCat, err := findOrCreateCat("Environnement")
    if err != nil {
        return err
    }

    envTags := []models.Tag{
        {Name: "Hprod", Color: "#7E57C2"},
        {Name: "Pprod", Color: "#42A5F5"},
        {Name: "Prod", Color: "#66BB6A"},
        {Name: "Developpement", Color: "#FFA726"},
        {Name: "Integration", Color: "#29B6F6"},
        {Name: "Formation", Color: "#AB47BC"},
    }

    for i := range envTags {
        envTags[i].CategoryID = &envCat.ID
    }

    return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&envTags).Error
}
