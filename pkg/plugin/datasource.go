package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"time"

	harper "github.com/HarperDB/sdk-go"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces - only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
	_ backend.CallResourceHandler   = (*Datasource)(nil)
)

type Settings struct {
	OpsAPIURL string `json:"opsAPIURL"`
	Username  string `json:"username"`
}

func NewDatasource(_ context.Context, s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	var settings Settings
	err := json.Unmarshal(s.JSONData, &settings)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling settings from JSON: %w", err)
	}

	password, exists := s.DecryptedSecureJSONData["password"]
	if !exists {
		return backend.DataResponse{}, fmt.Errorf("no password found for Harper connection")
	}

	client := harper.NewClient(settings.OpsAPIURL, settings.Username, password)

	ds := &Datasource{
		settings:     settings,
		harperClient: client,
	}
	resourceHandler := ds.newResourceHandler()
	ds.CallResourceHandler = resourceHandler
	return ds, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
	settings Settings
	backend.CallResourceHandler
	harperClient *harper.Client
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res, err := d.query(ctx, req.PluginContext, q)
		if err != nil {
			response.Responses[q.RefID] = backend.ErrDataResponse(backend.StatusBadRequest, err.Error())
		} else {
			response.Responses[q.RefID] = res
		}
	}

	return response, nil
}

type SortVal struct {
	Attribute  string   `json:"attribute"`
	Descending bool     `json:"descending"`
	Next       *SortVal `json:"next"`
}

type SearchValue struct {
	Val  any    `json:"val"`
	Type string `json:"type"`
}

type Condition struct {
	Attribute  string       `json:"attribute"`
	Comparator string       `json:"comparator"`
	Value      SearchValue  `json:"value"`
	Operator   string       `json:"operator"`
	Conditions []*Condition `json:"conditions"`
}

type Conditions []*Condition

type SearchByConditionsQuery struct {
	Database   string     `json:"database"`
	Table      string     `json:"table"`
	Operator   string     `json:"operator"`
	Sort       SortVal    `json:"sort"`
	Attributes []string   `json:"attributes"`
	Conditions Conditions `json:"conditions"`
}

type GetAnalyticsQuery struct {
	Metric     string     `json:"metric"`
	Attributes []string   `json:"attributes"`
	From       int64      `json:"from"`
	To         int64      `json:"to"`
	Conditions Conditions `json:"conditions"`
}

type Query interface {
	SearchByConditionsQuery | GetAnalyticsQuery
}

type queryOperation struct {
	Operation string `json:"operation"`
}

type queryModel[Q Query] struct {
	Operation  string `json:"operation"`
	QueryAttrs Q      `json:"queryAttrs"`
}

type analyticsTable struct {
	Headers    []string
	FieldTypes []data.FieldType
	Rows       [][]any
}

