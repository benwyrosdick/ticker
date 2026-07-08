package asset_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/achannarasappa/ticker/v5/internal/asset"
	c "github.com/achannarasappa/ticker/v5/internal/common"
)

var _ = Describe("Asset", func() {

	Describe("GetAssets", func() {
		It("should return assets", func() {
			inputContext := c.Context{}
			inputAssetGroupQuote := fixtureAssetGroupQuote

			expectedAssets := fixtureAssets
			expectedHoldingSummary := HoldingSummary{}

			outputAssets, outputHoldingSummary := GetAssets(inputContext, inputAssetGroupQuote)

			Expect(outputAssets).To(Equal(expectedAssets))
			Expect(outputHoldingSummary).To(Equal(expectedHoldingSummary))
		})

		When("there are lots", func() {
			It("should return assets with holdings and a summary of holdings", func() {
				inputContext := c.Context{}
				inputAssetGroupQuote := fixtureAssetGroupQuote
				inputAssetGroupQuote.AssetGroup.ConfigAssetGroup.Holdings = []c.Lot{
					{
						Symbol:    "TWKS",
						UnitCost:  100,
						Quantity:  10,
						FixedCost: 7,
					},
					{
						Symbol:    "TWKS",
						UnitCost:  75,
						Quantity:  10,
						FixedCost: 0,
					},
					{
						Symbol:    "MSFT",
						UnitCost:  400,
						Quantity:  10,
						FixedCost: 0,
					},
				}

				outputAssets, outputHoldingSummary := GetAssets(inputContext, inputAssetGroupQuote)

				Expect(outputAssets).To(HaveLen(3))

				Expect(outputAssets[0].Holding.Value).To(Equal(2200.0))
				Expect(outputAssets[0].Holding.Cost).To(Equal(1757.0))
				Expect(outputAssets[0].Holding.Quantity).To(Equal(20.0))
				Expect(outputAssets[0].Holding.UnitValue).To(Equal(110.0))
				Expect(outputAssets[0].Holding.UnitCost).To(Equal(87.85))
				Expect(outputAssets[0].Holding.Weight).To(Equal(50.0))

				Expect(outputAssets[1].Holding.Value).To(Equal(2200.0))
				Expect(outputAssets[1].Holding.Cost).To(Equal(4000.0))
				Expect(outputAssets[1].Holding.Quantity).To(Equal(10.0))
				Expect(outputAssets[1].Holding.UnitValue).To(Equal(220.0))
				Expect(outputAssets[1].Holding.UnitCost).To(Equal(400.0))
				Expect(outputAssets[1].Holding.Weight).To(Equal(50.0))

				Expect(outputAssets[2].Holding).To(Equal(c.Holding{}))

				Expect(outputHoldingSummary.Cost).To(Equal(5757.0))
				Expect(outputHoldingSummary.Value).To(Equal(4400.0))
				Expect(outputHoldingSummary.TotalChange.Amount).To(Equal(-1357.0))
				Expect(outputHoldingSummary.TotalChange.Percent).To(Equal(-23.57130449887094))
				Expect(outputHoldingSummary.DayChange.Amount).To(Equal(400.0))
				Expect(outputHoldingSummary.DayChange.Percent).To(Equal(9.090909090909092))
			})
		})

		When("there is a currency to convert", func() {

			inputContext := c.Context{
				Config: c.Config{
					Currency: "EUR",
				},
				Reference: c.Reference{},
			}
			inputAssetGroupQuote := fixtureAssetGroupQuote
			inputAssetGroupQuote.AssetQuotes = []c.AssetQuote{
				{
					Name:          "ThoughtWorks",
					Symbol:        "TWKS",
					Class:         c.AssetClassStock,
					Currency:      c.Currency{FromCurrencyCode: "USD", ToCurrencyCode: "EUR", Rate: 1.5},
					QuotePrice:    c.QuotePrice{Price: 110.0, PricePrevClose: 100.0, PriceOpen: 100.0, PriceDayHigh: 110.0, PriceDayLow: 90.0, Change: 10.0, ChangePercent: 10.0},
					QuoteExtended: c.QuoteExtended{FiftyTwoWeekHigh: 150, FiftyTwoWeekLow: 50, MarketCap: 1000000},
				},
				{
					Name:       "Microsoft Inc",
					Symbol:     "MSFT",
					Class:      c.AssetClassStock,
					Currency:   c.Currency{FromCurrencyCode: "GBP", ToCurrencyCode: "EUR", Rate: 2.0},
					QuotePrice: c.QuotePrice{Price: 220.0, PricePrevClose: 200.0, PriceOpen: 200.0, PriceDayHigh: 220.0, PriceDayLow: 180.0, Change: 20.0, ChangePercent: 10.0},
				},
			}
			inputAssetGroupQuote.AssetGroup.ConfigAssetGroup.Holdings = []c.Lot{
				{
					Symbol:    "TWKS",
					UnitCost:  100,
					Quantity:  10,
					FixedCost: 0,
				},
				{
					Symbol:    "MSFT",
					UnitCost:  100,
					Quantity:  1,
					FixedCost: 0,
				},
			}

			outputAssets, outputHoldingSummary := GetAssets(inputContext, inputAssetGroupQuote)

			It("should set currency from and to when converting", func() {
				Expect(outputAssets[0].Currency.FromCurrencyCode).To(Equal("USD"))
				Expect(outputAssets[0].Currency.ToCurrencyCode).To(Equal("EUR"))
			})
			It("should convert the currencies for quote", func() {
				Expect(outputAssets[0].QuotePrice.Price).To(Equal(165.0))
				Expect(outputAssets[0].QuotePrice.PricePrevClose).To(Equal(150.0))
				Expect(outputAssets[0].QuotePrice.PriceOpen).To(Equal(150.0))
				Expect(outputAssets[0].QuotePrice.PriceDayHigh).To(Equal(165.0))
				Expect(outputAssets[0].QuotePrice.PriceDayLow).To(Equal(135.0))
				Expect(outputAssets[0].QuotePrice.Change).To(Equal(15.0))
			})
			It("should convert the currencies for quote extended", func() {
				Expect(outputAssets[0].QuoteExtended.FiftyTwoWeekHigh).To(Equal(225.0))
				Expect(outputAssets[0].QuoteExtended.FiftyTwoWeekLow).To(Equal(75.0))
				Expect(outputAssets[0].QuoteExtended.MarketCap).To(Equal(1500000.0))
			})
			It("should convert holding values", func() {
				Expect(outputAssets[0].Holding.Value).To(Equal(1650.0))
				Expect(outputAssets[0].Holding.Cost).To(Equal(1500.0))
				Expect(outputAssets[0].Holding.UnitValue).To(Equal(165.0))
				Expect(outputAssets[0].Holding.UnitCost).To(Equal(150.0))
			})
			It("should convert holding summary values", func() {
				Expect(outputHoldingSummary.Cost).To(Equal(1700.0))
				Expect(outputHoldingSummary.Value).To(Equal(2090.0))
				Expect(outputHoldingSummary.TotalChange.Amount).To(Equal(390.0))
				Expect(outputHoldingSummary.DayChange.Amount).To(Equal(190.0))
			})

			When("and the summary conversion only option is set", func() {

				inputContextSummaryConversion := inputContext
				inputContextSummaryConversion.Config = c.Config{
					Currency:                   "EUR",
					CurrencyConvertSummaryOnly: true,
				}

				_, outputHoldingSummary := GetAssets(inputContextSummaryConversion, inputAssetGroupQuote)

				It("should convert holding summary values", func() {
					Expect(outputHoldingSummary.Cost).To(Equal(1700.0))
					Expect(outputHoldingSummary.Value).To(Equal(2090.0))
					Expect(outputHoldingSummary.TotalChange.Amount).To(Equal(390.0))
					Expect(outputHoldingSummary.DayChange.Amount).To(Equal(190.0))
				})

			})

			When("and the disable unit cost conversion option is set", func() {

				inputContextDisableUnitCostConversion := inputContext
				inputContextDisableUnitCostConversion.Config = c.Config{
					Currency:                          "EUR",
					CurrencyDisableUnitCostConversion: true,
				}

				outputAssets, outputHoldingSummary := GetAssets(inputContextDisableUnitCostConversion, inputAssetGroupQuote)

				It("should not convert holding costs", func() {
					Expect(outputAssets[0].Holding.Cost).To(Equal(1000.0)) // 1000 EUR unconverted since option is set
					Expect(outputAssets[0].Holding.UnitCost).To(Equal(100.0))
					Expect(outputAssets[0].Holding.Value).To(Equal(1650.0)) // Conversion 10 shares @ 110 USD/share to EUR
					Expect(outputAssets[0].Holding.TotalChange.Amount).To(Equal(650.0))
					Expect(outputAssets[0].Holding.TotalChange.Percent).To(Equal(65.0))
					Expect(outputHoldingSummary.Cost).To(Equal(1100.0)) // Sum of 1000 EUR + 100 EUR
					Expect(outputHoldingSummary.DayChange.Percent).To(Equal(9.090909090909092))
					Expect(outputHoldingSummary.TotalChange.Amount).To(Equal(990.0))
					Expect(outputHoldingSummary.TotalChange.Percent).To(Equal(90.0))
				})

			})

		})

		When("there is no explicit currency conversion", func() {
			When("and there is a holding with a non-US currency", func() {

				inputContext := c.Context{
					Reference: c.Reference{},
				}
				inputAssetGroupQuote := fixtureAssetGroupQuote
				inputAssetQuotes := make([]c.AssetQuote, len(fixtureAssetGroupQuote.AssetQuotes))
				copy(inputAssetQuotes, fixtureAssetGroupQuote.AssetQuotes)
				inputAssetQuotes[0].Currency.FromCurrencyCode = "EUR"
				inputAssetQuotes[0].Currency.ToCurrencyCode = "USD"
				inputAssetQuotes[0].Currency.Rate = 0.5
				inputAssetGroupQuote.AssetQuotes = inputAssetQuotes
				inputAssetGroupQuote.AssetGroup.ConfigAssetGroup.Holdings = []c.Lot{
					{
						Symbol:    "TWKS",
						UnitCost:  100,
						Quantity:  10,
						FixedCost: 0,
					},
				}

				outputAssets, outputHoldingSummary := GetAssets(inputContext, inputAssetGroupQuote)

				It("should convert the currency for the holding summary", func() {
					Expect(outputHoldingSummary.Value).To(Equal(550.0))
					Expect(outputHoldingSummary.Cost).To(Equal(500.0))
				})
				It("should not convert the currencies for quote", func() {
					Expect(outputAssets[0].QuotePrice.Price).To(Equal(110.0))
					Expect(outputAssets[0].QuoteExtended.MarketCap).To(Equal(1000000.0))
				})
				It("should not convert the currency for the holding", func() {
					Expect(outputAssets[0].Holding.Value).To(Equal(1100.0))
					Expect(outputAssets[0].Holding.Cost).To(Equal(1000.0))
				})
			})
		})

		When("there are no asset quotes", func() {
			It("should return empty assets and holdings summary", func() {
				outputAssets, outputHoldingSummary := GetAssets(c.Context{}, c.AssetGroupQuote{})

				Expect(outputAssets).To(Equal([]c.Asset{}))
				Expect(outputHoldingSummary).To(Equal(HoldingSummary{}))
			})
		})

		When("there are options", func() {

			// An underlying present only as an option (not held, not watchlisted) yields a single option row
			optionOnlyQuote := c.AssetGroupQuote{
				AssetGroup: c.AssetGroup{ConfigAssetGroup: c.ConfigAssetGroup{
					Options: []c.Option{
						{Symbol: "MSFT", StrikePrice: 200.0, Type: "call", Premium: 10.0, Contracts: 2},
					},
				}},
				AssetQuotes: []c.AssetQuote{
					{Name: "Microsoft Inc", Symbol: "MSFT", Class: c.AssetClassStock, Currency: c.Currency{FromCurrencyCode: "USD"}, QuotePrice: c.QuotePrice{Price: 220.0}},
				},
			}

			It("should return a single option row for an option-only underlying", func() {
				outputAssets, _ := GetAssets(c.Context{}, optionOnlyQuote)

				Expect(outputAssets).To(HaveLen(1))
				Expect(outputAssets[0].Symbol).To(Equal("MSFT"))
				Expect(outputAssets[0].Class).To(Equal(c.AssetClassOption))
				Expect(outputAssets[0].QuoteOption.StrikePrice).To(Equal(200.0))
				Expect(outputAssets[0].QuoteOption.BreakevenPrice).To(Equal(210.0)) // call: strike + premium
				Expect(outputAssets[0].QuoteOption.DiffToStrike).To(Equal(20.0))    // 220 - 200
			})

			It("should show a held underlying and its option as separate rows", func() {
				inputAssetGroupQuote := c.AssetGroupQuote{
					AssetGroup: c.AssetGroup{ConfigAssetGroup: c.ConfigAssetGroup{
						Holdings: []c.Lot{{Symbol: "MSFT", Quantity: 5, UnitCost: 150}},
						Options:  []c.Option{{Symbol: "MSFT", StrikePrice: 200.0, Type: "call", Premium: 10.0, Contracts: 2}},
					}},
					AssetQuotes: []c.AssetQuote{
						{Name: "Microsoft Inc", Symbol: "MSFT", Class: c.AssetClassStock, Currency: c.Currency{FromCurrencyCode: "USD"}, QuotePrice: c.QuotePrice{Price: 220.0, Change: 20.0, ChangePercent: 10.0}},
					},
				}

				outputAssets, _ := GetAssets(c.Context{}, inputAssetGroupQuote)

				Expect(outputAssets).To(HaveLen(2))
				Expect(outputAssets[0].Class).To(Equal(c.AssetClassStock))
				Expect(outputAssets[0].Holding.Quantity).To(Equal(5.0))
				Expect(outputAssets[1].Class).To(Equal(c.AssetClassOption))
				Expect(outputAssets[1].QuoteOption.StrikePrice).To(Equal(200.0))
			})

			It("should show multiple options on the same underlying as separate distinct rows", func() {
				inputAssetGroupQuote := c.AssetGroupQuote{
					AssetGroup: c.AssetGroup{ConfigAssetGroup: c.ConfigAssetGroup{
						Options: []c.Option{
							{Symbol: "TSLA", StrikePrice: 400.0, Type: "call", Premium: 5.0, Contracts: 1, Expiration: "2026-07-10"},
							{Symbol: "TSLA", StrikePrice: 450.0, Type: "put", Premium: 8.0, Contracts: 2, Expiration: "2026-08-15"},
						},
					}},
					AssetQuotes: []c.AssetQuote{
						{Name: "Tesla", Symbol: "TSLA", Class: c.AssetClassStock, Currency: c.Currency{FromCurrencyCode: "USD"}, QuotePrice: c.QuotePrice{Price: 420.0}},
					},
				}

				outputAssets, _ := GetAssets(c.Context{}, inputAssetGroupQuote)

				Expect(outputAssets).To(HaveLen(2))
				Expect(outputAssets[0].QuoteOption.StrikePrice).To(Equal(400.0))
				Expect(outputAssets[0].QuoteOption.BreakevenPrice).To(Equal(405.0)) // call: 400 + 5
				Expect(outputAssets[1].QuoteOption.StrikePrice).To(Equal(450.0))
				Expect(outputAssets[1].QuoteOption.BreakevenPrice).To(Equal(442.0)) // put: 450 - 8
				Expect(outputAssets[0].Name).NotTo(Equal(outputAssets[1].Name))
			})

			It("should keep a watchlisted underlying visible alongside its option", func() {
				inputAssetGroupQuote := fixtureAssetGroupQuote
				inputAssetGroupQuote.AssetGroup.ConfigAssetGroup.Options = []c.Option{
					{Symbol: "MSFT", StrikePrice: 200.0, Type: "call", Premium: 10.0, Contracts: 2},
				}

				outputAssets, _ := GetAssets(c.Context{}, inputAssetGroupQuote)

				// TWKS, MSFT (watchlist) + MSFT (option) + SOL1-USD
				Expect(outputAssets).To(HaveLen(4))

				var stockCount, optionCount int
				for _, asset := range outputAssets {
					if asset.Symbol == "MSFT" && asset.Class == c.AssetClassStock {
						stockCount++
					}
					if asset.Symbol == "MSFT" && asset.Class == c.AssetClassOption {
						optionCount++
					}
				}
				Expect(stockCount).To(Equal(1))
				Expect(optionCount).To(Equal(1))
			})
		})

	})
})
