package gobis_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/orange-cloudfoundry/gobis"
)

type TestStruct struct {
	Foo string
	Bar int
}
type SecondTestStruct struct {
	Zorro string
}

var _ = Describe("Utils", func() {
	Context("When passing one interface", func() {
		It("should return the corresponding map", func() {
			m := InterfaceToMap(TestStruct{
				Foo: "bar",
				Bar: 1,
			})
			Expect(m["foo"]).Should(Equal("bar"))
			Expect(m["bar"]).Should(Equal(1))
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
			Expect(m["foo"]).Should(Equal("bar"))
			Expect(m["zorro"]).Should(Equal("garcia"))
			Expect(m["bar"]).Should(Equal(1))
		})
	})
})