func (d *Datasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) (backend.DataResponse, error) {
	var response backend.DataResponse

	var qo queryOperation

	err := json.Unmarshal(query.JSON, &qo)
	if err != nil {
		return backend.DataResponse{}, fmt.Errorf("could not unmarshal Grafana query JSON: '%s': '%w'", query.JSON, err)
	}

	switch qo.Operation {
	case "get_analytics":
		var qm queryModel[GetAnalyticsQuery]
		err := json.Unmarshal(query.JSON, &qm)
		if err != nil {
			return backend.DataResponse{}, fmt.Errorf("could not unmarshal get_analytics query JSON: '%s': '%w'", query.JSON, err)
		}
		request := qm.QueryAttrs

		conditions := make(harper.SearchConditions, 0)
		for _, c := range request.Conditions {
			conditions = append(conditions, harper.SearchCondition{
				Attribute:  c.Attribute,
				Comparator: c.Comparator,
				Value:      c.Value.Val,
			})
		}

		req := harper.GetAnalyticsRequest{
			Metric:        request.Metric,
			GetAttributes: request.Attributes,
			StartTime:     request.From,
			EndTime:       request.To,
		}
		if len(conditions) > 0 {
			req.Conditions = conditions
		}

		results, err := d.harperClient.GetAnalytics(req)
		if err != nil {
			return backend.DataResponse{}, fmt.Errorf("could not query Harper analytics: '%s': '%w'", query.JSON, err)
		}

		// Collect the superset of all fields in the results.
		// Grafana gets very cranky if any rows have a different set of fields (columns), so we have to make sure they
		// all have all of them.
		allFields := make(map[string]bool)
		for _, result := range results {
			for k := range result {
				if k != "metric" {
					allFields[k] = true
				}
			}
		}

		headers := slices.Collect(maps.Keys(allFields))
		// Sort the header names so they don't get jumbled on every Grafana refresh
		sort.Strings(headers)

		grafanaAnalytics := analyticsTable{
			Headers: headers,
		}

		grafanaAnalytics.FieldTypes = make([]data.FieldType, len(grafanaAnalytics.Headers))

		for _, result := range results {
			row := make([]any, len(grafanaAnalytics.Headers))
			for i, header := range grafanaAnalytics.Headers {
				val, ok := result[header]
				if !ok {
					row[i] = nil
				} else {
					switch v := val.(type) {
					case string:
						if grafanaAnalytics.FieldTypes[i] == data.FieldTypeUnknown {
							grafanaAnalytics.FieldTypes[i] = data.FieldTypeNullableString
						}
						row[i] = &v
					case bool:
						if grafanaAnalytics.FieldTypes[i] == data.FieldTypeUnknown {
							grafanaAnalytics.FieldTypes[i] = data.FieldTypeNullableBool
						}
						row[i] = &v
					case float64:
						if grafanaAnalytics.FieldTypes[i] == data.FieldTypeUnknown {
							grafanaAnalytics.FieldTypes[i] = data.FieldTypeNullableFloat64
						}
						row[i] = &v
					case int64:
						if grafanaAnalytics.FieldTypes[i] == data.FieldTypeUnknown {
							grafanaAnalytics.FieldTypes[i] = data.FieldTypeNullableInt64
						}
						row[i] = &v
					case time.Time:
						if grafanaAnalytics.FieldTypes[i] == data.FieldTypeUnknown {
							grafanaAnalytics.FieldTypes[i] = data.FieldTypeNullableTime
						}
						row[i] = &v
					case nil:
						if grafanaAnalytics.FieldTypes[i] == data.FieldTypeUnknown {
							grafanaAnalytics.FieldTypes[i] = data.FieldTypeNullableString
						}
						row[i] = nil
					}
				}
			}
			grafanaAnalytics.Rows = append(grafanaAnalytics.Rows, row)
		}

		frame := data.NewFrameOfFieldTypes(
			"response", 0,
			grafanaAnalytics.FieldTypes...,
		).SetMeta(
			&data.FrameMeta{
				Type:        data.FrameTypeTimeSeriesLong,
				TypeVersion: data.FrameTypeVersion{0, 1},
			},
		).SetRefID(query.RefID)

		err = frame.SetFieldNames(grafanaAnalytics.Headers...)
		if err != nil {
			return backend.DataResponse{}, fmt.Errorf("could not set field names on frame: '%s': '%w'", query.JSON, err)
		}

		for _, row := range grafanaAnalytics.Rows {
			frame.AppendRow(row...)
		}

		if frame.Rows() == 0 {
			// early return here so we don't get an error about being unable to convert to wide format
			response.Frames = append(response.Frames, frame)
			return response, nil
		}

		wideFrame, err := data.LongToWide(frame, &data.FillMissing{Mode: data.FillModeNull})
		if err != nil {
			return backend.DataResponse{}, fmt.Errorf("could not convert frame to wide format: '%w'", err)
		}

		response.Frames = append(response.Frames, wideFrame)
		return response, nil
	default:
		return backend.DataResponse{}, errors.New("unsupported Harper operation: " + qo.Operation)
	}
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	res := &backend.CheckHealthResult{}

	err := d.harperClient.Healthcheck()
	if err != nil {
		res.Status = backend.HealthStatusError
		var opErr *harper.OperationError
		if errors.As(err, &opErr) {
			res.Message = fmt.Sprintf("Health check returned unexpected status code: '%d' with message: '%s'",
				opErr.StatusCode, opErr.Message)
		} else {
			res.Message = "Health check returned unexpected error: " + err.Error()
		}
		return res, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}
