package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	harper "github.com/HarperFast/sdk-go"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestQueryData(t *testing.T) {
	ds := Datasource{}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A"},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	if len(resp.Responses) != 1 {
		t.Fatal("QueryData must return a response")
	}
}

func TestTimeMillisUnmarshal(t *testing.T) {
	cases := []struct {
		name    string
		json    string
		want    int64
		wantErr bool
	}{
		{"number", `1700000000000`, 1700000000000, false},
		{"numeric string", `"1700000000000"`, 1700000000000, false},
		{"from macro", `"${__from}"`, 0, false},
		{"to macro", `"${__to}"`, 0, false},
		{"null", `null`, 0, false},
		{"empty string", `""`, 0, false},
		{"non-numeric string", `"yesterday"`, 0, true},
		{"object", `{}`, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got timeMillis
			err := json.Unmarshal([]byte(c.json), &got)
			if c.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if int64(got) != c.want {
				t.Fatalf("got %d, want %d", got, c.want)
			}
		})
	}
}

// Drives query() through the real Harper client against a fake ops API and asserts the
// start_time/end_time the plugin sends for each shape of from/to Grafana can produce.
func TestGetAnalyticsTimeBoundsSentToHarper(t *testing.T) {
	rangeFrom := time.UnixMilli(1700000000000)
	rangeTo := time.UnixMilli(1700000600000)

	cases := []struct {
		name       string
		queryAttrs string
		wantFrom   int64
		wantTo     int64
	}{
		{"explicit numbers win", `{"metric":"m","from":1600000000000,"to":1600000060000}`, 1600000000000, 1600000060000},
		{"uninterpolated macros default to request range", `{"metric":"m","from":"${__from}","to":"${__to}"}`, 1700000000000, 1700000600000},
		{"omitted defaults to request range", `{"metric":"m"}`, 1700000000000, 1700000600000},
		{"only to omitted", `{"metric":"m","from":1600000000000}`, 1600000000000, 1700000600000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got struct {
				StartTime int64 `json:"start_time"`
				EndTime   int64 `json:"end_time"`
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Errorf("decode request: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[]`))
			}))
			defer srv.Close()

			ds := &Datasource{harperClient: harper.NewClientWithHTTPClient(srv.Client(), srv.URL, "u", "p")}
			_, err := ds.query(context.Background(), backend.PluginContext{}, backend.DataQuery{
				RefID:     "A",
				TimeRange: backend.TimeRange{From: rangeFrom, To: rangeTo},
				JSON:      []byte(`{"operation":"get_analytics","queryAttrs":` + c.queryAttrs + `}`),
			})
			if err != nil {
				t.Fatalf("query: %v", err)
			}
			if got.StartTime != c.wantFrom || got.EndTime != c.wantTo {
				t.Fatalf("harper received start_time=%d end_time=%d, want %d/%d", got.StartTime, got.EndTime, c.wantFrom, c.wantTo)
			}
		})
	}
}
