package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	harper "github.com/HarperFast/sdk-go"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
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

func TestTimeBoundUnmarshal(t *testing.T) {
	cases := []struct {
		name    string
		json    string
		want    timeBound
		wantErr string
	}{
		{"number", `1700000000000`, timeBound{millis: 1700000000000}, ""},
		{"numeric string", `"1700000000000"`, timeBound{millis: 1700000000000}, ""},
		{"padded numeric string", `" 1700000000000 "`, timeBound{millis: 1700000000000}, ""},
		{"from macro", `"${__from}"`, timeBound{macro: rangeFrom}, ""},
		{"to macro", `"${__to}"`, timeBound{macro: rangeTo}, ""},
		{"from macro without braces", `"$__from"`, timeBound{macro: rangeFrom}, ""},
		{"to macro without braces", `"$__to"`, timeBound{macro: rangeTo}, ""},
		{"null", `null`, timeBound{}, ""},
		{"empty string", `""`, timeBound{}, ""},
		{"zero", `0`, timeBound{}, "positive"},
		{"negative", `-5`, timeBound{}, "positive"},
		{"fraction", `1700000000000.5`, timeBound{}, "whole unix milliseconds"},
		{"overflow", `99999999999999999999`, timeBound{}, "whole unix milliseconds"},
		{"boolean", `true`, timeBound{}, "whole unix milliseconds"},
		{"array", `[1700000000000]`, timeBound{}, "whole unix milliseconds"},
		{"object", `{}`, timeBound{}, "whole unix milliseconds"},
		{"other macro", `"${__interval}"`, timeBound{}, "whole unix milliseconds"},
		{"misspelt macro", `"${__fron}"`, timeBound{}, "whole unix milliseconds"},
		{"non-numeric string", `"yesterday"`, timeBound{}, "whole unix milliseconds"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got timeBound
			err := json.Unmarshal([]byte(c.json), &got)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("err = %v", err)
				}
				if got != c.want {
					t.Fatalf("got %+v, want %+v", got, c.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("err = %v, want one containing %q", err, c.wantErr)
			}
		})
	}
}

type fakeHarper struct {
	srv      *httptest.Server
	requests atomic.Int32
	body     struct {
		StartTime *int64 `json:"start_time"`
		EndTime   *int64 `json:"end_time"`
	}
}

func newFakeHarper(t *testing.T, rows string) *fakeHarper {
	t.Helper()
	f := &fakeHarper{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests.Add(1)
		if err := json.NewDecoder(r.Body).Decode(&f.body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(rows))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeHarper) datasource() *Datasource {
	return &Datasource{harperClient: harper.NewClientWithHTTPClient(f.srv.Client(), f.srv.URL, "u", "p")}
}

func analyticsQuery(queryAttrs string, timeRange backend.TimeRange) backend.DataQuery {
	return backend.DataQuery{
		RefID:     "A",
		TimeRange: timeRange,
		JSON:      []byte(`{"operation":"get_analytics","queryAttrs":` + queryAttrs + `}`),
	}
}

var tenMinutes = backend.TimeRange{From: time.UnixMilli(1700000000000), To: time.UnixMilli(1700000600000)}

func TestGetAnalyticsTimeBoundsSentToHarper(t *testing.T) {
	cases := []struct {
		name       string
		queryAttrs string
		timeRange  backend.TimeRange
		wantFrom   int64
		wantTo     int64
	}{
		{"explicit numbers win", `{"metric":"m","from":1600000000000,"to":1600000060000}`, tenMinutes, 1600000000000, 1600000060000},
		{"numeric strings", `{"metric":"m","from":"1600000000000","to":" 1600000060000 "}`, tenMinutes, 1600000000000, 1600000060000},
		{"explicit numbers need no time range", `{"metric":"m","from":1600000000000,"to":1600000060000}`, backend.TimeRange{}, 1600000000000, 1600000060000},
		{"equal endpoints", `{"metric":"m","from":1600000000000,"to":1600000000000}`, tenMinutes, 1600000000000, 1600000000000},
		{"uninterpolated macros take the request range", `{"metric":"m","from":"${__from}","to":"${__to}"}`, tenMinutes, 1700000000000, 1700000600000},
		{"macros without braces", `{"metric":"m","from":"$__from","to":"$__to"}`, tenMinutes, 1700000000000, 1700000600000},
		{"a macro means its own end wherever it sits", `{"metric":"m","from":1600000000000,"to":"${__from}"}`, tenMinutes, 1600000000000, 1700000000000},
		{"the to macro in from means the range end", `{"metric":"m","from":"${__to}","to":1800000000000}`, tenMinutes, 1700000600000, 1800000000000},
		{"omitted takes the request range", `{"metric":"m"}`, tenMinutes, 1700000000000, 1700000600000},
		{"only to omitted", `{"metric":"m","from":1600000000000}`, tenMinutes, 1600000000000, 1700000600000},
		{"only from omitted", `{"metric":"m","to":1800000000000}`, tenMinutes, 1700000000000, 1800000000000},
		{"null and empty string are unset", `{"metric":"m","from":null,"to":""}`, tenMinutes, 1700000000000, 1700000600000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := newFakeHarper(t, `[]`)
			if _, err := fake.datasource().query(context.Background(), backend.PluginContext{}, analyticsQuery(c.queryAttrs, c.timeRange)); err != nil {
				t.Fatalf("query: %v", err)
			}
			if fake.body.StartTime == nil || fake.body.EndTime == nil {
				t.Fatalf("harper received start_time=%v end_time=%v, want both present", fake.body.StartTime, fake.body.EndTime)
			}
			if *fake.body.StartTime != c.wantFrom || *fake.body.EndTime != c.wantTo {
				t.Fatalf("harper received start_time=%d end_time=%d, want %d/%d", *fake.body.StartTime, *fake.body.EndTime, c.wantFrom, c.wantTo)
			}
		})
	}
}

