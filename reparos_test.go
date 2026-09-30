package reparos

import (
	"testing"
)

func TestCorrectZeroConfig(t *testing.T) {
	// Test zero-config top-level function
	corrected, err := Correct("d pasteur q3")
	if err != nil {
		t.Fatalf("Correct error: %v", err)
	}

	expected := "đường pasteur quận 3"
	if corrected != expected {
		t.Errorf("Expected '%s', got '%s'", expected, corrected)
	}
}

func TestPredictorExplicit(t *testing.T) {
	predictor, err := New(WithBeamSize(5))
	if err != nil {
		t.Fatalf("Failed to initialize predictor: %v", err)
	}
	defer predictor.Close()

	tests := []struct {
		input       string
		expectedTop string
	}{
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
		})
	}
}
