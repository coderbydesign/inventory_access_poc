package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/authzed/authzed-go/v1"
	"github.com/jackc/pgx/v5"
	"github.com/merlante/inventory-access-poc/api"
	"go.opentelemetry.io/otel/trace"
)

type ContentServer struct {
	Tracer        trace.Tracer
	SpicedbClient *authzed.Client
	PostgresConn  *pgx.Conn
}

type PackageName struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

type Package struct {
	NameId          int    `json:"name_id"`
	Evra            string `json:"evra"`
	DescriptionHash string `json:"description_hash"`
	SummaryHash     string `json:"summary_hash"`
	AdvisoryId      int    `json:"advisory_id"`
	Synced          bool   `json:"synced"`
}

func (p Package) VisitGetContentPackagesResponse(w http.ResponseWriter) error {
	jsonResponse, err := json.Marshal(p)
	if err != nil {
		return err
	}

	w.Write(jsonResponse)
	if err != nil {
		return err
	}

	return nil
}

func (c *ContentServer) GetContentPackages(ctx context.Context, request api.GetContentPackagesRequestObject) (api.GetContentPackagesResponseObject, error) {
	// Next:
	// - query from postgres
	// - join with spicedb data
	// - return results
	ctx, span := c.Tracer.Start(ctx, "GetContentPackages")
	defer span.End()

	_, spiceSpan := c.Tracer.Start(ctx, "SpiceDB pre-filter call")
	time.Sleep(time.Second) // mimics the delay calling out to SpiceDB
	spiceSpan.End()

	_, pgSpan := c.Tracer.Start(ctx, "Postgres query")
	p := Package{NameId: 123, Evra: "1-2", DescriptionHash: "Testing", SummaryHash: "FooBar", AdvisoryId: 321, Synced: false}
	pgSpan.End()

	return p, nil
}
