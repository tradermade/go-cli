package api

import (
	"context"
	"net/url"
)

// SymbolsResponse lists available codes for an asset class.
// available_currencies maps code → human name (e.g. "EUR" → "Euro").
type SymbolsResponse struct {
	AvailableCurrencies map[string]string `json:"available_currencies"`
	Endpoint            string            `json:"endpoint"`
}

// LiveCurrenciesList returns the currency codes usable with the live and
// convert endpoints.
func (c *Client) LiveCurrenciesList(ctx context.Context) (*SymbolsResponse, error) {
	return c.symbolList(ctx, "/live_currencies_list")
}

// LiveCryptoList returns the cryptocurrency codes usable with live endpoints.
func (c *Client) LiveCryptoList(ctx context.Context) (*SymbolsResponse, error) {
	return c.symbolList(ctx, "/live_crypto_list")
}

// SymbolListRaw returns a forex or crypto list response exactly as sent.
func (c *Client) SymbolListRaw(ctx context.Context, crypto bool) ([]byte, error) {
	path := "/live_currencies_list"
	if crypto {
		path = "/live_crypto_list"
	}
	return c.getBody(ctx, path, url.Values{})
}

func (c *Client) symbolList(ctx context.Context, path string) (*SymbolsResponse, error) {
	var out SymbolsResponse
	if err := c.get(ctx, path, url.Values{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
