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

package services

import (
	"api-core-v2/models"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

func SyncUserFromClaims(db *gorm.DB, claims map[string]interface{}) error {

	sub := claims["sub"].(string)
	email := claims["email"].(string)
	name := claims["name"].(string)
	given := claims["given_name"].(string)
	family := claims["family_name"].(string)
	preferred := claims["preferred_username"].(string)
	groupsJson, _ := json.Marshal(claims["groups"])

	var user models.User
	result := db.Where("sub = ?", sub).First(&user)

	now := time.Now()

	if result.Error == gorm.ErrRecordNotFound {
		user = models.User{
			Sub:               sub,
			Email:             email,
			Name:              name,
			GivenName:         given,
			FamilyName:        family,
			PreferredUsername: preferred,
			Groups:            groupsJson,
			FirstLogin:        now,
			LastLogin:         &now,
			LoginCount:        1,
			Iss:               claims["iss"].(string),
		}
		return db.Create(&user).Error
	}

	user.Email = email
	user.Name = name
	user.GivenName = given
	user.FamilyName = family
	user.PreferredUsername = preferred
	user.Groups = groupsJson
	user.Iss = claims["iss"].(string)

	user.LastLogin = &now
	user.LoginCount++

	return db.Save(&user).Error
}
