package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/philusdevs/graphql-assessment/dataloader"
	"github.com/philusdevs/graphql-assessment/graph/generated"
	"github.com/philusdevs/graphql-assessment/graph/model"
	"github.com/spf13/viper"
)

func (r *mutationResolver) Authentication(ctx context.Context, input model.NewUser) (*model.Token, error) {

	t := jwt.New(jwt.SigningMethodHS256)
	t.Claims = &CustomClaims{
		&jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Minute * 30).Unix(),
		},
		UserInfo{Username: input.Username},
	}
	token, err := t.SignedString(dataloader.JwtKey)
	if err != nil {
		return nil, err
	}

	return &model.Token{JwtToken: token}, nil
}

func (r *queryResolver) Peoples(ctx context.Context, first *int) ([]*model.People, error) {
	// endpoints from config
	endpoint := viper.Get("graphql-assessment.starwars-api.endpoint").(string)
	//page path from config
	pagePath := viper.Get("graphql-assessment.starwars-api.page-path").(string)

	var url string
	if first == nil {
		// build url passing args, endpoint, path, no paging
		url = fmt.Sprintf("%s%s", endpoint, pagePath)
	} else {
		// build url passing args, endpoint, path, page number
		url = fmt.Sprintf("%s%s%d", endpoint, pagePath, *first)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	jsonData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(jsonData), &r.Results)
	if err != nil {
		return nil, err
	}

	return r.Results.Peoples, nil
}

func (r *queryResolver) PeopleByName(ctx context.Context, name string) (*model.People, error) {
	return dataloader.CtxLoaders(ctx).PeopleByName.Load(name)
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
