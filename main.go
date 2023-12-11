package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/merlante/inventory-access-poc/opentelemetry"
	"go.opentelemetry.io/otel"
	"net/http"
	"os"

	"github.com/merlante/inventory-access-poc/api"
	"github.com/merlante/inventory-access-poc/cachecontent"
	"github.com/merlante/inventory-access-poc/client"
	"github.com/merlante/inventory-access-poc/migration"
	"github.com/merlante/inventory-access-poc/server"
)

var (
	spiceDBURL   = "localhost:50051"
	spiceDBToken = "foobar"
	contentPgUri = "postgres://postgres:secret@content-postgres:5434/content?sslmode=disable"
)

func main() {
	overwriteVarsFromEnv()

	otelShutdown, err := initOpenTelemetry()
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	if os.Getenv("RUN_ACTION") == "REFRESH_PACKAGE_CACHES" {
		RefreshPackagesCaches()
	} else {
		initServer()
	}
}

func RefreshPackagesCaches() {
	cachecontent.Configure(contentPgUri)
	cachecontent.RefreshPackagesCaches(nil)
}

func initServer() {
	spiceDbClient, err := client.GetSpiceDbClient(spiceDBURL, spiceDBToken)
	if err != nil {
		err := fmt.Errorf("%v", err)
		fmt.Println(err)
		os.Exit(1)
	}

	pgConn, err := client.GetPostgresConnection(contentPgUri)
	if err != nil {
		err := fmt.Errorf("%v", err)
		fmt.Println(err)
		os.Exit(1)
	}
	defer pgConn.Close(context.Background())

	if os.Getenv("RUN_ACTION") == "MIGRATE_CONTENT_TO_SPICEDB" {
		fmt.Printf("Running migration from ContentDB to SpiceDB")
		migrator := migration.NewPSQLToSpiceDBMigration(pgConn, spiceDbClient)
		if err := migrator.MigrateContentHostsAndSystemsToSpiceDb(context.TODO()); err != nil {
			panic(err)
		}
		return
	}
	if os.Getenv("RUN_ACTION") == "MIGRATE_PACKAGES_TO_SPICEDB" {
		fmt.Printf("Running migration of packages from ContentDB to SpiceDB")
		migrator := migration.NewPSQLToSpiceDBMigration(pgConn, spiceDbClient)
		if err := migrator.MigratePackages(context.TODO()); err != nil {
			panic(err)
		}
		return
	}

	tracer := otel.Tracer("HttpServer")

	srv := server.ContentServer{
		Tracer:        tracer,
		SpicedbClient: spiceDbClient,
		PostgresConn:  pgConn,
	}
	r := api.Handler(api.NewStrictHandler(&srv, nil))

	sErr := http.ListenAndServe(":8080", r)

	if sErr != nil {
		err := fmt.Errorf("error at server startup: %v", sErr)
		fmt.Println(err)
		os.Exit(1)
	}
}

func overwriteVarsFromEnv() {
	envSpicedbUrl := os.Getenv("SPICEDB_URL")
	if envSpicedbUrl != "" {
		spiceDBURL = envSpicedbUrl
	}

	envSpicedbPsk := os.Getenv("SPICEDB_PSK")
	if envSpicedbPsk != "" {
		spiceDBToken = envSpicedbPsk
	}

	envContentPgUri := os.Getenv("CONTENT_POSTGRES_URI")
	if envContentPgUri != "" {
		contentPgUri = envContentPgUri
	}
}

func initOpenTelemetry() (shutdown func(context.Context) error, err error) {
	// Set up OpenTelemetry.
	serviceName := "inventory_access_poc"
	serviceVersion := "0.1.0"
	shutdown, err = opentelemetry.SetupOTelSDK(context.TODO(), serviceName, serviceVersion)

	return
}
