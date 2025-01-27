package pipeline

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

// Can't use fake plugin here dye cycle import
type TestInputPlugin struct{}

func (p *TestInputPlugin) Start(_ AnyConfig, _ *InputPluginParams) {}
func (p *TestInputPlugin) Stop()                                   {}
func (p *TestInputPlugin) Commit(_ *Event)                         {}
func (p *TestInputPlugin) PassEvent(_ *Event) bool {
	return true
}

func TestPipelineStreamEvent(t *testing.T) {
	settings := &Settings{
		Capacity:           5,
		Decoder:            "json",
		MetricHoldDuration: DefaultMetricHoldDuration,
	}
	p := New("test", settings, prometheus.NewRegistry())

	streamID := StreamID(123123)
	procs := int32(7)
	p.procCount = atomic.NewInt32(procs)
	p.input = &TestInputPlugin{}
	event := newEvent()
	event.SourceID = SourceID(streamID)
	event.streamName = DefaultStreamName
	event.SeqID = 123456789

	p.streamEvent(event)

	assert.Equal(t, event, p.streamer.getStream(streamID, DefaultStreamName).first)

	p.UseSpread()
	p.streamEvent(event)

	expectedStreamID := StreamID(event.SeqID % uint64(procs))

	assert.Equal(t, event, p.streamer.getStream(expectedStreamID, DefaultStreamName).first)
}

func TestCheckInputBytes(t *testing.T) {
	cases := []struct {
		name             string
		pipelineSettings *Settings
		input            []byte
		want             []byte
		wantOk           bool
	}{
		{
			name: "empty_input",
			pipelineSettings: &Settings{
				Capacity:           5,
				Decoder:            "raw",
				MetricHoldDuration: DefaultMetricHoldDuration,
			},
			input:  []byte(""),
			wantOk: false,
		},
		{
			name: "only_newline",
			pipelineSettings: &Settings{
				Capacity:           5,
				Decoder:            "raw",
				MetricHoldDuration: DefaultMetricHoldDuration,
			},
			input:  []byte("\n"),
			wantOk: false,
		},
		{
			name: "too_long_input",
			pipelineSettings: &Settings{
				Capacity:           5,
				Decoder:            "raw",
				MaxEventSize:       1,
				MetricHoldDuration: DefaultMetricHoldDuration,
			},
			input:  []byte("i'm longer than 1 byte"),
			wantOk: false,
		},
		{
			name: "no_cutoff",
			pipelineSettings: &Settings{
				Capacity:           5,
				Decoder:            "raw",
				MetricHoldDuration: DefaultMetricHoldDuration,
				MaxEventSize:       20,
				CutOffEventByLimit: true,
			},
			input:  []byte("some loooooooog"),
			want:   []byte("some loooooooog"),
			wantOk: true,
		},
		{
			name: "cutoff_no_newline",
			pipelineSettings: &Settings{
				Capacity:           5,
				Decoder:            "raw",
				MetricHoldDuration: DefaultMetricHoldDuration,
				MaxEventSize:       10,
				CutOffEventByLimit: true,
			},
			input:  []byte("some loooooooog"),
			want:   []byte("some loooo"),
			wantOk: true,
		},
		{
			name: "cutoff_newline",
			pipelineSettings: &Settings{
				Capacity:           5,
				Decoder:            "raw",
				MetricHoldDuration: DefaultMetricHoldDuration,
				MaxEventSize:       10,
				CutOffEventByLimit: true,
			},
			input:  []byte("some loooooooog\n"),
			want:   []byte("some loooo\n"),
			wantOk: true,
		},
		{
			name: "cutoff_with_msg",
			pipelineSettings: &Settings{
				Capacity:              5,
				Decoder:               "raw",
				MetricHoldDuration:    DefaultMetricHoldDuration,
				MaxEventSize:          10,
				CutOffEventByLimit:    true,
				CutOffEventByLimitMsg: "<cutoff>",
			},
			input:  []byte("some loooooooog\n"),
			want:   []byte("some loooo<cutoff>\n"),
			wantOk: true,
		},
	}

	for _, tCase := range cases {
		t.Run(tCase.name, func(t *testing.T) {
			pipe := New("test_pipeline", tCase.pipelineSettings, prometheus.NewRegistry())

			data, ok := pipe.checkInputBytes(tCase.input)

			assert.Equal(t, tCase.wantOk, ok)
			if !tCase.wantOk {
				return
			}
			assert.Equal(t, tCase.want, data)
		})
	}
}
