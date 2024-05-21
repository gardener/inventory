package main

import (
	"fmt"
	"io"
	"os"

	"ariga.io/atlas-provider-gorm/gormschema"

	awsmodels "github.com/gardener/inventory/pkg/aws/models"
)

func main() {
	models := []any{
		// AWS models
		&awsmodels.Region{},
		&awsmodels.AvailabilityZone{},
		&awsmodels.VPC{},
	}

	stmts, err := gormschema.New("postgres").Load(models...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load GORM schema: %v\n", err)
		os.Exit(1)
	}

	_, err = io.WriteString(os.Stdout, stmts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write statements: %v\n", err)
	}
}
