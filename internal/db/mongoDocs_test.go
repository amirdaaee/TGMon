package db_test

import (
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/brianvoe/gofakeit/v7"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MongoDocs", func() {
	Describe("MediaFileDoc", Label("MediaFileDoc"), func() {
		newDoc := func() db.MediaFileDoc {
			d := db.MediaFileDoc{}
			gofakeit.Struct(&d)
			return d
		}
		// ...

	},
	)
})
