package main

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/vingarcia/ddd-go-layout/assets/html"
	"github.com/vingarcia/ddd-go-layout/cmd/api/middlewares"
	"github.com/vingarcia/ddd-go-layout/cmd/api/usersctrl"
	"github.com/vingarcia/ddd-go-layout/cmd/api/venuesctrl"
	"github.com/vingarcia/ddd-go-layout/domain"
	"github.com/vingarcia/ddd-go-layout/domain/users"
	"github.com/vingarcia/ddd-go-layout/domain/venues"
	"github.com/vingarcia/ddd-go-layout/infra/env"
	"github.com/vingarcia/ddd-go-layout/infra/jsonlogs"
	"github.com/vingarcia/ddd-go-layout/infra/memorycache"
	"github.com/vingarcia/ddd-go-layout/infra/redis"
	"github.com/vingarcia/ddd-go-layout/infra/rest"
	"github.com/vingarcia/ddd-go-layout/infra/usersrepo"
	"github.com/vingarcia/ksql"

	adapter "github.com/vingarcia/go-adapter/fiber/v2"

	_ "github.com/lib/pq"
)

func main() {
	ctx := context.Background()

	// Read all configs at once so its easy to spot all of them:
	port := env.GetString("PORT", "80")
	logLevel := env.GetString("LOG_LEVEL", "INFO")
	foursquareBaseURL := env.MustGetString("FOURSQUARE_BASE_URL")
	foursquareClientID := env.MustGetString("FOURSQUARE_CLIENT_ID")
	foursquareSecret := env.MustGetString("FOURSQUARE_SECRET")
	redisURL := env.GetString("REDIS_URL", "")
	redisPassword := env.GetString("REDIS_PASSWORD", "")
	dbURL := env.MustGetString("DATABASE_URL")

	// Dependency Injection goes here:
	logger := jsonlogs.NewClient(logLevel)

	restClient := rest.NewClient(30 * time.Second)

	var cacheClient domain.CacheProvider
	if redisURL != "" {
		cacheClient = redis.NewClient(redisURL, redisPassword, 24*time.Hour)
	} else {
		cacheClient = memorycache.NewClient(24*time.Hour, 10*time.Minute)
	}

	venuesService := venues.NewService(
		logger,
		restClient,
		cacheClient,
		foursquareBaseURL,
		foursquareClientID,
		foursquareSecret,
	)

	// The controllers handle HTTP stuff so the services can be kept as simple as possible
	// only working on top of the domain language, i.e. types and interfaces from the domain/ package
	venuesController := venuesctrl.NewController(venuesService)

	db, err := ksql.New("postgres", dbURL, ksql.Config{})
	if err != nil {
		logger.Fatal(ctx, "unable to start database", domain.LogBody{
			"db_url": dbURL,
			"error":  err.Error(),
		})
	}

	usersRepo := usersrepo.NewClient(db)

	usersService := users.NewService(logger, usersRepo)

	usersController := usersctrl.NewController(usersService)

	// Any framework you need for serving HTTP or GRPC goes in the main package,
	//
	// It should be kept here because the main package is the only one that is allowed
	// to depend on anything, and also because this logic is unique to this endpoint,
	// so you won't reuse it anywhere else.
	app := fiber.New()

	app.Use(middlewares.HandleRequestID())
	app.Use(middlewares.HandleError(logger))

	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})
	app.Get("/venues/:latitude,:longitude", adapter.Adapt(venuesController.GetVenuesByCoordinates))
	app.Get("/venues/details/:id", adapter.Adapt(venuesController.GetDetails))

	app.Post("/users", usersController.UpsertUser)
	app.Get("/users/:id", usersController.GetUser)

	// Just an example on how to serve html templates using the embed library
	// and explicit arguments with a "builder function":
	app.Get("/example-html", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return html.WriteExamplePage(c, "username", "user address", 42)
	})

	logger.Info(ctx, "server-starting-up", domain.LogBody{
		"port": port,
	})
	if err := app.Listen(":" + port); err != nil {
		logger.Error(ctx, "server-stopped-with-an-error", domain.LogBody{
			"error": err.Error(),
		})
	}
}
