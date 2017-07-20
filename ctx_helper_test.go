package gobis_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/orange-cloudfoundry/gobis"
	"net/http"
)

var _ = Describe("CtxHelper", func() {
	Context("InjectContextValue", func() {
		It("should inject simple value when it's found in context", func() {
			req := &http.Request{}
			AddContextValue(req, "key", "value")
			var val string
			err := InjectContextValue(req, "key", &val)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).Should(Equal("value"))
		})
		It("should inject pointer value when it's found in context", func() {
			req := &http.Request{}
			value := "value"
			AddContextValue(req, "key", &value)
			var val *string
			err := InjectContextValue(req, "key", &val)
			Expect(err).NotTo(HaveOccurred())
			Expect(*val).Should(Equal("value"))

			*val = "changed"
			var valChanged *string
			err = InjectContextValue(req, "key", &valChanged)
			Expect(err).NotTo(HaveOccurred())
			Expect(*valChanged).Should(Equal("changed"))
		})
		It("should inject complex value when it's found in context", func() {
			req := &http.Request{}
			sValues := []string{"val1", "val2", "val3"}
			AddContextValue(req, "key", sValues)
			var val []string
			err := InjectContextValue(req, "key", &val)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).Should(HaveLen(len(sValues)))
			for i := 0; i < len(sValues); i++ {
				Expect(val[i]).Should(Equal(sValues[i]))
			}
		})
		It("should inject nothing if key not found and not give error", func() {
			req := &http.Request{}
			var val string
			err := InjectContextValue(req, "key", &val)
			Expect(err).NotTo(HaveOccurred())
			Expect(val).Should(Equal(""))
		})
		It("should return error if a pointer is not given", func() {
			req := &http.Request{}
			AddContextValue(req, "key", "value")
			var val string
			err := InjectContextValue(req, "key", val)
			Expect(err).Should(HaveOccurred())
		})
		It("should return error if value found is not type of the request interface", func() {
			req := &http.Request{}
			AddContextValue(req, "key", "value")
			var val int
			err := InjectContextValue(req, "key", val)
			Expect(err).Should(HaveOccurred())
		})
	})
})
