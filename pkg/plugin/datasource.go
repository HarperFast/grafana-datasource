package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"net/http"
	"net/url"
	"strconv"
	"time"

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

func NewDatasource(ctx context.Context, s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	var settings Settings
	err := json.Unmarshal(s.JSONData, &settings)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling settings from JSON: %w", err)
	}

	opts, err := s.HTTPClientOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting http client options: %w", err)
	}

	client, err := httpclient.New(opts)
	if err != nil {
		return nil, fmt.Errorf("error creating http client: %w", err)
	}

	return &Datasource{
		settings:   settings,
		httpClient: client,
	}, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
	settings   Settings
	httpClient *http.Client
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct{}

func (d *Datasource) query(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	// create data frame response.
	// For an overview on data frames and how grafana handles them:
	// https://grafana.com/developers/plugin-tools/introduction/data-frames
	frame := data.NewFrame("response")

	// add fields.
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, []time.Time{query.TimeRange.From, query.TimeRange.To}),
		data.NewField("values", nil, []int64{10, 20}),
	)

	// add the frames to the response.
	response.Frames = append(response.Frames, frame)

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	res := &backend.CheckHealthResult{}
	config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)

	if err != nil {
		res.Status = backend.HealthStatusError
		res.Message = "Unable to load settings"
		return res, nil
	}

	if config.OpsAPIURL == "" {
		res.Status = backend.HealthStatusError
		res.Message = "Operations API URL is empty"
		return res, nil
	}

	if config.Username == "" {
		res.Status = backend.HealthStatusError
		res.Message = "Username is empty"
		return res, nil
	}

	if config.Secrets.Password == "" {
		res.Status = backend.HealthStatusError
		res.Message = "Password is missing"
		return res, nil
	}

	opsAPIURL, err := url.Parse(config.OpsAPIURL)
	if err != nil {
		res.Status = backend.HealthStatusError
		res.Message = "Invalid OpsAPI URL"
		return res, nil
	}

	healthCheckURL := opsAPIURL.JoinPath("/health")
	log.DefaultLogger.Info("Health check", "url", healthCheckURL)
	healthReq, err := http.NewRequestWithContext(ctx, http.MethodGet, healthCheckURL.String(), nil)
	if err != nil {
		res.Status = backend.HealthStatusError
		res.Message = "Could not create health check request: " + err.Error()
		return res, nil
	}

	healthResp, err := d.httpClient.Do(healthReq)
	if err != nil {
		res.Status = backend.HealthStatusError
		res.Message = "Could not execute health check request: " + err.Error()
		return res, nil
	}
	defer func() {
		if err := healthResp.Body.Close(); err != nil {
			log.DefaultLogger.Error("could not close health check response body", "error", err)
		}
	}()

	if healthResp.StatusCode != http.StatusOK {
		res.Status = backend.HealthStatusError
		res.Message = "Health check returned unexpected status code: " + strconv.Itoa(healthResp.StatusCode)
		return res, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}
