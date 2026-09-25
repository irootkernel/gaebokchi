package rpc

import (
	"gaebokchi/internal/testutil"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"os"
	"testing"
)

func TestRpcSuite(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Rpc Suite")
}

type testT struct{ ginkgo.GinkgoTInterface }

func TestMain(m *testing.M) {
	if os.Getenv("GAEBOKCHI_FAKE") == "1" {
		if err := testutil.RunFakeScript(os.Stdin, os.Stdout, os.Stderr); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}
