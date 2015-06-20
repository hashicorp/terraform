package common_test

import (
	. "github.com/maximilien/softlayer-go/common"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utility", func() {
	var (
		result bool
		err    error
	)

	Context("#ValidateJson", func() {
		It("returns true if the input string is valid Json format", func() {
			result, err = ValidateJson(`{"correct_json":"whatever"}`)
			Expect(result).To(Equal(true))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns false if the input string is invalid Json format", func() {
			result, err = ValidateJson(`{{"wrong_json":"whatever"}`)
			Expect(result).To(Equal(false))
			Expect(err).To(HaveOccurred())
		})
	})
})
