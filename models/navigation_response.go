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

type NavItem struct {
	Title        string      `json:"title"`
	Path         string      `json:"path,omitempty"`
	Icon         string      `json:"icon,omitempty"`
	Caption      string      `json:"caption,omitempty"`
	Disabled     bool        `json:"disabled,omitempty"`
	DeepMatch    bool        `json:"deepMatch,omitempty"`
	Info         interface{} `json:"info,omitempty"`
	Children     []NavItem   `json:"children,omitempty"`
}

type NavSection struct {
	Subheader string    `json:"subheader"`
	Items     []NavItem `json:"items"`
}
