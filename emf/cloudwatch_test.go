package emf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// cloudWatchMock mocks client from package cloudwatchlogs.
type cloudWatchMock struct {
	dim map[string]namespace // namespace name => namespace
}

type namespace map[string]dimension // dimension key => dimension

type dimension map[string]met // metric name => met

type met struct {
	definition MetricDefinition
	value      int
}

// newCloudWatchMock creates mock client that implements method PutLogEvents
// from package cloudwatchlogs.
func newCloudWatchMock() *cloudWatchMock {
	return &cloudWatchMock{
		dim: map[string]namespace{},
	}
}

// PutLogEvents mocks method from package cloudwatchlogs.
func (c *cloudWatchMock) PutLogEvents(_ context.Context,
	params *cloudwatchlogs.PutLogEventsInput,
	_ ...func(*Options)) (*cloudwatchlogs.PutLogEventsOutput, error) {

	var output cloudwatchlogs.PutLogEventsOutput

	for _, e := range params.LogEvents {
		if err := c.putOne(e); err != nil {
			return &output, nil
		}
	}

	return &output, nil
}

func (c *cloudWatchMock) putOne(e types.InputLogEvent) error {
	root := map[string]any{}
	if err := json.Unmarshal([]byte(aws.ToString(e.Message)), &root); err != nil {
		return err
	}
	meta, hasMeta := root["_aws"]
	if !hasMeta {
		return errors.New("missing metadata field _aws")
	}
	data, errJSON := json.Marshal(meta)
	if errJSON != nil {
		return fmt.Errorf("error metadata json marshal: %v", errJSON)
	}
	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}
	for _, cwm := range metadata.CloudWatchMetrics {
		if err := c.addMetricDirective(root, cwm); err != nil {
			return err
		}
	}
	return nil
}

func (c *cloudWatchMock) addMetricDirective(root map[string]any, cwm *MetricDirective) error {
	namespaceName := cwm.Namespace
	var namespaceDimensionsKeys DimensionSet
	for _, set := range cwm.Dimensions {
		namespaceDimensionsKeys = append(namespaceDimensionsKeys, set...)
	}
	namespaceDimensions := map[string]string{}
	for _, key := range namespaceDimensionsKeys {
		val, hasDim := root[key]
		if !hasDim {
			return fmt.Errorf("missing dimension value for: %s", key)
		}
		dimVal, isStr := val.(string)
		if !isStr {
			return fmt.Errorf("dimension value not a string: %s: %#T: %v", key, val, val)
		}
		namespaceDimensions[key] = dimVal
	}
	for _, def := range cwm.Metrics {
		v, hasValue := root[def.Name]
		if !hasValue {
			return fmt.Errorf("missing metric value for: %s", def.Name)
		}
		val, isNum := v.(float64)
		if !isNum {
			return fmt.Errorf("metric value not a number: %s: %#T: %v", def.Name, v, v)
		}
		ns, foundNs := c.dim[namespaceName]
		if !foundNs {
			ns = namespace{}
			c.dim[namespaceName] = ns
		}
		dk := getDimensionKey(namespaceName, namespaceDimensions, namespaceDimensionsKeys)
		dim, foundDim := ns[dk]
		if !foundDim {
			dim = dimension{}
			ns[dk] = dim
		}
		m := met{
			definition: def,
			value:      int(val),
		}
		dim[def.Name] = m
	}
	return nil
}

// requireMetric defines metrics parameters required with method require.
type requireMetric struct {
	namespace        string
	dimensions       map[string]string
	metricName       string
	metricUnit       string
	metricResolution int
	metricValue      int
}

// require validates that a metrics sent with PutLogEvents is available.
func (c *cloudWatchMock) require(req requireMetric) error {
	ns, foundNs := c.dim[req.namespace]
	if !foundNs {
		return fmt.Errorf("namespace not found: %s", req.namespace)
	}
	keys := getDimensionSet(req.dimensions)
	dk := getDimensionKey(req.namespace, req.dimensions, keys)
	dim, foundDim := ns[dk]
	if !foundDim {
		return fmt.Errorf("dimensions not found: %v: %s", req.dimensions, dk)
	}
	m, foundMetric := dim[req.metricName]
	if !foundMetric {
		return fmt.Errorf("metric not found: %s", req.metricName)
	}
	if req.metricUnit != m.definition.Unit {
		return fmt.Errorf("metric unit: expected=%s got=%s", req.metricUnit, m.definition.Unit)
	}
	if req.metricResolution != m.definition.StorageResolution {
		return fmt.Errorf("metric resolution: expected=%d got=%d", req.metricResolution, m.definition.StorageResolution)
	}
	if req.metricValue != m.value {
		return fmt.Errorf("metric value: expected=%d got=%d", req.metricValue, m.value)
	}
	return nil
}
