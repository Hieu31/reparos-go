package reparos

import (
	"testing"
)

func TestPredictor(t *testing.T) {
	predictor, err := New("models/v4_int8", WithBeamSize(5))
	if err != nil {
		t.Fatalf("Failed to initialize predictor: %v", err)
	}
	defer predictor.Close()

	tests := []struct {
		input       string
		expectedTop string
	}{
		{
			input:       "d pasteur q3",
			expectedTop: "đường pasteur quận 3",
		},
		{
			input:       "bv cho ray",
			expectedTop: "bệnh viện chợ rẫy",
		},
		{
			input:       "nga tu hang xanh",
			expectedTop: "ngã tư hàng xanh",
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			res, err := predictor.Predict(tc.input)
			if err != nil {
				t.Fatalf("Predict error: %v", err)
			}

			if res.Top1Query != tc.expectedTop {
				t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expectedTop, res.Top1Query)
			}

			if res.LatencyMs <= 0 {
				t.Errorf("Expected positive latency, got %f", res.LatencyMs)
			}
		})
	}
}
