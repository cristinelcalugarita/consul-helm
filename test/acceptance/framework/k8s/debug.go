package k8s

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	terratestk8s "github.com/gruntwork-io/terratest/modules/k8s"
	terratestLogger "github.com/gruntwork-io/terratest/modules/logger"
	"github.com/hashicorp/consul-helm/test/acceptance/framework/helpers"
	"github.com/hashicorp/consul-helm/test/acceptance/framework/logger"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WritePodsDebugInfoIfFailed calls kubectl describe and kubectl logs --all-containers
// on pods filtered by the labelSelector and writes it to the debugDirectory.
func WritePodsDebugInfoIfFailed(t *testing.T, kubectlOptions *terratestk8s.KubectlOptions, debugDirectory, labelSelector string) {
	t.Helper()

	if t.Failed() {
		// Create k8s client from kubectl options
		client := helpers.KubernetesClientFromOptions(t, kubectlOptions)

		contextName := helpers.KubernetesContextFromOptions(t, kubectlOptions)

		// Create a directory for the test
		testDebugDirectory := filepath.Join(debugDirectory, t.Name(), contextName)
		require.NoError(t, os.MkdirAll(testDebugDirectory, 0755))

		logger.Logf(t, "dumping logs and pod info for %s to %s", labelSelector, testDebugDirectory)
		pods, err := client.CoreV1().Pods(kubectlOptions.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
		require.NoError(t, err)

		for _, pod := range pods.Items {
			// Get logs for each pod, passing the discard terratestLogger to make sure secrets aren't printed to test logs.
			logs, err := RunKubectlAndGetOutputWithLoggerE(t, kubectlOptions, terratestLogger.Discard, "logs", "--all-containers=true", pod.Name)

			// Write logs or err to file name <pod.Name>.log
			logFilename := filepath.Join(testDebugDirectory, fmt.Sprintf("%s.log", pod.Name))
			if err != nil {
				logs = fmt.Sprintf("Error getting logs: %s: %s", err, logs)
			}
			require.NoError(t, ioutil.WriteFile(logFilename, []byte(logs), 0600))

			// Describe pod
			desc, err := RunKubectlAndGetOutputWithLoggerE(t, kubectlOptions, terratestLogger.Discard, "describe", "pod", pod.Name)

			// Write pod info or error to file name <pod.Name>.txt
			if err != nil {
				desc = fmt.Sprintf("Error describing pod: %s: %s", err, desc)
			}
			descFilename := filepath.Join(testDebugDirectory, fmt.Sprintf("%s.txt", pod.Name))
			require.NoError(t, ioutil.WriteFile(descFilename, []byte(desc), 0600))
		}
	}
}
