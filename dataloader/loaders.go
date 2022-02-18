package dataloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/philusdevs/graphql-assessment/graph/model"
	"github.com/spf13/viper"
)

type ctxKeyType struct{ name string }

var ctxPeople = ctxKeyType{"peopleCtx"}
var ctxReqToken = ctxKeyType{"tokenCtx"}

// For signing key context
var ctxSignedKey = ctxKeyType{"signedKey"}

// Create the JWT key used to create the signature
var JwtKey = []byte("my_secret_key")

type loaders struct {
	PeopleByName *PeopleLoader
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		ldrs := loaders{}
		ldrs.PeopleByName = &PeopleLoader{
			maxBatch: 100,
			wait:     1 * time.Millisecond,
			fetch: func(keys []string) ([]*model.People, []error) {
				name := keys[0]

				// Endpoints from config
				endpoint := viper.Get("graphql-assessment.starwars-api.endpoint").(string)
				// Search path from config
				searchPath := viper.Get("graphql-assessment.starwars-api.search-path").(string)

				searchUrl := fmt.Sprintf("%s%s%s", endpoint, searchPath, url.QueryEscape(name))
				errors := make([]error, 2)
				// Get call to Star Wars REST api
				res, err := http.Get(searchUrl)
				if err != nil {
					errors = append(errors, err)
					return nil, errors
				}

				jsonData, err := ioutil.ReadAll(res.Body)
				if err != nil {
					errors = append(errors, err)
					return nil, errors
				}

				var result *model.Results
				err = json.Unmarshal([]byte(jsonData), &result)
				if err != nil {
					errors = append(errors, err)
					return nil, errors
				}
				return result.Peoples, nil
			},
		}

		ctx := context.WithValue(r.Context(), ctxPeople, ldrs)
		ctx = context.WithValue(ctx, ctxSignedKey, JwtKey)
		ctx = context.WithValue(ctx, ctxReqToken, r.Header.Get("Authorization"))
		next.ServeHTTP(rw, r.WithContext(ctx))
	})
}

func CtxLoaders(ctx context.Context) loaders {
	return ctx.Value(ctxPeople).(loaders)
}

func RequestToken(ctx context.Context) string {
	return ctx.Value(ctxReqToken).(string)
}

func IsValidJWT(ctx context.Context, tokenString string) bool {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Printf("Unexpected signing method: %v", token.Header["alg"])
		}
		return JwtKey, nil
	})

	if err != nil {
		log.Printf("[Token Error] - %s", err)
		return false
	}

	if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return true
	} else {
		return false
	}
}
