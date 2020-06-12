package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/pkg/errors"
	"github.com/stevenferrer/solr-go"
	solrindex "github.com/stevenferrer/solr-go/index"
	solrschema "github.com/stevenferrer/solr-go/schema"
)

type M = map[string]interface{}

const (
	collection = "multi-select-demo"
	dataPath   = "phones.json"
)

func main() {
	initSchema := flag.Bool("init-schema", false, "initialize solr schema")
	index := flag.Bool("index", false, "index products")
	flag.Parse()

	solrClient := solr.NewClient("localhost", 8983)

	ctx := context.Background()
	if *initSchema {
		log.Print("initializing solr schema...")
		err := initSolrSchema(ctx, solrClient.Schema())
		if err != nil {
			log.Fatal(err)
		}
	}

	if *index {
		log.Println("indexing products...")
		err := indexProducts(ctx, solrClient.Index())
		if err != nil {
			log.Fatal(err)
		}
	}

	r := chi.NewRouter()

	// basic middlewares
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// search handler
	r.Method(http.MethodGet, "/search", &searchHandler{
		solrClient: solrClient,
	})

	addr := ":8081"
	log.Printf("listening on %s\n", addr)
	err := http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatal(err)
	}
}

func initSolrSchema(ctx context.Context, schemaClient solrschema.Client) error {
	// define the fields
	fields := []solrschema.Field{
		{
			Name:    "docType",
			Type:    "string",
			Indexed: true,
			Stored:  true,
		},
		{
			Name:    "name",
			Type:    "text_general",
			Indexed: true,
			Stored:  true,
		},
		{
			Name:    "category",
			Type:    "text_gen_sort",
			Indexed: true,
			Stored:  true,
		},
		{
			Name:    "brand",
			Type:    "text_gen_sort",
			Indexed: true,
			Stored:  true,
		},
		{
			Name:    "productType",
			Type:    "string",
			Indexed: true,
			Stored:  true,
		},
	}

	for _, field := range fields {
		err := schemaClient.AddField(ctx, collection, field)
		if err != nil {
			return errors.Wrap(err, "add field")
		}
	}

	// define copy field
	err := schemaClient.AddCopyField(ctx, collection, solrschema.CopyField{
		Source: "*",
		Dest:   "_text_",
	})
	if err != nil {
		return errors.Wrap(err, "add copy field")
	}

	return nil
}

func indexProducts(ctx context.Context, indexClient solrindex.JSONClient) error {
	b, err := ioutil.ReadFile(dataPath)
	if err != nil {
		return err
	}

	var docs []M
	err = json.Unmarshal(b, &docs)
	if err != nil {
		return err
	}

	err = indexClient.AddMultiple(ctx, collection, docs)
	if err != nil {
		return err
	}

	return nil
}
