package emf

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

// go test -v -count 1 -run '^TestEmpty$' ./emf
func TestEmpty(t *testing.T) {

	metric := New(Options{})

	list := metric.Render()

	if len(list) != 0 {
		t.Errorf("list size: expected=0 got=%d", len(list))
	}
}

func TestReset(t *testing.T) {

	metric := New(Options{})

	dim1 := map[string]string{"dimKey1": "dimVal1"}
	dim2 := map[string]string{"dimKey2": "dimVal2"}

	metric1 := MetricDefinition{
		Name:              "speed1",
		Unit:              "Bytes/Second",
		StorageResolution: 1,
	}

	metric.Record("emf-test-ns1", metric1, dim1, 100)
	metric.Record("emf-test-ns1", metric1, dim1, 110)
	metric.Record("emf-test-ns1", metric1, dim2, 50)

	{
		countMetrics, countDimensions := metric.count()
		if countMetrics != 2 {
			t.Fatalf("metrics expected=%d got=%d", 2, countMetrics)
		}
		if countDimensions != 2 {
			t.Fatalf("metrics expected=%d got=%d", 2, countDimensions)
		}
	}

	metric.Reset()

	{
		countMetrics, countDimensions := metric.count()
		if countMetrics != 0 {
			t.Fatalf("metrics expected=%d got=%d", 0, countMetrics)
		}
		if countDimensions != 0 {
			t.Fatalf("metrics expected=%d got=%d", 0, countDimensions)
		}
	}
}

