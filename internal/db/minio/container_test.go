package minio_test

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/db/minio"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MinioContainer", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("NewMinioContainer", func() {
		It("returns error if minio.New fails", func() {
			// Use an invalid endpoint to force minio.New to fail
			config := minio.MinioContainerConfig{}
			_, err := minio.NewMinioContainer(ctx, config, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error creating minio client"))
		})
	})
})
