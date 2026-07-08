package snaptrade

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

var _ = Describe("Store", func() {

	var fs afero.Fs

	BeforeEach(func() {
		fs = afero.NewMemMapFs()
	})

	It("should return ok=false when no secret is stored", func() {
		secret, ok, err := NewStore(fs, "/data").GetUserSecret("ben")

		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
		Expect(secret).To(BeEmpty())
	})

	It("should persist and read back a user secret", func() {
		store := NewStore(fs, "/data")

		Expect(store.SaveUserSecret("ben", "secret-123")).To(Succeed())

		secret, ok, err := store.GetUserSecret("ben")

		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(secret).To(Equal("secret-123"))

		exists, _ := afero.Exists(fs, "/data/ticker/snaptrade.json")
		Expect(exists).To(BeTrue())
	})

	It("should preserve other users when saving", func() {
		store := NewStore(fs, "/data")

		Expect(store.SaveUserSecret("ben", "secret-ben")).To(Succeed())
		Expect(store.SaveUserSecret("alice", "secret-alice")).To(Succeed())

		benSecret, _, _ := store.GetUserSecret("ben")
		aliceSecret, _, _ := store.GetUserSecret("alice")

		Expect(benSecret).To(Equal("secret-ben"))
		Expect(aliceSecret).To(Equal("secret-alice"))
	})
})
