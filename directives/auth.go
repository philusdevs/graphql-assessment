package directives

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/philusdevs/graphql-assessment/dataloader"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func Authenticate(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	token := dataloader.RequestToken(ctx)
	if !dataloader.IsValidJWT(ctx, token) {
		return nil, &gqlerror.Error{
			Message: "Access Denied",
		}
	}

	return next(ctx)
}
