package start

import (
	"encoding/json"
	"os"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	lighthouseclient "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// GetCommand creates the cobra command for starting jobs
func GetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "start",
		Run: func(cmd *cobra.Command, args []string) {
			execute()
		},
	}
	return cmd
}

func execute() {
	logrus.Info("Creating LighthouseJob...")
	var lhjob lighthousev1alpha1.LighthouseJob
	if err := json.Unmarshal([]byte(os.Getenv("LHJOB")), &lhjob); err != nil {
		logrus.WithError(err).Fatal("Could not unmarshal job")
	}
	if err := createLhJob(lhjob); err != nil {
		logrus.WithError(err).Fatal("Failed to create job")
	}
}

func createLhJob(lhjob lighthousev1alpha1.LighthouseJob) error {
	cfg, err := clients.GetConfig("", "")
	if err != nil {
		return err
	}
	lhclient, err := lighthouseclient.NewForConfig(cfg)
	if err != nil {
		return err
	}
	if _, err := lhclient.LighthouseV1alpha1().LighthouseJobs(lhjob.Namespace).Create(&lhjob); err != nil {
		return err
	}
	return nil
}
