package common_test

import (
	. "github.com/TheWeatherCompany/softlayer-go/common"

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

	Context("#IsHttpErrorCode", func() {
		It("returns true for codes >= 400", func() {
			errorCodes := []int{400, 401, 403, 500, 501, 509, 600}
			var inError bool = false
			for _, code := range errorCodes {
				inError = IsHttpErrorCode(code)
				Expect(inError).To(BeTrue())
			}
		})

		It("returns false for codes <= 400", func() {
			errorCodes := []int{399, 200, 201, 203}
			var inError bool = false
			for _, code := range errorCodes {
				inError = IsHttpErrorCode(code)
				Expect(inError).To(BeFalse())
			}
		})
	})
})
