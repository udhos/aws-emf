[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/aws-emf/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/aws-emf)](https://goreportcard.com/report/github.com/udhos/aws-emf)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/aws-emf.svg)](https://pkg.go.dev/github.com/udhos/aws-emf)

# aws-emf

This Go module [https://github.com/udhos/aws-emf](https://github.com/udhos/aws-emf) helps in utilizing the [AWS CloudWatch Embedded Metric Format](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/CloudWatch_Embedded_Metric_Format.html).

# Synopsis

```golang
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
```

# Examples

# Example issuing logs to stdout

[./examples/emf-example/main.go](./examples/emf-example/main.go)

# Example issuing logs to stdout in the format required by CLI (aws logs put-log-events)

```bash
emf-example-cw-cli > events

aws logs put-log-events --log-group-name my-logs --log-stream-name 20150601 --log-events file://events
```

[./examples/emf-example-cw-cli/main.go](./examples/emf-example-cw-cli/main.go)

# Example issuing logs directly to CloudWatch Log Group

[./examples/emf-example-cw-putlogevents/main.go](./examples/emf-example-cw-cli/main.go)
