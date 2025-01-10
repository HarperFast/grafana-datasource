package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	harperdb "github.com/HarperDB-Add-Ons/sdk-go"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"reflect"
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
		return backend.DataResponse{}, fmt.Errorf("no password found for HarperDB connection")
	}

	client := harperdb.NewClient(settings.OpsAPIURL, settings.Username, password)

	return &Datasource{
		settings:  settings,
		hdbClient: client,
	}, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
	settings  Settings
	hdbClient *harperdb.Client
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

type Condition struct {
	SearchAttribute string       `json:"search_attribute"`
	SearchType      string       `json:"search_type"`
	SearchValue     any          `json:"search_value"`
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

type queryModel struct {
	Operation  string                  `json:"operation"`
	QueryAttrs SearchByConditionsQuery `json:"queryAttrs"`
}

// flattenMap recursively converts nested HDB query result data structures
// into flat Grafana fields by appending prefixes to the field names
// reflecting the key path to that data in the original structure.
func flattenMap(source map[string]any, dest map[string]*data.Field, namePrefix string) {
	for name, value := range source {
		fieldName := namePrefix + name

		switch v := value.(type) {
		// TODO: Add more supported types
		case string:
			if dest[fieldName] == nil {
				dest[fieldName] = data.NewField(fieldName, nil, []string{})
			}
			dest[fieldName].Append(v)
		case bool:
			if dest[fieldName] == nil {
				dest[fieldName] = data.NewField(fieldName, nil, []bool{})
			}
			dest[fieldName].Append(v)
		case float64:
			if dest[fieldName] == nil {
				dest[fieldName] = data.NewField(fieldName, nil, []float64{})
			}
			dest[fieldName].Append(v)
		case int64:
			if dest[fieldName] == nil {
				dest[fieldName] = data.NewField(fieldName, nil, []int64{})
			}
			dest[fieldName].Append(v)
		case time.Time:
			// TODO: I don't think this case is currently ever hit because we
			// don't explicitly unmarshal any fields from JSON to time.Time.
			// So this requires transforming the field type on the Grafana
			// side for time series data. This works but is perhaps a little
			// annoying for users. Need to think through whether or not there's
			// a better approach to have time fields come through as time.Time
			// so that transformation isn't necessary. - WSM 2025-01-09
			if dest[fieldName] == nil {
				dest[fieldName] = data.NewField(fieldName, nil, []time.Time{})
			}
			dest[fieldName].Append(v)
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

	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.DataResponse{}, fmt.Errorf("could not unmarshal Grafana query JSON: '%s': '%w'", query.JSON, err)
	}

	log.DefaultLogger.Debug("Query", "query", qm)

	// create data frame response.
	// For an overview on data frames and how grafana handles them:
	// https://grafana.com/developers/plugin-tools/introduction/data-frames
	frame := data.NewFrame("response")
	frame.RefID = query.RefID

	switch qm.Operation {
	case "search_by_conditions":
		search := qm.QueryAttrs

		conditions := make([]harperdb.SearchCondition, 0)
		for _, condition := range search.Conditions {
			conditions = append(conditions, harperdb.SearchCondition{
				Attribute: condition.SearchAttribute,
				Type:      condition.SearchType,
				Value:     condition.SearchValue,
				Operator:  condition.Operator,
			})
		}

		log.DefaultLogger.Debug("Query", "conditions", conditions)

		getAttrs := search.Attributes
		if len(getAttrs) == 0 {
			getAttrs = []string{"*"}
		}

		log.DefaultLogger.Debug("Query", "getAttrs", getAttrs)

		opts := harperdb.SearchByConditionsOptions{Operator: search.Operator}

		log.DefaultLogger.Debug("Query", "opts", opts)

		log.DefaultLogger.Debug("Query", "database", search.Database, "table", search.Table)
		results := make([]map[string]any, 0)
		err := d.hdbClient.SearchByConditions(search.Database, search.Table, &results, conditions, getAttrs, opts)
		if err != nil {
			return backend.DataResponse{}, fmt.Errorf("error querying HarperDB: %w", err)
		}

		log.DefaultLogger.Debug("Query", "results", results)

		mappedFields := make(map[string]*data.Field)
		for _, result := range results {
			log.DefaultLogger.Debug("Result fields", "count", len(result), "result", result)
			flattenMap(result, mappedFields, "")
		}

		fields := make(data.Fields, 0)
		for _, field := range mappedFields {
			fields = append(fields, field)
		}

		frame.Fields = fields
	default:
		return backend.DataResponse{}, errors.New("unsupported HDB operation: " + qm.Operation)
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

	err := d.hdbClient.Healthcheck()
	if err != nil {
		res.Status = backend.HealthStatusError
		var opErr *harperdb.OperationError
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
