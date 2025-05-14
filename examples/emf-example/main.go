// Package main implements the tool.
package main

import "github.com/udhos/aws-emf/emf"

func main() {
	metric := emf.New(emf.Options{})

	dim1 := map[string]string{"dimKey1": "dimVal1"}
	dim2 := map[string]string{"dimKey1": "dimVal1", "dimKey2": "dimVal2"}

	metric1 := emf.MetricDefinition{
		Name:              "metric1",
		Unit:              "Bytes/Second",
		StorageResolution: 1,
	}

	metric2 := emf.MetricDefinition{
		Name: "metric2",
	}

	// Se você enviou antes alguma métrica que não vai sobrescrever com Record() agora, use o Reset().
	// Caso contrário, as métricas não sobrescritas serão reenvidas.
	metric.Reset()

	metric.Record("emf-test-ns1", metric1, nil, 10)  // métrica sem dimensões
	metric.Record("emf-test-ns1", metric1, dim1, 20) // métrica com 1 dimensão
	metric.Record("emf-test-ns1", metric1, dim2, 30) // métrica com 2 dimensões
	metric.Record("emf-test-ns1", metric2, nil, 40)  // outra métrica sem dimensões
	metric.Record("emf-test-ns2", metric1, nil, 50)  // métrica sem dimensões em outro namespace

	metric.Println()
}
