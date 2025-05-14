// Package main implements the tool.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/udhos/aws-emf/emf"
)

func main() {

	cfg, errConfig := config.LoadDefaultConfig(context.TODO())
	if errConfig != nil {
		log.Fatal(errConfig)
	}
	logsClient := cloudwatchlogs.NewFromConfig(cfg)
	logGroup := "emf-test"
	logStream := "emf-test"
	retentionInDays := int32(5)

	if _, errCreateGroup := logsClient.CreateLogGroup(context.TODO(), &cloudwatchlogs.CreateLogGroupInput{LogGroupName: aws.String(logGroup)}); errCreateGroup != nil {
		log.Print(errCreateGroup)
	}
	if _, errRetention := logsClient.PutRetentionPolicy(context.TODO(), &cloudwatchlogs.PutRetentionPolicyInput{LogGroupName: aws.String(logGroup), RetentionInDays: aws.Int32(retentionInDays)}); errRetention != nil {
		log.Print(errRetention)
	}
	if _, errCreateStream := logsClient.CreateLogStream(context.TODO(), &cloudwatchlogs.CreateLogStreamInput{LogGroupName: aws.String(logGroup), LogStreamName: aws.String(logStream)}); errCreateStream != nil {
		log.Print(errCreateStream)
	}

	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(logStream),
	}

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

	input.LogEvents = metric.CloudWatchLogEvents()

	_, err := logsClient.PutLogEvents(context.TODO(), input)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	fmt.Printf("*** enviado para CloudWatch Logs Group: %s ***\n", logGroup)
}
