package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/adrg/xdg"

	"github.com/achannarasappa/ticker/v5/internal/brokerage/snaptrade"
	"github.com/achannarasappa/ticker/v5/internal/cli/symbol"
	c "github.com/achannarasappa/ticker/v5/internal/common"
)

// buildSnapTradeGroups fetches the user's brokerage accounts and positions from
// SnapTrade and returns one asset group per account. Every failure degrades to
// an empty result so ticker always starts; warnings are emitted only in debug mode.
func buildSnapTradeGroups(d c.Dependencies, config c.Config, tickerSymbolToSourceSymbol symbol.TickerSymbolToSourceSymbol) []c.AssetGroup {

	st := config.SnapTrade

	if st.ClientID == "" || st.ConsumerKey == "" {
		return nil
	}

	userID, userSecret := "", ""

	if !st.IsPersonal() {
		if st.UserID == "" {
			logSnapTradeWarning(config, "commercial keys require `user-id`", nil)

			return nil
		}

		secret, ok, err := snaptrade.NewStore(d.Fs, xdg.DataHome).GetUserSecret(st.UserID)
		if err != nil || !ok {
			logSnapTradeWarning(config, "not connected — run `ticker snaptrade connect`", err)

			return nil
		}

		userID, userSecret = st.UserID, secret
	}

	client := snaptrade.New(d.SnapTradeBaseURL, st.ClientID, st.ConsumerKey)

	accounts, err := client.ListAccounts(userID, userSecret)
	if err != nil {
		logSnapTradeWarning(config, "unable to list accounts", err)

		return nil
	}

	groups := make([]c.AssetGroup, 0, len(accounts))

	for _, account := range accounts {
		positions, err := client.GetPositions(userID, userSecret, account.ID)
		if err != nil {
			logSnapTradeWarning(config, "unable to get positions", err)

			continue
		}

		// Skip accounts with no equity holdings (e.g. a crypto-only account returns
		// nothing here) rather than showing an empty tab.
		if stockConfig := snaptrade.TransformToConfigAssetGroup(account, positions); len(stockConfig.Holdings) > 0 {
			group := buildAssetGroup(stockConfig, tickerSymbolToSourceSymbol)
			group.IsSnapTrade = true
			groups = append(groups, group)
		}

		// Options come from a separate endpoint and get their own group per account so
		// they don't collide with equity holdings on the same underlying symbol.
		optionPositions, err := client.ListOptionHoldings(userID, userSecret, account.ID)
		if err != nil {
			logSnapTradeWarning(config, "unable to get option holdings", err)

			continue
		}

		if optionsConfig, ok := snaptrade.TransformToOptionsConfigAssetGroup(account, optionPositions); ok {
			optionsGroup := buildAssetGroup(optionsConfig, tickerSymbolToSourceSymbol)
			optionsGroup.IsSnapTrade = true
			groups = append(groups, optionsGroup)
		}
	}

	return groups
}

// RefreshSnapTradeGroups re-fetches SnapTrade accounts and positions and returns
// the rebuilt asset groups. Used by the UI's on-demand refresh keybinding.
func RefreshSnapTradeGroups(d c.Dependencies, config c.Config) ([]c.AssetGroup, error) {

	tickerSymbolToSourceSymbol, err := symbol.GetTickerSymbols(d.SymbolsURL)
	if err != nil {
		return nil, err
	}

	return buildSnapTradeGroups(d, config, tickerSymbolToSourceSymbol), nil
}

// ConnectSnapTrade runs the one-time connection flow: register the user if
// needed, generate a connection-portal link, and open it in the browser. The
// browser opener is injected for testability.
func ConnectSnapTrade(d c.Dependencies, config c.Config, openBrowser func(string) error) error {

	st := config.SnapTrade

	var missing []string
	if st.ClientID == "" {
		missing = append(missing, "client-id")
	}
	if st.ConsumerKey == "" {
		missing = append(missing, "consumer-key")
	}
	if !st.IsPersonal() && st.UserID == "" {
		missing = append(missing, "user-id")
	}
	if len(missing) > 0 {
		return fmt.Errorf("snaptrade: missing required config under `snaptrade:` in .ticker.yml: %s", strings.Join(missing, ", ")) //nolint:goerr113
	}

	client := snaptrade.New(d.SnapTradeBaseURL, st.ClientID, st.ConsumerKey)

	userID, userSecret := "", ""

	// Commercial keys register (once) and persist a user secret. Personal keys are
	// already provisioned with their single user, so registration is skipped.
	if !st.IsPersonal() {
		store := snaptrade.NewStore(d.Fs, xdg.DataHome)

		secret, ok, err := store.GetUserSecret(st.UserID)
		if err != nil {
			return err
		}

		if !ok {
			secret, err = client.RegisterUser(st.UserID)
			if err != nil {
				return fmt.Errorf("snaptrade: register user: %w", err)
			}

			if err := store.SaveUserSecret(st.UserID, secret); err != nil {
				return fmt.Errorf("snaptrade: save user secret: %w", err)
			}
		}

		userID, userSecret = st.UserID, secret
	}

	redirectURI, err := client.GetLoginRedirectURI(userID, userSecret)
	if err != nil {
		return fmt.Errorf("snaptrade: generate connection link: %w", err)
	}

	fmt.Println("Open the following URL to connect your brokerage:")
	fmt.Println()
	fmt.Println("  " + redirectURI)
	fmt.Println()

	if openBrowser != nil {
		if err := openBrowser(redirectURI); err != nil {
			fmt.Fprintln(os.Stderr, "(unable to open browser automatically — open the URL above manually)")
		}
	}

	fmt.Println("After authorizing in the browser, run `ticker` to see your holdings.")

	return nil
}

// OpenBrowser opens targetURL in the user's default browser.
func OpenBrowser(targetURL string) error {

	var name string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		name = "open"
	case "windows":
		name = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	default:
		name = "xdg-open"
	}

	args = append(args, targetURL)

	return exec.Command(name, args...).Start()
}

func logSnapTradeWarning(config c.Config, message string, err error) {

	if !config.Debug {
		return
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "snaptrade: %s: %v\n", message, err)

		return
	}

	fmt.Fprintf(os.Stderr, "snaptrade: %s\n", message)
}