func TestGetAnalyticsRejectsUnusableTimeBounds(t *testing.T) {
	cases := []struct {
		name       string
		queryAttrs string
		timeRange  backend.TimeRange
		wantErr    string
	}{
		{"from after to", `{"metric":"m","from":1700000600000,"to":1700000000000}`, tenMinutes, "needs a time range"},
		{"swapped macros", `{"metric":"m","from":"${__to}","to":"${__from}"}`, tenMinutes, "needs a time range"},
		{"omitted bounds and a zero time range", `{"metric":"m"}`, backend.TimeRange{}, "needs a time range"},
		{"explicit zero", `{"metric":"m","from":0,"to":1700000600000}`, tenMinutes, "could not unmarshal get_analytics query JSON"},
		{"negative", `{"metric":"m","from":1700000000000,"to":-1}`, tenMinutes, "could not unmarshal get_analytics query JSON"},
		{"fraction", `{"metric":"m","from":1700000000000.5}`, tenMinutes, "could not unmarshal get_analytics query JSON"},
		{"overflow", `{"metric":"m","to":99999999999999999999}`, tenMinutes, "could not unmarshal get_analytics query JSON"},
		{"boolean", `{"metric":"m","from":true}`, tenMinutes, "could not unmarshal get_analytics query JSON"},
		{"array", `{"metric":"m","from":[1700000000000]}`, tenMinutes, "could not unmarshal get_analytics query JSON"},
		{"unknown macro", `{"metric":"m","from":"${__interval}"}`, tenMinutes, "could not unmarshal get_analytics query JSON"},
		{"non-numeric string", `{"metric":"m","to":"yesterday"}`, tenMinutes, "could not unmarshal get_analytics query JSON"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := newFakeHarper(t, `[]`)
			_, err := fake.datasource().query(context.Background(), backend.PluginContext{}, analyticsQuery(c.queryAttrs, c.timeRange))
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("err = %v, want one containing %q", err, c.wantErr)
			}
			if n := fake.requests.Load(); n != 0 {
				t.Fatalf("harper received %d request(s), want none", n)
			}
		})
	}
}

func TestGetAnalyticsRowsBecomeAWideFrame(t *testing.T) {
	fake := newFakeHarper(t, `[
		{"id":1700000000000,"metric":"connections","node":"a","connections":5},
		{"id":1700000060000,"metric":"connections","node":"a","connections":7}
	]`)
	resp, err := fake.datasource().query(context.Background(), backend.PluginContext{}, analyticsQuery(`{"metric":"connections"}`, tenMinutes))
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(resp.Frames) != 1 {
		t.Fatalf("got %d frames, want 1", len(resp.Frames))
	}
	frame := resp.Frames[0]
	if frame.Rows() != 2 {
		t.Fatalf("got %d rows, want 2", frame.Rows())
	}
	var timeField, valueField *data.Field
	for _, f := range frame.Fields {
		switch f.Type() {
		case data.FieldTypeTime, data.FieldTypeNullableTime:
			timeField = f
		case data.FieldTypeFloat64, data.FieldTypeNullableFloat64:
			valueField = f
		}
	}
	if timeField == nil || valueField == nil {
		t.Fatalf("wide frame lacks a time or value field: %v", frame.Fields)
	}
	if valueField.Labels["node"] != "a" {
		t.Fatalf("value field labels = %v, want node=a", valueField.Labels)
	}
	if got, _ := valueField.FloatAt(1); got != 7 {
		t.Fatalf("second value = %v, want 7", got)
	}
}

func TestQueryDataIsolatesABadQuery(t *testing.T) {
	fake := newFakeHarper(t, `[{"id":1700000000000,"metric":"m","node":"a","v":1}]`)
	good := analyticsQuery(`{"metric":"m"}`, tenMinutes)
	bad := analyticsQuery(`{"metric":"m","from":1700000600000,"to":1700000000000}`, tenMinutes)
	bad.RefID = "B"

	resp, err := fake.datasource().QueryData(context.Background(), &backend.QueryDataRequest{Queries: []backend.DataQuery{good, bad}})
	if err != nil {
		t.Fatalf("QueryData: %v", err)
	}
	if a := resp.Responses["A"]; a.Error != nil || len(a.Frames) != 1 {
		t.Fatalf("A: error=%v frames=%d, want a frame and no error", a.Error, len(a.Frames))
	}
	if b := resp.Responses["B"]; b.Error == nil || !strings.Contains(b.Error.Error(), "needs a time range") {
		t.Fatalf("B: error=%v, want the time range error", b.Error)
	}
	if n := fake.requests.Load(); n != 1 {
		t.Fatalf("harper received %d request(s), want 1 (only the good query)", n)
	}
}
