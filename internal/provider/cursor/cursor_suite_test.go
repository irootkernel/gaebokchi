package cursor

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"testing"
)

func TestCursorSuite(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Cursor Suite")
}

type testT struct{ ginkgo.GinkgoTInterface }
