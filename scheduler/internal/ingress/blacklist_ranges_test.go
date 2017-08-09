package ingress_test

import (
	"net"

	"code.cloudfoundry.org/scalable-syslog/scheduler/internal/ingress"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BlacklistRanges", func() {
	Describe("validates", func() {
		It("accepts valid IP address range", func() {
			_, err := ingress.NewBlacklistRanges(
				ingress.BlacklistRange{Start: "127.0.2.2", End: "127.0.2.4"},
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error with an invalid start address", func() {
			_, err := ingress.NewBlacklistRanges(
				ingress.BlacklistRange{Start: "127.0.2.2.1", End: "127.0.2.4"},
			)
			Expect(err).To(MatchError("invalid IP Address for Blacklist IP Range: 127.0.2.2.1"))
		})

		It("returns an error with an invalid end address", func() {
			_, err := ingress.NewBlacklistRanges(
				ingress.BlacklistRange{Start: "127.0.2.2", End: "127.0.2.4.3"},
			)
			Expect(err).To(HaveOccurred())
		})

		It("validates multiple blacklist ranges", func() {
			_, err := ingress.NewBlacklistRanges(
				ingress.BlacklistRange{Start: "127.0.2.2", End: "127.0.2.4"},
				ingress.BlacklistRange{Start: "127.0.2.2", End: "127.0.2.4.5"},
			)
			Expect(err).To(HaveOccurred())
		})

		It("validates start IP is before end IP", func() {
			_, err := ingress.NewBlacklistRanges(
				ingress.BlacklistRange{Start: "10.10.10.10", End: "10.8.10.12"},
			)
			Expect(err).To(MatchError("invalid Blacklist IP Range: Start 10.10.10.10 has to be before End 10.8.10.12"))
		})

		It("accepts start and end as the same", func() {
			_, err := ingress.NewBlacklistRanges(
				ingress.BlacklistRange{Start: "127.0.2.2", End: "127.0.2.2"},
			)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("CheckBlacklist()", func() {
		It("allows all urls for empty blacklist range", func() {
			ranges, _ := ingress.NewBlacklistRanges()

			err := ranges.CheckBlacklist(net.ParseIP("127.0.0.1"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error when the IP is in the blacklist range", func() {
			ranges, err := ingress.NewBlacklistRanges(
				ingress.BlacklistRange{Start: "127.0.1.2", End: "127.0.3.4"},
			)
			Expect(err).ToNot(HaveOccurred())

			err = ranges.CheckBlacklist(net.ParseIP("127.0.2.2"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ParseHost()", func() {
		It("does not return an error on valid URL", func() {
			ranges, _ := ingress.NewBlacklistRanges()

			for _, testUrl := range validIPs {
				host, err := ranges.ParseHost(testUrl)
				Expect(err).ToNot(HaveOccurred())
				Expect(host).ToNot(Equal(""))
			}
		})

		It("returns error on malformatted URL", func() {
			ranges, _ := ingress.NewBlacklistRanges()

			for _, testUrl := range malformattedURLs {
				host, err := ranges.ParseHost(testUrl)
				Expect(err).To(HaveOccurred())
				Expect(host).To(Equal(""))
			}
		})
	})

	Describe("ResolveAddr()", func() {
		It("does not return an error when able to resolve", func() {
			ranges, _ := ingress.NewBlacklistRanges()

			ip, err := ranges.ResolveAddr("localhost")
			Expect(err).ToNot(HaveOccurred())
			Expect(ip.String()).To(Equal("127.0.0.1"))
		})

		It("returns an error when it fails to resolve", func() {
			ranges, _ := ingress.NewBlacklistRanges()

			_, err := ranges.ResolveAddr("vcap.me.junky-garbage")
			Expect(err).To(HaveOccurred())
		})
	})
})

var validIPs = []string{
	"http://127.0.0.1",
	"http://127.0.1.1",
	"http://127.0.3.5",
	"https://127.0.1.1",
	"syslog://127.0.1.1",
	"syslog://127.0.1.1:3000",
	"syslog://127.0.1.1:3000/test",
	"syslog://127.0.1.1:3000?app=great",
}

var invalidIPs = []string{
	"http://127.0.2.2",
	"http://127.0.2.3",
	"http://127.0.2.4",
	"https://127.0.2.3",
	"syslog://127.0.2.3",
	"syslog://127.0.2.3:3000",
	"syslog://127.0.2.3:3000/test",
	"syslog://127.0.2.3:3000?app=great",
	"://127.0.2.3:3000?app=great",
}

var malformattedURLs = []string{
	"127.0.0.1:300/new",
	"syslog:127.0.0.1:300/new",
	"<nil>",
}
