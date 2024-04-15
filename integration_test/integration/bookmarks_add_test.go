package integration

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pranav-patil/go-serverless-api/pkg/util"
)

const baseURI = "http://localhost:8080/emprovise/api"

var _ = BeforeSuite(func() {
	fmt.Println("BeforeSuite")
})

var _ = Describe("Go Serverless API Methods", Label("library"), func() {
	var restyClient util.RestClient

	BeforeEach(func() {
		var err error
		restyClient, err = util.NewRestyClient(baseURI, 10)
		if err != nil {
			return
		}
	})

	Context("bookmarks are added and fetched", func() {
		It("should Post JSON bookmarks for user", func() {
			fmt.Println("=========] Post JSON bookmarks for user")

			request := `{ "bookmarks": [
					{ "url": "https://jalammar.github.io/illustrated-transformer/" },
					{ "url": "https://zapier.com/blog/claude-ai/" },
					{ "url": "https://github.com/openxla/xla" }
				] }`

			code, response, err := restyClient.Post("/bookmarks", "BBB", "application/json", request)
			fmt.Println("=========] should return nil error")
			Expect(err).Should(BeNil())
			fmt.Println("=========] should return 201 status code")
			Expect(code).To(Equal(201))

			Expect(response["totalCount"]).To(Equal(float64(3)))
			bookmarks := response["bookmarks"].([]interface{})
			Expect(len(bookmarks)).To(BeNumerically("==", 3))
		})

		It("should Get JSON bookmarks for user", func() {
			fmt.Println("=========] Get JSON bookmarks for user")

			code, response, err := restyClient.Get("/bookmarks", "BBB", "application/json")
			fmt.Println("=========] should return nil error")
			Expect(err).Should(BeNil())
			fmt.Println("=========] should return 200 status code")
			Expect(code).To(Equal(200))

			Expect(response["totalCount"]).To(Equal(float64(3)))
			bookmarks := response["bookmarks"].([]interface{})
			Expect(len(bookmarks)).To(BeNumerically("==", 3))
		})

		It("should Delete JSON bookmarks for user", func() {
			fmt.Println("=========] Delete JSON bookmarks for user")

			code, _, err := restyClient.Delete("/bookmarks", "BBB")
			fmt.Println("=========] should return nil error")
			Expect(err).Should(BeNil())
			fmt.Println("=========] should return 202 status code")
			Expect(code).To(Equal(202))
		})

		It("should Get JSON bookmarks for user", func() {
			fmt.Println("=========] Given a call")

			code, _, err := restyClient.Get("/bookmarks", "BBB", "application/json")

			fmt.Println("=========] should return 404 status code")
			Expect(code).To(Equal(404))

			fmt.Println("=========] should return nil error")
			Ω(err).Should(HaveOccurred())
			Expect(err).ShouldNot(MatchError(errors.New("bookmarks not found")))
		})
	})
})
