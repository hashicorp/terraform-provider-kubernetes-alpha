package provider

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	tftest "github.com/hashicorp/terraform-plugin-test"

	k8shelper "github.com/hashicorp/terraform-provider-kubernetes-alpha/acctest/helper/kubernetes"
	provider "github.com/hashicorp/terraform-provider-kubernetes-alpha/provider"
)

var kubernetesHelper *k8shelper.Helper

var providerName = "kubernetes-alpha"
var binhelper *tftest.Helper

func TestMain(m *testing.M) {
	if tftest.RunningAsPlugin() {
		provider.Serve()
		os.Exit(0)
		return
	}

	sourceDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	binhelper = tftest.AutoInitProviderHelper(providerName, sourceDir)
	defer binhelper.Close()

	kubernetesHelper = k8shelper.InitHelper()

	rand.Seed(time.Now().UTC().UnixNano())

	exitcode := m.Run()
	os.Exit(exitcode)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz")

func randName() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return fmt.Sprintf("tf-acc-test-%s", string(b))
}
