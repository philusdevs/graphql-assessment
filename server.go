package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/philusdevs/graphql-assessment/dataloader"
	"github.com/philusdevs/graphql-assessment/directives"
	"github.com/philusdevs/graphql-assessment/graph"
	"github.com/philusdevs/graphql-assessment/graph/generated"
	"github.com/spf13/viper"
)

const defaultPort = "8000"

func main() {
	viper.SetConfigFile("config.yml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error while reading config file %s", err)
		os.Exit(1)
	}

	port := strconv.Itoa(viper.Get("graphql-assessment.server.port").(int))
	if port == "" {
		port = defaultPort
	}

	c := generated.Config{Resolvers: &graph.Resolver{}}
	c.Directives.IsAuthenticated = directives.Authenticate

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(c))
	router := http.NewServeMux()

	router.Handle("/", playground.Handler("GraphQL playground", "/query"))
	router.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, dataloader.Middleware(router)))
}
