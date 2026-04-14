package internal

// This file ensures that dependencies required for subsequent development phases
// are tracked in go.mod. These imports will be used by handler implementations.

import (
	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/jung-kurt/gofpdf"
	_ "github.com/robfig/cron/v3"
	_ "github.com/xuri/excelize/v2"
	_ "golang.org/x/crypto/argon2"
)
