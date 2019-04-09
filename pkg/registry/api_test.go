package registry_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/pivotal/monitoring-indicator-protocol/pkg/registry/status_store"

	. "github.com/onsi/gomega"

	"github.com/pivotal/monitoring-indicator-protocol/pkg/indicator"
	"github.com/pivotal/monitoring-indicator-protocol/pkg/registry"
)

func TestRegisterHandler(t *testing.T) {
	t.Run("it returns 200 if the request is valid", func(t *testing.T) {
		g := NewGomegaWithT(t)

		body := bytes.NewBuffer([]byte(`---
apiVersion: v0

product: 
  name: redis-tile
  version: 0.11

metadata:
  deployment: redis-abc-123

indicators:
- name: test_performance_indicator
  promql: prom
  thresholds:
  - level: warning
    gte: 50`))

		req := httptest.NewRequest("POST", "/register", body)
		resp := httptest.NewRecorder()

		docStore := registry.NewDocumentStore(1 * time.Minute, time.Now)
		handle := registry.NewRegisterHandler(docStore)
		handle(resp, req)

		g.Expect(resp.Header().Get("Content-Type")).To(Equal("application/json"))
		g.Expect(resp.Code).To(Equal(http.StatusOK))
		g.Expect(docStore.AllDocuments()).To(ConsistOf(indicator.Document{
			APIVersion: "v0",
			Product:    indicator.Product{Name: "redis-tile", Version: "0.11"},
			Metadata: map[string]string{
				"deployment": "redis-abc-123",
			},
			Layout: indicator.Layout{
				Sections: []indicator.Section{{
					Title: "Metrics",
					Indicators: []indicator.Indicator{{
						Name:   "test_performance_indicator",
						PromQL: "prom",
						Alert: indicator.Alert{
							For:  "1m",
							Step: "1m",
						},
						Presentation: &indicator.Presentation{
							CurrentValue: false,
							ChartType:    "step",
							Frequency:    0,
							Labels:       []string{},
						},
						Thresholds: []indicator.Threshold{
							{
								Level:    "warning",
								Operator: indicator.GreaterThanOrEqualTo,
								Value:    50,
							},
						},
					}},
				}},
			},
			Indicators: []indicator.Indicator{{
				Name:   "test_performance_indicator",
				PromQL: "prom",
				Alert: indicator.Alert{
					For:  "1m",
					Step: "1m",
				},
				Presentation: &indicator.Presentation{
					CurrentValue: false,
					ChartType:    "step",
					Frequency:    0,
					Labels:       []string{},
				},
				Thresholds: []indicator.Threshold{
					{
						Level:    "warning",
						Operator: indicator.GreaterThanOrEqualTo,
						Value:    50,
					},
				},
			}},
		}))
	})

	t.Run("it returns 400 if there are validation errors", func(t *testing.T) {
		g := NewGomegaWithT(t)

		body := bytes.NewBuffer([]byte(`---
apiVersion: v0
indicators:
- promql: " "
  name: none
`))

		req := httptest.NewRequest("POST", "/register?deployment=redis-abc", body)
		resp := httptest.NewRecorder()

		docStore := registry.NewDocumentStore(1 * time.Minute, time.Now)
		handle := registry.NewRegisterHandler(docStore)
		handle(resp, req)

		g.Expect(docStore.AllDocuments()).To(HaveLen(0))

		g.Expect(resp.Code).To(Equal(http.StatusBadRequest))

		responseBody, err := ioutil.ReadAll(resp.Body)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(responseBody).To(MatchJSON(`{ "errors": ["product name is required", "product version is required", "indicators[0] promql is required"]}`))
	})

	t.Run("it returns 400 if the yml is invalid", func(t *testing.T) {
		g := NewGomegaWithT(t)

		body := bytes.NewBuffer([]byte(`---
indicators: aasdfasdf`))

		req := httptest.NewRequest("POST", "/register?deployment=redis-abc&product=redis-tile", body)
		resp := httptest.NewRecorder()

		docStore := registry.NewDocumentStore(1 * time.Minute, time.Now)
		handle := registry.NewRegisterHandler(docStore)
		handle(resp, req)

		g.Expect(docStore.AllDocuments()).To(HaveLen(0))

		g.Expect(resp.Code).To(Equal(http.StatusBadRequest))

		responseBody, err := ioutil.ReadAll(resp.Body)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(responseBody).To(MatchJSON(`{ "errors": ["could not unmarshal indicators: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str ` + "`aasdfasdf`" + ` into []indicator.yamlIndicator"] }`))
	})
}

