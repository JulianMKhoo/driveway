package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

func TestTerraformNginxService(t *testing.T) {
	t.Parallel()
	serviceName := "nginx-highway-service"
	output := "<h1>Welcome to nginx!</h1>"
	kubeOptions := k8s.NewKubectlOptions("", "", "highway")
	_, getKubeErr := k8s.GetServiceE(t, kubeOptions, serviceName)
	if getKubeErr != nil {
		defer k8s.DeleteNamespace(t, kubeOptions, "argocd")
		fmt.Println("Infrastructure not found. Running Terraform Apply...")
		terraformOptions := &terraform.Options{TerraformDir: "../../terraform"}
		defer terraform.Destroy(t, terraformOptions)
		terraform.InitAndApply(t, terraformOptions)
		k8s.WaitUntilServiceAvailable(t, kubeOptions, serviceName, 12, 5*time.Second)
	} else {
		fmt.Println("Infrastructure already exists. Skipping Apply and running health checks...")
	}
	tunnel := k8s.NewTunnel(kubeOptions, k8s.ResourceTypeService, serviceName, 0, 80)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	url := fmt.Sprintf("http://%s", tunnel.Endpoint())
	http_helper.HttpGetWithCustomValidation(t, url, nil, func(statusCode int, body string) bool {
		return statusCode == 200 && strings.Contains(body, output)
	})
}
