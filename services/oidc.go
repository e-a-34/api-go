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
	"context"
	"log"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
)

type OIDCService struct {
	Verifier *oidc.IDTokenVerifier
	Provider *oidc.Provider
}

func InitOIDC() *OIDCService {
	issuer := os.Getenv("OIDC_ISSUER")
	clientID := os.Getenv("OIDC_CLIENT_ID")

	if issuer == "" || clientID == "" {
		log.Fatal("❌ OIDC_ISSUER ou OIDC_CLIENT_ID manquant dans .env")
	}

	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		log.Fatalf("❌ Impossible d'initialiser l'OIDC provider (%s): %v", issuer, err)
	}

	verifier := provider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})

	log.Println("🔐 OIDC initialisé avec succès pour :", issuer)

	return &OIDCService{
		Verifier: verifier,
		Provider: provider,
	}
}