func TestMeta(t *testing.T) {

	metric := New(Options{})

	dim1 := map[string]string{"dimKey1": "dimVal1"}
	dim2 := map[string]string{"dimKey2": "dimVal2"}

	metric1 := MetricDefinition{
		Name:              "speed1",
		Unit:              "Bytes/Second",
		StorageResolution: 1,
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, dim1, 100)
	metric.Record("emf-test-ns1", metric1, dim1, 110)
	metric.Record("emf-test-ns1", metric1, dim2, 50)

	list := metric.Render()
	data := list[0]

	metaFound, err := hasMeta([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if !metaFound {
		t.Fatalf("meta not found: %s", string(data))
	}
	t.Logf("has meta ok")
}

func hasMeta(data []byte) (bool, error) {

	{
		var m struct {
			Meta Metadata `json:"_aws"`
		}
		if err := json.Unmarshal(data, &m); err != nil {
			return false, err
		}
	}

	{
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return false, err
		}
		metadata, found := m["_aws"]
		if !found {
			return false, errors.New("missing _aws")
		}
		metadataMap, isMap := metadata.(map[string]interface{})
		if !isMap {
			return false, errors.New("_aws value is not a map")
		}
		if _, foundCloudWatchMetrics := metadataMap["CloudWatchMetrics"]; !foundCloudWatchMetrics {
			return false, errors.New("missing _aws.CloudWatchMetrics")
		}
		if _, foundTimestamp := metadataMap["Timestamp"]; !foundTimestamp {
			return false, errors.New("missing _aws.Timestamp")
		}
	}

	return true, nil
}

// go test -v -count 1 -run '^TestOneMetric$' ./emf
func TestOneMetric(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, nil, 100)
	list := metric.Render()
	data := list[0]

	t.Logf("output: %s", data)

	const expect = `{"_aws":{"CloudWatchMetrics":[{"Namespace":"emf-test-ns1","Dimensions":[],"Metrics":[{"Name":"speed1"}]}],"Timestamp":0},"speed1":100}`
	if expect != data {
		t.Fatalf("expected=%s got=%s", expect, data)
	}

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      100,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestOneMetricChangeOnce$' ./emf
func TestOneMetricChangeOnce(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, nil, 100)
	metric.Record("emf-test-ns1", metric1, nil, 101)
	list := metric.Render()
	data := list[0]

	t.Logf("output: %s", data)

	const expect = `{"_aws":{"CloudWatchMetrics":[{"Namespace":"emf-test-ns1","Dimensions":[],"Metrics":[{"Name":"speed1"}]}],"Timestamp":0},"speed1":101}`
	if expect != data {
		t.Fatalf("expected=%s got=%s", expect, data)
	}

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      101,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestOneMetricChangeTwice$' ./emf
func TestOneMetricChangeTwice(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, nil, 100)
	metric.Record("emf-test-ns1", metric1, nil, 101)
	metric.Record("emf-test-ns1", metric1, nil, 102)
	list := metric.Render()
	data := list[0]

	t.Logf("output: %s", data)

	const expect = `{"_aws":{"CloudWatchMetrics":[{"Namespace":"emf-test-ns1","Dimensions":[],"Metrics":[{"Name":"speed1"}]}],"Timestamp":0},"speed1":102}`
	if expect != data {
		t.Fatalf("expected=%s got=%s", expect, data)
	}

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      102,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestTwoMetrics$' ./emf
func TestTwoMetrics(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric2 := MetricDefinition{
		Name: "speed2",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, nil, 100)
	metric.Record("emf-test-ns1", metric2, nil, 50)
	list := metric.Render()
	data := list[0]

	t.Logf("output: %s", data)

	const expect = `{"_aws":{"CloudWatchMetrics":[{"Namespace":"emf-test-ns1","Dimensions":[],"Metrics":[{"Name":"speed1"},{"Name":"speed2"}]}],"Timestamp":0},"speed1":100,"speed2":50}`
	if expect != data {
		t.Fatalf("expected=%s got=%s", expect, data)
	}

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      100,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "speed2",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      50,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestTwoMetricsChangeOne$' ./emf
func TestTwoMetricsChangeOne(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric2 := MetricDefinition{
		Name: "speed2",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, nil, 100)
	metric.Record("emf-test-ns1", metric1, nil, 101)
	metric.Record("emf-test-ns1", metric2, nil, 50)
	list := metric.Render()
	data := list[0]

	t.Logf("output: %s", data)

	const expect = `{"_aws":{"CloudWatchMetrics":[{"Namespace":"emf-test-ns1","Dimensions":[],"Metrics":[{"Name":"speed1"},{"Name":"speed2"}]}],"Timestamp":0},"speed1":101,"speed2":50}`
	if expect != data {
		t.Fatalf("expected=%s got=%s", expect, data)
	}

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      101,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "speed2",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      50,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestTwoMetricsChangeBoth$' ./emf
func TestTwoMetricsChangeBoth(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric2 := MetricDefinition{
		Name: "speed2",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, nil, 100)
	metric.Record("emf-test-ns1", metric1, nil, 101)
	metric.Record("emf-test-ns1", metric2, nil, 50)
	metric.Record("emf-test-ns1", metric2, nil, 51)
	list := metric.Render()
	data := list[0]

	t.Logf("output: %s", data)

	const expect = `{"_aws":{"CloudWatchMetrics":[{"Namespace":"emf-test-ns1","Dimensions":[],"Metrics":[{"Name":"speed1"},{"Name":"speed2"}]}],"Timestamp":0},"speed1":101,"speed2":51}`
	if expect != data {
		t.Fatalf("expected=%s got=%s", expect, data)
	}

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      101,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "speed2",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      51,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestOneMetricWithUnitAndResolution$' ./emf
func TestOneMetricWithUnitAndResolution(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	metric1 := MetricDefinition{
		Name:              "speed1",
		Unit:              "Bytes/Second",
		StorageResolution: 1,
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, nil, 100)
	list := metric.Render()
	data := list[0]

	t.Logf("output: %s", data)

	const expect = `{"_aws":{"CloudWatchMetrics":[{"Namespace":"emf-test-ns1","Dimensions":[],"Metrics":[{"Name":"speed1","Unit":"Bytes/Second","StorageResolution":1}]}],"Timestamp":0},"speed1":100}`
	if expect != data {
		t.Fatalf("expected=%s got=%s", expect, data)
	}

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "speed1",
		metricUnit:       "Bytes/Second",
		metricResolution: 1,
		metricValue:      100,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestOneMetricWithDimension$' ./emf
func TestOneMetricWithDimension(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	dim1 := map[string]string{"dimKey1": "dimVal1"}

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, dim1, 100)
	list := metric.Render()
	data := list[0]

	t.Logf("output: %s", data)

	const expect = `{"_aws":{"CloudWatchMetrics":[{"Namespace":"emf-test-ns1","Dimensions":[["dimKey1"]],"Metrics":[{"Name":"speed1"}]}],"Timestamp":0},"dimKey1":"dimVal1","speed1":100}`
	if expect != data {
		t.Fatalf("expected=%s got=%s", expect, data)
	}

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey1": "dimVal1"},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      100,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestOneMetricWithTwoDimensions$' ./emf
func TestOneMetricWithTwoDimensions(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	dim := map[string]string{"dimKey1": "dimVal1", "dimKey2": "dimVal2"}

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, dim, 100)
	list := metric.Render()
	data := list[0]

	t.Logf("output: %s", data)

	const expect = `{"_aws":{"CloudWatchMetrics":[{"Namespace":"emf-test-ns1","Dimensions":[["dimKey1","dimKey2"]],"Metrics":[{"Name":"speed1"}]}],"Timestamp":0},"dimKey1":"dimVal1","dimKey2":"dimVal2","speed1":100}`
	if expect != data {
		t.Fatalf("expected=%s got=%s", expect, data)
	}

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey1": "dimVal1", "dimKey2": "dimVal2"},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      100,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestTwoMetricsWithSameDimensions$' ./emf
func TestTwoMetricsWithSameDimensions(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	dim := map[string]string{"dimKey1": "dimVal1", "dimKey2": "dimVal2"}

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric2 := MetricDefinition{
		Name: "speed2",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, dim, 100)
	metric.Record("emf-test-ns1", metric2, dim, 50)
	list := metric.Render()
	data := list[0]

	t.Logf("output: %s", data)

	const expect = `{"_aws":{"CloudWatchMetrics":[{"Namespace":"emf-test-ns1","Dimensions":[["dimKey1","dimKey2"]],"Metrics":[{"Name":"speed1"},{"Name":"speed2"}]}],"Timestamp":0},"dimKey1":"dimVal1","dimKey2":"dimVal2","speed1":100,"speed2":50}`
	if expect != data {
		t.Fatalf("expected=%s got=%s", expect, data)
	}

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey1": "dimVal1", "dimKey2": "dimVal2"},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      100,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey1": "dimVal1", "dimKey2": "dimVal2"},
		metricName:       "speed2",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      50,
	}); errRequire != nil {
		t.Fatalf("require error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestTwoMetricsWithSameDimensionsAndDistinctNamespaces$' ./emf
func TestTwoMetricsWithSameDimensionsAndDistinctNamespaces(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	dim := map[string]string{"dimKey1": "dimVal1", "dimKey2": "dimVal2"}

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric2 := MetricDefinition{
		Name: "speed2",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, dim, 100)
	metric.Record("emf-test-ns2", metric2, dim, 50)

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey1": "dimVal1", "dimKey2": "dimVal2"},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      100,
	}); errRequire != nil {
		t.Fatalf("require emf-test-ns1 error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns2",
		dimensions:       map[string]string{"dimKey1": "dimVal1", "dimKey2": "dimVal2"},
		metricName:       "speed2",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      50,
	}); errRequire != nil {
		t.Fatalf("require emf-test-ns2 error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestSameMetricsTwiceWithDistinctDimensions$' ./emf
func TestSameMetricsTwiceWithDistinctDimensions(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	dim1 := map[string]string{"dimKey1": "dimVal1"}
	dim2 := map[string]string{"dimKey2": "dimVal2"}

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, dim1, 100)
	metric.Record("emf-test-ns1", metric1, dim2, 50)

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey1": "dimVal1"},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      100,
	}); errRequire != nil {
		t.Fatalf("require speed1 error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey2": "dimVal2"},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      50,
	}); errRequire != nil {
		t.Fatalf("require speed2 error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestSameMetricsTwiceWithDistinctDimensionsChangeBoth$' ./emf
func TestSameMetricsTwiceWithDistinctDimensionsChangeBoth(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	dim1 := map[string]string{"dimKey1": "dimVal1"}
	dim2 := map[string]string{"dimKey2": "dimVal2"}

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, dim1, 100)
	metric.Record("emf-test-ns1", metric1, dim2, 50)
	metric.Record("emf-test-ns1", metric1, dim1, 101)
	metric.Record("emf-test-ns1", metric1, dim2, 51)

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey1": "dimVal1"},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      101,
	}); errRequire != nil {
		t.Fatalf("require speed1 error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey2": "dimVal2"},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      51,
	}); errRequire != nil {
		t.Fatalf("require speed2 error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestSameMetricWithSameDimensionsWithDistinctNamespaces$' ./emf
func TestSameMetricWithSameDimensionsWithDistinctNamespaces(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	dim1 := map[string]string{"dimKey1": "dimVal1"}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, dim1, 100)
	metric.Record("emf-test-ns2", metric1, dim1, 50)

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey1": "dimVal1"},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      100,
	}); errRequire != nil {
		t.Fatalf("require speed1 error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns2",
		dimensions:       map[string]string{"dimKey1": "dimVal1"},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      50,
	}); errRequire != nil {
		t.Fatalf("require speed2 error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestSameMetricWithoutDimensionsWithDistinctNamespaces$' ./emf
func TestSameMetricWithoutDimensionsWithDistinctNamespaces(t *testing.T) {

	metric := New(Options{UnixMilli: func() int64 { return 0 }})

	metric1 := MetricDefinition{
		Name: "speed1",
	}

	metric.Reset()
	metric.Record("emf-test-ns1", metric1, nil, 100)
	metric.Record("emf-test-ns2", metric1, nil, 50)

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      100,
	}); errRequire != nil {
		t.Fatalf("require speed1 error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns2",
		dimensions:       map[string]string{},
		metricName:       "speed1",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      50,
	}); errRequire != nil {
		t.Fatalf("require speed2 error: %v", errRequire)
	}
}

// go test -v -count 1 -run '^TestCloudWatchSendExample$' ./emf
func TestCloudWatchSendExample(t *testing.T) {

	metric := New(Options{})

	dim1 := map[string]string{"dimKey1": "dimVal1"}
	dim2 := map[string]string{"dimKey1": "dimVal1", "dimKey2": "dimVal2"}

	metric1 := MetricDefinition{
		Name:              "metric1",
		Unit:              "Bytes/Second",
		StorageResolution: 1,
	}

	metric2 := MetricDefinition{
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

	cw := newCloudWatchMock()
	input := &cloudwatchlogs.PutLogEventsInput{}
	input.LogEvents = metric.CloudWatchLogEvents()
	_, err := cw.PutLogEvents(context.TODO(), input)
	if err != nil {
		t.Fatalf("PutLogEvents error: %v", err)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "metric1",
		metricUnit:       "Bytes/Second",
		metricResolution: 1,
		metricValue:      10,
	}); errRequire != nil {
		t.Fatalf("1 - require ns1/metric1 error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey1": "dimVal1"},
		metricName:       "metric1",
		metricUnit:       "Bytes/Second",
		metricResolution: 1,
		metricValue:      20,
	}); errRequire != nil {
		t.Fatalf("2 - require ns1/metric1/dim1 error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{"dimKey1": "dimVal1", "dimKey2": "dimVal2"},
		metricName:       "metric1",
		metricUnit:       "Bytes/Second",
		metricResolution: 1,
		metricValue:      30,
	}); errRequire != nil {
		t.Fatalf("3 - require ns1/metric1/dim2 error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns1",
		dimensions:       map[string]string{},
		metricName:       "metric2",
		metricUnit:       "",
		metricResolution: 0,
		metricValue:      40,
	}); errRequire != nil {
		t.Fatalf("4 - require ns1/metric2 error: %v", errRequire)
	}
	if errRequire := cw.require(requireMetric{
		namespace:        "emf-test-ns2",
		dimensions:       map[string]string{},
		metricName:       "metric1",
		metricUnit:       "Bytes/Second",
		metricResolution: 1,
		metricValue:      50,
	}); errRequire != nil {
		t.Fatalf("5 - require ns2/metric1 error: %v", errRequire)
	}
}
