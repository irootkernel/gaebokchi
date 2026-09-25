package process

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"os"
	"testing"
)

func TestProcessSuite(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Process Suite")
}

type testT struct{ ginkgo.GinkgoTInterface }

func TestMain(m *testing.M) {
	if os.Getenv("GO_GAEBOKCHI_HELPER") == "1" {
		runHelperProcess()
	}
	os.Exit(m.Run())
}
