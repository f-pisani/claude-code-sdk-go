package claudecode

import (
	"testing"
)

func TestOptionsBufferSizes(t *testing.T) {
	tests := []struct {
		name              string
		options           *Options
		expectedMsgBuffer int
		expectedErrBuffer int
	}{
		{
			name:              "nil options returns defaults",
			options:           nil,
			expectedMsgBuffer: 10,
			expectedErrBuffer: 1,
		},
		{
			name:              "new options has defaults",
			options:           NewOptions(),
			expectedMsgBuffer: 10,
			expectedErrBuffer: 1,
		},
		{
			name: "custom buffer sizes",
			options: &Options{
				MessageBufferSize: 20,
				ErrorBufferSize:   5,
			},
			expectedMsgBuffer: 20,
			expectedErrBuffer: 5,
		},
		{
			name: "zero buffer sizes return defaults",
			options: &Options{
				MessageBufferSize: 0,
				ErrorBufferSize:   0,
			},
			expectedMsgBuffer: 10,
			expectedErrBuffer: 1,
		},
		{
			name: "negative buffer sizes return defaults",
			options: &Options{
				MessageBufferSize: -5,
				ErrorBufferSize:   -1,
			},
			expectedMsgBuffer: 10,
			expectedErrBuffer: 1,
		},
		{
			name: "only message buffer custom",
			options: &Options{
				MessageBufferSize: 100,
				ErrorBufferSize:   0,
			},
			expectedMsgBuffer: 100,
			expectedErrBuffer: 1,
		},
		{
			name: "only error buffer custom",
			options: &Options{
				MessageBufferSize: 0,
				ErrorBufferSize:   10,
			},
			expectedMsgBuffer: 10,
			expectedErrBuffer: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgBufSize := tt.options.GetMessageBufferSize()
			if msgBufSize != tt.expectedMsgBuffer {
				t.Errorf("GetMessageBufferSize() = %d, want %d", msgBufSize, tt.expectedMsgBuffer)
			}

			errBufSize := tt.options.GetErrorBufferSize()
			if errBufSize != tt.expectedErrBuffer {
				t.Errorf("GetErrorBufferSize() = %d, want %d", errBufSize, tt.expectedErrBuffer)
			}
		})
	}
}

func TestOptionsWithBufferSizesInQuery(t *testing.T) {
	// Test that buffer sizes are used when creating channels
	// This is more of a usage example since we can't directly test channel buffer sizes

	options := &Options{
		MessageBufferSize: 50,
		ErrorBufferSize:   2,
		MaxThinkingTokens: 8000,
	}

	// Verify the getter methods work correctly
	if options.GetMessageBufferSize() != 50 {
		t.Errorf("GetMessageBufferSize() = %d, want 50", options.GetMessageBufferSize())
	}
	if options.GetErrorBufferSize() != 2 {
		t.Errorf("GetErrorBufferSize() = %d, want 2", options.GetErrorBufferSize())
	}
}
