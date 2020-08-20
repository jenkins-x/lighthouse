package engines

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type responseVendor struct {
	codes []int
	data  []string

	position int
}

func (r *responseVendor) next() (int, string) {
	code := r.codes[r.position]
	datum := r.data[r.position]

	r.position = r.position + 1
	if r.position == len(r.codes) {
		r.position = 0
	}

	return code, datum
}

func parrotServer(codes []int, data []string) *httptest.Server {
	vendor := responseVendor{
		codes: codes,
		data:  data,
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code, datum := vendor.next()
		w.WriteHeader(code)
		fmt.Fprint(w, datum)
	}))
}

func TestGetBuildID(t *testing.T) {
	oldSleep := sleep
	sleep = func(time.Duration) { return }
	defer func() { sleep = oldSleep }()

	var testCases = []struct {
		name        string
		codes       []int
		data        []string
		expected    string
		expectedErr bool
	}{
		{
			name:        "all good",
			codes:       []int{200},
			data:        []string{"yay"},
			expected:    "yay",
			expectedErr: false,
		},
		{
			name:        "fail then success",
			codes:       []int{500, 200},
			data:        []string{"boo", "yay"},
			expected:    "yay",
			expectedErr: false,
		},
		{
			name:        "fail",
			codes:       []int{500},
			data:        []string{"boo"},
			expected:    "boo",
			expectedErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			buildNumSrv := parrotServer(testCase.codes, testCase.data)

			actual, actualErr := GetBuildID("dummy", buildNumSrv.URL)
			if testCase.expectedErr && actualErr == nil {
				t.Errorf("%s: expected an error but got none", testCase.name)
			} else if !testCase.expectedErr && actualErr != nil {
				t.Errorf("%s: expected no error but got one: %v", testCase.name, actualErr)
			} else if !testCase.expectedErr && actual != testCase.expected {
				t.Errorf("%s: expected response %v but got: %v", testCase.name, testCase.expected, actual)
			}

			buildNumSrv.Close()
		})
	}
}
