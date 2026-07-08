package cli_test

import (
	"net/http"

	"github.com/adrg/xdg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/spf13/afero"

	"github.com/achannarasappa/ticker/v5/internal/brokerage/snaptrade"
	"github.com/achannarasappa/ticker/v5/internal/cli"
	c "github.com/achannarasappa/ticker/v5/internal/common"
)

var _ = Describe("SnapTrade", func() {

	var (
		server *ghttp.Server
		fs     afero.Fs
		dep    c.Dependencies
		config c.Config
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		fs = afero.NewMemMapFs()
		dep = c.Dependencies{
			Fs:               fs,
			SymbolsURL:       server.URL() + "/symbols.csv",
			SnapTradeBaseURL: server.URL(),
		}
		config = c.Config{
			SnapTrade: c.ConfigSnapTrade{
				ClientID:    "client-id",
				ConsumerKey: "consumer-key",
				UserID:      "ben",
			},
		}

		server.RouteToHandler("GET", "/symbols.csv",
			ghttp.RespondWith(http.StatusOK, `"BTC.X","BTC-USD","cb"`, http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}}),
		)
	})

	AfterEach(func() {
		server.Close()
	})

	seedSecret := func() {
		Expect(snaptrade.NewStore(fs, xdg.DataHome).SaveUserSecret("ben", "secret")).To(Succeed())
	}

	Describe("RefreshSnapTradeGroups", func() {

		When("the user is connected", func() {
			It("should return one SnapTrade-marked group per account with holdings from positions", func() {
				seedSecret()
				server.RouteToHandler("GET", "/accounts",
					ghttp.RespondWithJSONEncoded(http.StatusOK, []snaptrade.Account{
						{ID: "acct-1", Name: "Robinhood Individual"},
					}),
				)
				server.RouteToHandler("GET", "/accounts/acct-1/positions",
					ghttp.RespondWithJSONEncoded(http.StatusOK, []snaptrade.Position{
						{Symbol: &snaptrade.PositionSymbol{Symbol: &snaptrade.UniversalSymbol{Symbol: "AAPL"}}, Units: 10, AveragePurchasePrice: 150},
					}),
				)
				server.RouteToHandler("GET", "/accounts/acct-1/options",
					ghttp.RespondWithJSONEncoded(http.StatusOK, []snaptrade.OptionsPosition{}),
				)

				groups, err := cli.RefreshSnapTradeGroups(dep, config)

				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(1))
				Expect(groups[0].IsSnapTrade).To(BeTrue())
				Expect(groups[0].Name).To(Equal("Robinhood Individual"))
				Expect(groups[0].Holdings).To(Equal([]c.Lot{{Symbol: "AAPL", Quantity: 10, UnitCost: 150}}))
				Expect(groups[0].SymbolsBySource).To(HaveLen(1))
				Expect(groups[0].SymbolsBySource[0].Source).To(Equal(c.QuoteSourceYahoo))
			})
		})

		When("an account holds options", func() {
			It("should add a separate options group per account", func() {
				seedSecret()
				server.RouteToHandler("GET", "/accounts",
					ghttp.RespondWithJSONEncoded(http.StatusOK, []snaptrade.Account{{ID: "acct-1", Name: "Robinhood Individual"}}),
				)
				server.RouteToHandler("GET", "/accounts/acct-1/positions",
					ghttp.RespondWithJSONEncoded(http.StatusOK, []snaptrade.Position{
						{Symbol: &snaptrade.PositionSymbol{Symbol: &snaptrade.UniversalSymbol{Symbol: "AAPL"}}, Units: 10, AveragePurchasePrice: 150},
					}),
				)
				server.RouteToHandler("GET", "/accounts/acct-1/options",
					ghttp.RespondWithJSONEncoded(http.StatusOK, []snaptrade.OptionsPosition{
						{
							Symbol: &snaptrade.OptionBrokerageSymbol{OptionSymbol: &snaptrade.OptionsSymbol{
								OptionType:       "PUT",
								StrikePrice:      255,
								UnderlyingSymbol: &snaptrade.UnderlyingSymbol{Symbol: "AAPL"},
							}},
							Units:                2,
							AveragePurchasePrice: 2.7,
						},
					}),
				)

				groups, err := cli.RefreshSnapTradeGroups(dep, config)

				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(2))
				Expect(groups[0].Name).To(Equal("Robinhood Individual"))
				Expect(groups[1].Name).To(Equal("Robinhood Individual Options"))
				Expect(groups[1].IsSnapTrade).To(BeTrue())
				Expect(groups[1].Options).To(Equal([]c.Option{
					{Symbol: "AAPL", StrikePrice: 255, Type: "put", Premium: 2.7, Contracts: 2},
				}))
			})
		})

		When("using personal keys (no user-id, no stored secret)", func() {
			It("should fetch holdings without registering or a stored secret", func() {
				personalConfig := c.Config{SnapTrade: c.ConfigSnapTrade{ClientID: "client-id", ConsumerKey: "consumer-key"}}

				server.RouteToHandler("GET", "/accounts", ghttp.CombineHandlers(
					func(_ http.ResponseWriter, req *http.Request) {
						defer GinkgoRecover()
						Expect(req.URL.Query().Has("userId")).To(BeFalse())
					},
					ghttp.RespondWithJSONEncoded(http.StatusOK, []snaptrade.Account{{ID: "acct-1", Name: "Robinhood"}}),
				))
				server.RouteToHandler("GET", "/accounts/acct-1/positions",
					ghttp.RespondWithJSONEncoded(http.StatusOK, []snaptrade.Position{
						{Symbol: &snaptrade.PositionSymbol{Symbol: &snaptrade.UniversalSymbol{Symbol: "AAPL"}}, Units: 5, AveragePurchasePrice: 100},
					}),
				)
				server.RouteToHandler("GET", "/accounts/acct-1/options",
					ghttp.RespondWithJSONEncoded(http.StatusOK, []snaptrade.OptionsPosition{}),
				)

				groups, err := cli.RefreshSnapTradeGroups(dep, personalConfig)

				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(1))
				Expect(groups[0].Holdings).To(Equal([]c.Lot{{Symbol: "AAPL", Quantity: 5, UnitCost: 100}}))
			})
		})

		When("SnapTrade is not configured", func() {
			It("should return no groups", func() {
				groups, err := cli.RefreshSnapTradeGroups(dep, c.Config{})

				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			})
		})

		When("the user has not connected (no stored secret)", func() {
			It("should return no groups", func() {
				groups, err := cli.RefreshSnapTradeGroups(dep, config)

				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			})
		})

		When("the SnapTrade API returns an error", func() {
			It("should degrade to no groups without erroring", func() {
				seedSecret()
				server.RouteToHandler("GET", "/accounts",
					ghttp.RespondWith(http.StatusInternalServerError, ""),
				)

				groups, err := cli.RefreshSnapTradeGroups(dep, config)

				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			})
		})
	})

	Describe("ConnectSnapTrade", func() {

		When("required config is missing", func() {
			It("should return a helpful error naming the missing fields", func() {
				err := cli.ConnectSnapTrade(dep, c.Config{SnapTrade: c.ConfigSnapTrade{AccountType: "commercial"}}, func(string) error { return nil })

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("client-id"))
				Expect(err.Error()).To(ContainSubstring("consumer-key"))
				Expect(err.Error()).To(ContainSubstring("user-id"))
			})
		})

		When("using personal keys", func() {
			It("should skip registration and open the connection URL", func() {
				personalConfig := c.Config{SnapTrade: c.ConfigSnapTrade{ClientID: "client-id", ConsumerKey: "consumer-key"}}

				registered := false
				server.RouteToHandler("POST", "/snapTrade/registerUser", func(_ http.ResponseWriter, _ *http.Request) {
					registered = true
				})
				server.RouteToHandler("POST", "/snapTrade/login",
					ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]string{"redirectURI": "https://app.snaptrade.com/connect/xyz"}),
				)

				var opened string
				err := cli.ConnectSnapTrade(dep, personalConfig, func(uri string) error {
					opened = uri

					return nil
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(registered).To(BeFalse())
				Expect(opened).To(Equal("https://app.snaptrade.com/connect/xyz"))
			})
		})

		When("the user is not yet registered", func() {
			It("should register, persist the secret, and open the connection URL", func() {
				server.RouteToHandler("POST", "/snapTrade/registerUser",
					ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]string{"userId": "ben", "userSecret": "new-secret"}),
				)
				server.RouteToHandler("POST", "/snapTrade/login",
					ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]string{"redirectURI": "https://app.snaptrade.com/connect/abc"}),
				)

				var opened string
				err := cli.ConnectSnapTrade(dep, config, func(uri string) error {
					opened = uri

					return nil
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(opened).To(Equal("https://app.snaptrade.com/connect/abc"))

				secret, ok, _ := snaptrade.NewStore(fs, xdg.DataHome).GetUserSecret("ben")
				Expect(ok).To(BeTrue())
				Expect(secret).To(Equal("new-secret"))
			})
		})
	})
})
