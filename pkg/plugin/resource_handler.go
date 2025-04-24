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
	datasource          *Datasource
	builtinMetricsCache []harper.ListMetricsResult
	logger              log.Logger
}

func newMetricsHandler(datasource *Datasource) *metricsHandler {
	return &metricsHandler{datasource: datasource, builtinMetricsCache: []harper.ListMetricsResult{}}
}

func (d *Datasource) newResourceHandler() backend.CallResourceHandler {
	mux := http.NewServeMux()

	mh := newMetricsHandler(d)
	mux.Handle("/metrics", mh)
	mux.Handle("/metrics/{metric}", mh)

	return httpadapter.New(mux)
}

func (mh *metricsHandler) listBuiltinMetrics() ([]harper.ListMetricsResult, error) {
	if len(mh.builtinMetricsCache) > 0 {
		return mh.builtinMetricsCache, nil
	}

	metrics, err := mh.datasource.harperClient.ListMetrics([]harper.MetricType{harper.MetricTypeBuiltin})
	if err != nil {
		return nil, err
	}

	mh.builtinMetricsCache = metrics

	return metrics, nil
}

func (mh *metricsHandler) listCustomMetrics() ([]harper.ListMetricsResult, error) {
	// TODO: Decide if / how we want to cache these too
	// These are more resource intensive to query, but they also change a LOT
	// more often than the builtins

	metrics, err := mh.datasource.harperClient.ListMetrics([]harper.MetricType{harper.MetricTypeCustom})
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func (mh *metricsHandler) listMetrics(metricTypes []string) ([]harper.ListMetricsResult, error) {
	var metrics []harper.ListMetricsResult
	var err error

	if len(metricTypes) == 0 || slices.Contains(metricTypes, "builtin") {
		metrics, err = mh.listBuiltinMetrics()
		if err != nil {
			return nil, err
		}
	} else {
		metrics = make([]harper.ListMetricsResult, 0)
	}

	if len(metricTypes) == 0 || slices.Contains(metricTypes, "custom") {
		customMetrics, err := mh.listCustomMetrics()
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, customMetrics...)
	}

	return metrics, nil
}

func (mh *metricsHandler) describeMetric(metric string) (*harper.DescribeMetricResult, error) {
	return mh.datasource.harperClient.DescribeMetric(metric)
}

func (mh *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mh.logger = log.DefaultLogger.FromContext(r.Context())

	metric := r.PathValue("metric")
	mh.logger.Debug("metricsHandler request", "urlPathValue", metric)
	mh.logger.Debug("metricsHandler request", "urlParams", r.URL.Query())

	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	var jsonResp []byte
	if metric == "" {
		metricTypes := r.URL.Query()["types"]
		metrics, err := mh.listMetrics(metricTypes)
		if err != nil {
			mh.logger.Error("failed to list metrics", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResp, err = json.Marshal(metrics)
		if err != nil {
			mh.logger.Error("error marshaling metrics to JSON", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		metricAttrs, err := mh.describeMetric(metric)
		if err != nil {
			mh.logger.Error("failed to describe metric", "metric", metric, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResp, err = json.Marshal(metricAttrs)
		if err != nil {
			mh.logger.Error("error marshaling metric attributes to JSON", "metric", metric, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")

	_, err := w.Write(jsonResp)
	if err != nil {
		mh.logger.Error("error writing response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
