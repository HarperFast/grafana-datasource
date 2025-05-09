package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	harper "github.com/HarperDB/sdk-go"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"reflect"
	"sort"
	"strconv"
	"time"
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
	log.DefaultLogger.Info("QueryData", "request", req)

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
	SearchAttribute string       `json:"search_attribute"`
	SearchType      string       `json:"search_type"`
	SearchValue     SearchValue  `json:"search_value"`
	Operator        string       `json:"operator"`
	Conditions      []*Condition `json:"conditions"`
}

type SearchByConditionsQuery struct {
	Database   string      `json:"database"`
	Table      string      `json:"table"`
	Operator   string      `json:"operator"`
	Sort       SortVal     `json:"sort"`
	Attributes []string    `json:"attributes"`
	Conditions []Condition `json:"conditions"`
}

type GetAnalyticsQuery struct {
	Metric     string   `json:"metric"`
	Attributes []string `json:"attributes"`
	From       int64    `json:"from"`
	To         int64    `json:"to"`
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

func appendVal[T any](dest map[string]*data.Field, fieldName string, val T, fieldNils map[string]int) {
	if dest[fieldName] == nil {
		dest[fieldName] = data.NewField(fieldName, nil, []*T{})
		if fieldNils[fieldName] > 0 {
			for _ = range fieldNils[fieldName] {
				dest[fieldName].Append(nil)
			}
		}
	}
	dest[fieldName].Append(&val)
}

// flattenMap recursively converts nested Harper query result data structures
// into flat Grafana fields by appending prefixes to the field names
// reflecting the key path to that data in the original structure.
func flattenMap(source map[string]any, dest map[string]*data.Field, namePrefix string) {
	// keep track of initial nils so we can prepend those once we get a value that allows us to determine the field type
	fieldNils := make(map[string]int)

	for k, value := range source {
		fieldName := namePrefix + k

		switch v := value.(type) {
		case string:
			appendVal(dest, fieldName, v, fieldNils)
		case bool:
			appendVal(dest, fieldName, v, fieldNils)
		case float64:
			appendVal(dest, fieldName, v, fieldNils)
		case int64:
			appendVal(dest, fieldName, v, fieldNils)
		case time.Time:
			appendVal(dest, fieldName, v, fieldNils)
		case map[string]any:
			// For some reason neither this nor the []map[string]any cases
			// match when we seemingly have those types in the data. So I
			// implemented a reflection-based approach to catch them in the
			// default case below. - WSM 2025-01-09
			log.DefaultLogger.Debug("map found in query results", "map", v)
			//flattenMap(v, dest, fieldName+".")
		case []map[string]any:
			log.DefaultLogger.Debug("array of maps found in query results", "array", v)
			// See comment in map[string]any case.
			//for idx, val := range v {
			//	flattenMap(val, dest, fieldName+"."+strconv.Itoa(idx)+".")
			//}
		case nil:
			if dest[fieldName] == nil {
				fieldNils[fieldName] += 1
			} else {
				dest[fieldName].Append(nil)
			}
		default:
			// Sooo... this is a little hack-y and I don't love it. But when I
			// print out the type of maps and/or slices of maps coming from
			// query results, Go prints "map[]" which isn't a thing. It also
			// prints that for the key and element types of the map, which is
			// even weirder. So this reflection-based approach to catching and
			// handling these is a workaround until I figure out what's going
			// on there. - WSM 2025-01-09
			val := reflect.ValueOf(value)
			kind := val.Kind()
			switch kind {
			case reflect.Map:
				keyType := val.Type().Key()
				elemType := val.Type().Elem()
				log.DefaultLogger.Debug("Map types", "key", keyType, "elem", elemType)
				flattenMap(v.(map[string]any), dest, fieldName+".")
			case reflect.Slice:
				for i, x := range val.Interface().([]interface{}) {
					xv := reflect.ValueOf(x)
					log.DefaultLogger.Debug("xv", "Kind", xv.Kind())
					if xv.Kind() == reflect.Map {
						keyType := xv.Type().Key()
						elemType := xv.Type().Elem()
						log.DefaultLogger.Debug("Slice map types", "key", keyType, "elem", elemType)
						flattenMap(x.(map[string]any), dest, fieldName+"."+strconv.Itoa(i)+".")
					} else {
						if dest[fieldName] == nil {
							dest[fieldName] = data.NewField(fieldName, nil, v)
						} else {
							log.DefaultLogger.Warn("field name for slice already exists", "field", fieldName)
						}
					}
				}
			default:
				log.DefaultLogger.Warn("Unknown type in query response", "field", fieldName, "type", reflect.TypeOf(v), "value", v)
				val := reflect.ValueOf(value)
				log.DefaultLogger.Debug("val", "Kind", val.Kind())
			}
		}
	}
}

func (d *Datasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) (backend.DataResponse, error) {
	var response backend.DataResponse

	var qo queryOperation

	err := json.Unmarshal(query.JSON, &qo)
	if err != nil {
		return backend.DataResponse{}, fmt.Errorf("could not unmarshal Grafana query JSON: '%s': '%w'", query.JSON, err)
	}

	log.DefaultLogger.Debug("Query", "operation", qo)

	// create data frame response.
	// For an overview on data frames and how grafana handles them:
	// https://grafana.com/developers/plugin-tools/introduction/data-frames
	frame := data.NewFrame("response")
	frame.RefID = query.RefID

	switch qo.Operation {
	case "get_analytics":
		var qm queryModel[GetAnalyticsQuery]
		err := json.Unmarshal(query.JSON, &qm)
		if err != nil {
			return backend.DataResponse{}, fmt.Errorf("could not unmarshal get_analytics query JSON: '%s': '%w'", query.JSON, err)
		}
		request := qm.QueryAttrs
		log.DefaultLogger.Debug("Query", "request", request)

		metric := request.Metric

		getAttrs := request.Attributes
		if len(getAttrs) == 0 {
			getAttrs = []string{"*"}
		}

		results, err := d.harperClient.GetAnalytics(metric, getAttrs, request.From, request.To)
		if err != nil {
			return backend.DataResponse{}, fmt.Errorf("could not query Harper analytics: '%s': '%w'", query.JSON, err)
		}

		log.DefaultLogger.Debug("Get analytics query results", "results", results)

		mappedFields := make(map[string]*data.Field)
		for _, result := range results {
			log.DefaultLogger.Debug("get_analytics result fields", "count", len(result), "results", result)
			flattenMap(result, mappedFields, "")
		}

		fields := make(data.Fields, 0, len(mappedFields))
		// ensure stable sort order of fields; o/w Grafana changes their colors
		// and positions in the legend on every refresh
		keys := make([]string, 0, len(mappedFields))
		for k := range mappedFields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fields = append(fields, mappedFields[k])
		}

		frame.Fields = fields
	case "search_by_conditions":
		var qm queryModel[SearchByConditionsQuery]
		err := json.Unmarshal(query.JSON, &qm)
		if err != nil {
			return backend.DataResponse{}, fmt.Errorf("could not unmarshal search_by_conditions query JSON: '%s': '%w'", query.JSON, err)
		}
		search := qm.QueryAttrs

		conditions := make([]harper.SearchCondition, 0, len(search.Conditions))
		for _, condition := range search.Conditions {
			conditions = append(conditions, harper.SearchCondition{
				Attribute: condition.SearchAttribute,
				Type:      condition.SearchType,
				Value:     condition.SearchValue.Val,
				Operator:  condition.Operator,
			})
		}

		log.DefaultLogger.Debug("Query", "conditions", conditions)

		getAttrs := search.Attributes
		if len(getAttrs) == 0 {
			getAttrs = []string{"*"}
		}

		log.DefaultLogger.Debug("Query", "getAttrs", getAttrs)

		opts := harper.SearchByConditionsOptions{Operator: search.Operator}

		log.DefaultLogger.Debug("Query", "opts", opts)

		log.DefaultLogger.Debug("Query", "database", search.Database, "table", search.Table)
		results := make([]map[string]any, 0)
		err = d.harperClient.SearchByConditions(search.Database, search.Table, &results, conditions, getAttrs, opts)
		if err != nil {
			return backend.DataResponse{}, fmt.Errorf("error querying Harper: %w", err)
		}

		log.DefaultLogger.Debug("Query", "results", results)

		mappedFields := make(map[string]*data.Field)
		for _, result := range results {
			log.DefaultLogger.Debug("Result fields", "count", len(result), "results", result)
			flattenMap(result, mappedFields, "")
		}

		fields := make(data.Fields, 0, len(mappedFields))
		for _, field := range mappedFields {
			fields = append(fields, field)
		}

		frame.Fields = fields
	default:
		return backend.DataResponse{}, errors.New("unsupported Harper operation: " + qo.Operation)
	}

	response.Frames = append(response.Frames, frame)

	return response, nil
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
