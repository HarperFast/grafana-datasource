package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	harperdb "github.com/HarperDB-Add-Ons/sdk-go"
	"github.com/HarperDB/grafana-datasource/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
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

type queryModel struct {
	Operation  string              `json:"operation"`
	QueryAttrs map[string][]string `json:"queryAttrs"`
}

type harperDBRequest map[string]any

type sysInfoResponse struct {
	System map[string]json.RawMessage `json:"system"`
}

type searchByConditionsResponse struct {
	// TODO: Figure out how to unmarshal responses into this when appropriate
}

type harperDBResponse map[string]json.RawMessage

func (d *Datasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) (backend.DataResponse, error) {
	var response backend.DataResponse

	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.DataResponse{}, fmt.Errorf("could not unmarshal Grafana query JSON: '%s': '%w'", query.JSON, err)
	}

	// create data frame response.
	// For an overview on data frames and how grafana handles them:
	// https://grafana.com/developers/plugin-tools/introduction/data-frames
	frame := data.NewFrame("response")
	frame.RefID = query.RefID

	switch qm.Operation {
	case "system_information":
		sysInfoAttrs := qm.QueryAttrs["attributes"]
		var sysInfo *harperdb.SysInfo
		if len(sysInfoAttrs) > 0 {
			sysInfo, err = d.hdbClient.SystemInformation(sysInfoAttrs)
		} else {
			sysInfo, err = d.hdbClient.SystemInformationAll()
		}
		if err != nil {
			return backend.DataResponse{}, fmt.Errorf("could not get HarperDB system information: %w", err)
		}
		log.DefaultLogger.Debug("HarperDB system information", "sysInfo", sysInfo)

		frame.Fields = append(frame.Fields, models.SysInfoToFields(sysInfo, sysInfoAttrs)...)

		// TODO: Don't know if we need to do this, but it blows up on the duplicate field names
		//frame, err = data.LongToWide(frame, &data.FillMissing{
		//	Mode:  data.FillModePrevious,
		//	Value: 0,
		//})
		//if err != nil {
		//	return backend.DataResponse{}, fmt.Errorf("could not transform HarperDB system information response to wide format: %w", err)
		//}
	}

	// add the frames to the response.
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
