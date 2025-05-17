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

	// If you previously sent a metric that now is not being overwritten with
	// Record(), call Reset() to drop all previous values. Otherwise any
	// non-overwritten metric is going to get reissued with the old value.
	metric.Reset()

	metric.Record("emf-test-ns1", metric1, nil, 10)  // metric without dimension
	metric.Record("emf-test-ns1", metric1, dim1, 20) // metric with 1 dimension
	metric.Record("emf-test-ns1", metric1, dim2, 30) // metric with 2 dimensions
	metric.Record("emf-test-ns1", metric2, nil, 40)  // another metric without dimension
	metric.Record("emf-test-ns2", metric1, nil, 50)  // metric without dimension but at another namespace

	metric.Println()
}
