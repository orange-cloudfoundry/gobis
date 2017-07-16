package test_integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TestIntegration Suite")
}
