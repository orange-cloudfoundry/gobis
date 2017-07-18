package ctx_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCtx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ctx Suite")
}
