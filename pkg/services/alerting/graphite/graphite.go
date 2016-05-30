package graphite

import (
	"fmt"
	"github.com/franela/goreq"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/components/simplejson"
	m "github.com/grafana/grafana/pkg/models"
	"net/http"
	"net/url"
	"time"
)

type GraphiteClient struct{}

type GraphiteSerie struct {
	Datapoints [][2]float64
	Target     string
}

type GraphiteResponse []GraphiteSerie

func (this GraphiteClient) GetSeries(rule m.AlertRule) (m.TimeSeriesSlice, error) {
	query := &m.GetDataSourceByIdQuery{Id: rule.DatasourceId, OrgId: rule.OrgId}
	if err := bus.Dispatch(query); err != nil {
		return nil, err
	}

	v := url.Values{
		"format": []string{"json"},
		"target": []string{getTargetFromRule(rule)},
		"until":  []string{"now"},
		"from":   []string{"-" + rule.QueryRange},
	}

	res, err := goreq.Request{
		Method:  "POST",
		Uri:     query.Result.Url + "/render",
		Body:    v.Encode(),
		Timeout: 500 * time.Millisecond,
	}.Do()

	response := GraphiteResponse{}
	res.Body.FromJsonTo(&response)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected httpstatus 200, found %d", res.StatusCode)
	}

	timeSeries := make([]*m.TimeSeries, 0)

	for _, v := range response {
		timeSeries = append(timeSeries, m.NewTimeSeries(v.Target, v.Datapoints))
	}

	return timeSeries, nil
}

func getTargetFromRule(rule m.AlertRule) string {
	json, _ := simplejson.NewJson([]byte(rule.Query))

	return json.Get("target").MustString()
}