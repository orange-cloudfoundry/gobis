package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGobis(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gobis Suite")
}
