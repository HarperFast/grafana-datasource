package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/HarperDB/grafana-datasource/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"net/http"
	"net/url"
	"strconv"
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
	d.httpClient.CloseIdleConnections()
}

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
	// TODO: Refactor this into multiple functions / files / packages
	harperURL := d.settings.OpsAPIURL
	username := d.settings.Username

	instanceSettings := pCtx.DataSourceInstanceSettings
	password, exists := instanceSettings.DecryptedSecureJSONData["password"]

	if !exists {
		return backend.DataResponse{}, fmt.Errorf("no password found for HarperDB connection '%s'", pCtx.PluginID)
	}

	var response backend.DataResponse

	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.DataResponse{}, fmt.Errorf("could not unmarshal Grafana query JSON: '%s': '%w'", query.JSON, err)
	}

	reqData := make(harperDBRequest)
	reqData["operation"] = qm.Operation
	for k, v := range qm.QueryAttrs {
		reqData[k] = v
	}
	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return backend.DataResponse{}, fmt.Errorf("could not marshal HarperDB request JSON: '%+v': '%w'", reqData, err)
	}

	log.DefaultLogger.Debug("HarperDB request", "body", string(reqBody))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, harperURL, bytes.NewReader(reqBody))
	if err != nil {
		return backend.DataResponse{}, fmt.Errorf("could not create HTTP request: '%s': '%w'", harperURL, err)
	}

	req.SetBasicAuth(username, password)

	req.Header.Add("Content-Type", "application/json")

	httpResp, err := d.httpClient.Do(req)
	switch {
	case err == nil:
		break
	case errors.Is(err, context.DeadlineExceeded):
		return backend.DataResponse{}, err
	default:
		return backend.DataResponse{}, fmt.Errorf("could not execute HTTP request: '%s': '%w'", req.URL, err)
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			log.DefaultLogger.Error("query: failed to close response body", "error", err)
		}
	}()

	if httpResp.StatusCode != http.StatusOK {
		return backend.DataResponse{}, fmt.Errorf("expected 200 response, got %d", httpResp.StatusCode)
	}

	// TODO: Unmarshalling to map[string]any here is just for debugging in dev; remove this later
	//var body map[string]any
	//if err := json.NewDecoder(httpResp.Body).Decode(&body); err != nil {
	//	return backend.DataResponse{}, fmt.Errorf("could not decode response body: %w", err)
	//}
	//log.DefaultLogger.Debug("Query", "response", body)

	// TODO: Stop hard-coding this type and figure out how to do this dynamically based on operation
	//var results sysInfoResponse
	//if err := json.NewDecoder(httpResp.Body).Decode(&results); err != nil {
	//	return backend.DataResponse{}, fmt.Errorf("could not decode response body: %w", err)
	//}

	var results harperDBResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&results); err != nil {
		return backend.DataResponse{}, fmt.Errorf("could not decode response body: %w", err)
	}

	// create data frame response.
	// For an overview on data frames and how grafana handles them:
	// https://grafana.com/developers/plugin-tools/introduction/data-frames
	frame := data.NewFrame("response")
	frame.RefID = query.RefID

	//frame.Fields = append(frame.Fields, data.NewField("system", nil, results.System))

	for resultKey, resultVal := range results {
		resultVals := make([]json.RawMessage, 1)
		resultVals[0] = resultVal
		frame.Fields = append(frame.Fields,
			data.NewField(resultKey, nil, resultVals),
		)
	}

	// add the frames to the response.
	response.Frames = append(response.Frames, frame)

	return response, nil
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