func TestBulkStatusUpdateHandler(t *testing.T) {
	t.Run("it returns 200 & updates the store status", func(t *testing.T) {
		g := NewGomegaWithT(t)

		body := bytes.NewBuffer([]byte(`[{"name": "latency", "status": "critical"},{"name": "error_rate", "status": "warning"}]`))
		req := httptest.NewRequest("POST", "/", body)
		resp := httptest.NewRecorder()

		now := time.Now()
		store := status_store.New(func() time.Time { return now })

		req = mux.SetURLVars(req, map[string]string{
			"documentID": "my-component-1234234234",
		})
		registry.NewIndicatorStatusBulkUpdateHandler(store)(resp, req)

		g.Expect(store.StatusFor("my-component-1234234234", "latency")).To(Equal(status_store.IndicatorStatus{
			DocumentUID:   "my-component-1234234234",
			IndicatorName: "latency",
			Status:        "critical",
			UpdatedAt:     now,
		}))

		g.Expect(store.StatusFor("my-component-1234234234", "error_rate")).To(Equal(status_store.IndicatorStatus{
			DocumentUID:   "my-component-1234234234",
			IndicatorName: "error_rate",
			Status:        "warning",
			UpdatedAt:     now,
		}))
	})

	t.Run("it returns a 400 if indicator status are not passed into the body", func(t *testing.T) {
		g := NewGomegaWithT(t)

		body:= bytes.NewBuffer([]byte(`[""]`))
		req := httptest.NewRequest("POST", "/", body)
		resp := httptest.NewRecorder()

		now := time.Now()
		store := status_store.New(func() time.Time { return now })

		req = mux.SetURLVars(req, map[string]string{
			"documentID": "my-component-1234234234",
		})
		registry.NewIndicatorStatusBulkUpdateHandler(store)(resp, req)

		g.Expect(resp.Result().StatusCode).To(Equal(http.StatusBadRequest))
	})
}

func TestIndicatorDocumentsHandler(t *testing.T) {
	t.Run("it returns 200", func(t *testing.T) {
		g := NewGomegaWithT(t)

		req := httptest.NewRequest("POST", "/indicator-documents", nil)
		resp := httptest.NewRecorder()

		docStore := registry.NewDocumentStore(1 * time.Minute, time.Now)
		docStore.UpsertDocument(indicator.Document{
			Product: indicator.Product{Name: "my-product-a", Version: "1"},
			Metadata: map[string]string{
				"deployment": "abc-123",
			},
			Indicators: []indicator.Indicator{{
				Name: "test_errors1",
				Alert: indicator.Alert{
					For:  "5m",
					Step: "10s",
				},
			}, {
				Name: "test_errors2",
				Alert: indicator.Alert{
					For:  "5m",
					Step: "10s",
				},
				Presentation: &indicator.Presentation{
					Units: "nanoseconds",
				},
			}},
		})

		statusStore := status_store.New(func() time.Time { return time.Date(2012, 12, 1, 16, 45, 19, 0, time.UTC) })
		statusStore.UpdateStatus(status_store.UpdateRequest{
			Status:        "critical",
			IndicatorName: "test_errors2",
			DocumentUID:   "my-product-a-a902332065d69c1787f419e235a1f1843d98c884",
		})

		handle := registry.NewIndicatorDocumentsHandler(docStore, statusStore)
		handle(resp, req)

		g.Expect(resp.Header().Get("Content-Type")).To(Equal("application/json"))
		g.Expect(resp.Code).To(Equal(http.StatusOK))

		responseBody, err := ioutil.ReadAll(resp.Body)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(responseBody).To(MatchJSON(`
			[
 				{
                    "apiVersion": "",
					"uid": "my-product-a-a902332065d69c1787f419e235a1f1843d98c884",
                    "product": {
						"name": "my-product-a",
                    	"version": "1"
					},
                    "metadata": {
                      "deployment": "abc-123"
                    },
                    "indicators": [
                      {
                        "name": "test_errors1",
                        "promql": "",
                        "thresholds": [],
						"alert": {
							"for": "5m",
							"step": "10s"
						},
                        "presentation": null,
                        "status": null
                      },
                      {
                        "name": "test_errors2",
                        "promql": "",
                        "thresholds": [],
						"alert": {
							"for": "5m",
							"step": "10s"
						},
                        "presentation": {
                          "chartType": "",
                          "currentValue": false,
                          "frequency": 0,
                          "labels": [],
                          "units": "nanoseconds"
                        },
						"status": {
						  "value": "critical",
						  "updatedAt": "2012-12-01T16:45:19Z"
						}
                      }
                    ],
                    "layout": {
                      "title": "",
                      "description": "",
					  "sections": [],
                      "owner": ""
                    }
                  }
			]`))
	})
}
