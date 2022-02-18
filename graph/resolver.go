package graph

import (
	"github.com/golang-jwt/jwt"
	"github.com/philusdevs/graphql-assessment/graph/model"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

//go:generate go run github.com/99designs/gqlgen generate
type Resolver struct {
	Results *model.Results
}

type UserInfo struct {
	Username string
}

type CustomClaims struct {
	*jwt.StandardClaims
	UserInfo
}
