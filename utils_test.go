package gobis_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/orange-cloudfoundry/gobis"
	"github.com/mitchellh/mapstructure"
)

type TestStruct struct {
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
}
type SecondTestStruct struct {
	Zorro string `json:"zorro"`
}

var _ = Describe("Utils", func() {
	Context("When passing one interface", func() {
		It("should return the corresponding map", func() {
			m := InterfaceToMap(TestStruct{
				Foo: "bar",
				Bar: 1,
			})
			var t TestStruct
			err := mapstructure.Decode(m, &t)
			Expect(err).ToNot(HaveOccurred())

			Expect(t.Foo).Should(Equal("bar"))
			Expect(t.Bar).Should(Equal(1))
		})
	})
	Context("When passing multiple interface", func() {
		It("should return the corresponding map by merging multiple interface", func() {
			m := InterfaceToMap(
				TestStruct{
					Foo: "bar",
					Bar: 1,
				},
				SecondTestStruct{
					Zorro: "garcia",
				},
			)
			var t TestStruct
			err := mapstructure.Decode(m, &t)
			Expect(err).ToNot(HaveOccurred())
			var z SecondTestStruct
			err = mapstructure.Decode(m, &z)
			Expect(err).ToNot(HaveOccurred())

			Expect(z.Zorro).Should(Equal("garcia"))

		})
	})
})
