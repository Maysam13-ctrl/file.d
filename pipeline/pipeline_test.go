package pipeline_test

import (
	"testing"

	"github.com/ozontech/file.d/pipeline"
	"github.com/ozontech/file.d/plugin/input/fake"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func getFakeInputInfo() *pipeline.InputPluginInfo {
	input, _ := fake.Factory()
	return &pipeline.InputPluginInfo{
		PluginStaticInfo: &pipeline.PluginStaticInfo{
			Type:    "",
			Factory: nil,
			Config:  nil,
		},
		PluginRuntimeInfo: &pipeline.PluginRuntimeInfo{
			Plugin: input,
			ID:     "",
		},
	}
}

func TestInInvalidMessages(t *testing.T) {
	cases := []struct {
		name             string
		message          []byte
		pipelineSettings *pipeline.Settings
		offset           int64
		sourceID         pipeline.SourceID
	}{
		{
			name:    "empty_message",
			message: []byte(""),
			pipelineSettings: &pipeline.Settings{
				Capacity:           5,
				Decoder:            "json",
				MetricHoldDuration: pipeline.DefaultMetricHoldDuration,
			},
			offset:   int64(666),
			sourceID: pipeline.SourceID(1<<16 + int(1)),
		},
		{
			name:    "too_long_message",
			message: []byte("{\"value\":\"i'm longer than 1 byte\""),
			pipelineSettings: &pipeline.Settings{
				Capacity:           5,
				Decoder:            "json",
				MaxEventSize:       1,
				MetricHoldDuration: pipeline.DefaultMetricHoldDuration,
			},
			offset:   int64(666),
			sourceID: pipeline.SourceID(2<<16 + int(3)),
		},
	}

	for _, tCase := range cases {
		t.Run(tCase.name, func(t *testing.T) {
			pipe := pipeline.New("test_pipeline", tCase.pipelineSettings, prometheus.NewRegistry())

			pipe.SetInput(getFakeInputInfo())

			seqID := pipe.In(tCase.sourceID, "kafka", tCase.offset, tCase.message, false, nil)
			require.Equal(t, pipeline.EventSeqIDError, seqID)
		})
	}
}
