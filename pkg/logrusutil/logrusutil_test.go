package logrusutil

import (
	"os"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/logrusutil/stackdriver"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestCreateDefaultFormatter(t *testing.T) {
	envKey := "LOGRUS_FORMAT"
	origEnvValue, exist := os.LookupEnv(envKey)
	defer resetEnvValue(t, envKey, origEnvValue, exist)
	for _, tt := range createFormatterTests {
		err := os.Setenv("LOGRUS_FORMAT", tt.format)
		require.NoError(t, err)

		formatter := CreateDefaultFormatter()
		require.IsType(t, tt.expectedFormatter, formatter)
	}
}

func resetEnvValue(t *testing.T, envKey string, origEnvValue string, wasSet bool) {
	if wasSet {
		err := os.Setenv(envKey, origEnvValue)
		require.NoError(t, err)
	}
}

var createFormatterTests = []struct {
	format            string
	expectedFormatter interface{}
}{
	{
		format:            "text",
		expectedFormatter: &logrus.TextFormatter{},
	},
	{
		format:            "stackdriver",
		expectedFormatter: &stackdriver.Formatter{},
	},
}
