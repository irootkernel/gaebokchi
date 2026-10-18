package zcode

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"testing"
)

func TestZCodeSuite(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "ZCode Suite")
}

type testT struct{ ginkgo.GinkgoTInterface }
