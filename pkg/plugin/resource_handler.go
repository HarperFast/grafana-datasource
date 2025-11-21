package plugin

import (
	"encoding/json"
	harper "github.com/HarperDB/sdk-go"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"net/http"
	"slices"
)

type metricsHandler struct {
	datasource *Datasource
}

func newMetricsHandler(datasource *Datasource) *metricsHandler {
	return &metricsHandler{datasource: datasource}
}

func (d *Datasource) newResourceHandler() backend.CallResourceHandler {
	mux := http.NewServeMux()

	mh := newMetricsHandler(d)
	mux.Handle("/metrics", mh)
	mux.Handle("/metrics/{metric}", mh)

	return httpadapter.New(mux)
}

func (mh *metricsHandler) listMetrics(metricTypes []string) ([]harper.ListMetricsResult, error) {
	var metrics []harper.ListMetricsResult
	var err error

	var harperMetricTypes []harper.MetricType

	if len(metricTypes) == 0 {
		harperMetricTypes = []harper.MetricType{harper.MetricTypeBuiltin, harper.MetricTypeCustom}
	} else {
		if slices.Contains(metricTypes, "builtin") {
			harperMetricTypes = append(harperMetricTypes, harper.MetricTypeBuiltin)
		}
		if slices.Contains(metricTypes, "custom") {
			harperMetricTypes = append(harperMetricTypes, harper.MetricTypeCustom)
		}
	}

	metrics, err = mh.datasource.harperClient.ListMetrics(harperMetricTypes)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func (mh *metricsHandler) describeMetric(metric string) (*harper.DescribeMetricResult, error) {
	return mh.datasource.harperClient.DescribeMetric(metric)
}

func (mh *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metric := r.PathValue("metric")
	log.DefaultLogger.Debug("metricsHandler request", "urlPathValue", metric)
	log.DefaultLogger.Debug("metricsHandler request", "urlParams", r.URL.Query())

	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	var jsonResp []byte
	if metric == "" {
		metricTypes := r.URL.Query()["types"]
		metrics, err := mh.listMetrics(metricTypes)
		if err != nil {
			log.DefaultLogger.Error("failed to list metrics", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResp, err = json.Marshal(metrics)
		if err != nil {
			log.DefaultLogger.Error("error marshaling metrics to JSON", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		metricAttrs, err := mh.describeMetric(metric)
		if err != nil {
			log.DefaultLogger.Error("failed to describe metric", "metric", metric, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResp, err = json.Marshal(metricAttrs)
		if err != nil {
			log.DefaultLogger.Error("error marshaling metric attributes to JSON", "metric", metric, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")

	_, err := w.Write(jsonResp)
	if err != nil {
		log.DefaultLogger.Error("error writing response", "error", err)
	}
}
