// Package bitflyer provides functionality for bitFlyer api
// bitFlyer Lightningのapiクライアント
package bitflyer

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	BITFLYER_HOST = "api.bitflyer.jp"
	API_VERSION   = "v1"
)

type Client struct {
	URL        *url.URL
	HTTPClient *http.Client
	APIKey     string
	APISecret  string
}

func NewClient(apikey, apisecret string) *Client {
	u := &url.URL{Scheme: "https", Host: BITFLYER_HOST, Path: fmt.Sprintf("/%s", API_VERSION)}
	c := Client{URL: u, HTTPClient: &http.Client{}, APIKey: apikey, APISecret: apisecret}

	return &c
}

type Page struct {
	Count  int
	Before int
	After  int
}

func (p *Page) setPage(v url.Values) {
	if p.Count != 0 {
		v.Set("count", strconv.Itoa(p.Count))
	}
	if p.Before != 0 {
		v.Set("before", strconv.Itoa(p.Before))
	}
	if p.After != 0 {
		v.Set("after", strconv.Itoa(p.After))
	}
}

func (c *Client) newRequest(ctx context.Context, method, spath string, values url.Values, body io.Reader) (*http.Request, error) {
	u := *c.URL
	u.Path = path.Join(c.URL.Path, spath)
	u.RawQuery = values.Encode()
	log.Printf("[request URL] %#v\n", u.String())

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	if c.APIKey != "" && c.APISecret != "" {
		c.setAuthHeader(method, u.Path, body, req)
	}

	return req, nil
}

func (c *Client) setAuthHeader(method, path string, body io.Reader, req *http.Request) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	bodyBytes, _ := ioutil.ReadAll(body)
	text := timestamp + method + path + string(bodyBytes)
	sign := c.createHMAC(text, c.APISecret)
	req.Header.Set("ACCESS-KEY", c.APIKey)
	req.Header.Set("ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("ACCESS-SIGN", sign)
	req.Header.Set("Content-Type", "application/json")
}

func (c *Client) createHMAC(msg, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Client) getResponse(req *http.Request) (*http.Response, error) {
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	} else if res.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("status code: %d", res.StatusCode))
	}

	return res, nil
}

// * HTTP Public API
// ** マーケットの一覧
type Markets []struct {
	ProductCode string `json:"product_code"`
	Alias       string `json:"alias,omitempty"`
}

func (c *Client) GetMarkets(ctx context.Context) (*Markets, error) {
	req, err := c.newRequest(ctx, "GET", "markets", nil, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Markets
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// ** 板情報
type Board struct {
	MidPrice float64 `json:"mid_price"`
	Bids []struct {
		Price float64 `json:"price"`
		Size  float64 `json:"size"`
	} `json:"bids"`
	Asks []struct {
		Price float64 `json:"price"`
		Size  float64 `json:"size"`
	} `json:"asks"`
}

func (c *Client) GetBoard(ctx context.Context, productCode string) (*Board, error) {
	v := url.Values{}
	if productCode != "" {
		v.Set("product_code", productCode)
	}
	req, err := c.newRequest(ctx, "GET", "board", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Board
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// ** Ticker
type Ticker struct {
	ProductCode     string  `json:"product_code"`
	Timestamp       string  `json:"timestamp"`
	TickID          int     `json:"tick_id"`
	BestBid         float64 `json:"best_bid"`
	BestAsk         float64 `json:"best_ask"`
	BestBidSize     float64 `json:"best_bid_size"`
	BestAskSize     float64 `json:"best_ask_size"`
	TotalBidDepth   float64 `json:"total_bid_depth"`
	TotalAskDepth   float64 `json:"total_ask_depth"`
	Ltp             float64 `json:"ltp"`
	Volume          float64 `json:"volume"`
	VolumeByProduct float64 `json:"volume_by_product"`
}

func (c *Client) GetTicker(ctx context.Context, productCode string) (*Ticker, error) {
	v := url.Values{}
	if productCode != "" {
		v.Set("product_code", productCode)
	}
	req, err := c.newRequest(ctx, "GET", "ticker", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Ticker
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// ** 約定履歴
type Executions []struct {
	ID                         int     `json:"id"`
	ChildOrderID               string  `json:"child_order_id"`
	Side                       string  `json:"side"`
	Price                      float64 `json:"price"`
	Size                       float64 `json:"size"`
	Commission                 int     `json:"commission"`
	ExecDate                   string  `json:"exec_date"`
	BuyChildOrderAcceptanceID  string  `json:"buy_child_order_acceptance_id"`
	SellChildOrderAcceptanceID string  `json:"sell_child_order_acceptance_id"`
	ChildOrderAcceptanceID     string  `json:"child_order_acceptance_id"`
}

func (c *Client) GetExecutions(ctx context.Context, productCode string, page *Page) (*Executions, error) {
	v := url.Values{}
	if productCode != "" {
		v.Set("product_code", productCode)
	}
	if page != nil {
		page.setPage(v)
	}
	req, err := c.newRequest(ctx, "GET", "executions", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Executions
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// ** 取引所の状態
type Status struct {
	Status string `json:"status"`
}

func (c *Client) GetHealth(ctx context.Context) (*Status, error) {
	req, err := c.newRequest(ctx, "GET", "gethealth", nil, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Status
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// ** チャット
type Chats []struct {
	Nickname string `json:"nickname"`
	Message  string `json:"message"`
	Date     string `json:"date"`
}

func (c *Client) GetChats(ctx context.Context, fromDate string) (*Chats, error) {
	v := url.Values{}
	if fromDate != "" {
		v.Set("from_date", fromDate)
	}
	req, err := c.newRequest(ctx, "GET", "getchats", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Chats
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// * HTTP Private API
// ** API
// *** APIキーの権限を取得
type Permissions []string

func (c *Client) GetMyPermissions(ctx context.Context) (*Permissions, error) {
	req, err := c.newRequest(ctx, "GET", "me/getpermissions", nil, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Permissions
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// ** 資産
// *** 資産残高を取得
type Balance []struct {
	CurrencyCode string  `json:"currency_code"`
	Amount       float64 `json:"amount"`
	Available    float64 `json:"available"`
}

func (c *Client) GetMyBalance(ctx context.Context) (*Balance, error) {
	req, err := c.newRequest(ctx, "GET", "me/getbalance", nil, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Balance
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 証拠金の状態を取得
type Collateral struct {
	Collateral        int     `json:"collateral"`
	OpenPositionPnl   int     `json:"open_position_pnl"`
	RequireCollateral int     `json:"require_collateral"`
	KeepRate          float64 `json:"keep_rate"`
}

func (c *Client) GetMyCollateral(ctx context.Context) (*Collateral, error) {
	req, err := c.newRequest(ctx, "GET", "me/getcollateral", nil, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Collateral
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// ** 入出金
// *** 預入用ビットコイン・イーサリアムアドレス取得
type Address []struct {
	Type         string `json:"type"`
	CurrencyCode string `json:"currency_code"`
	Address      string `json:"address"`
}

func (c *Client) GetMyAddress(ctx context.Context) (*Address, error) {
	req, err := c.newRequest(ctx, "GET", "me/getaddress", nil, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Address
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** ビットコイン・イーサ預入履歴
type Coinins []struct {
	ID           int     `json:"id"`
	OrderID      string  `json:"order_id"`
	CurrencyCode string  `json:"currency_code"`
	Amount       float64 `json:"amount"`
	Address      string  `json:"address"`
	TxHash       string  `json:"tx_hash"`
	Status       string  `json:"status"`
	EventDate    string  `json:"event_date"`
}

func (c *Client) GetMyCoinins(ctx context.Context, page *Page) (*Coinins, error) {
	v := url.Values{}
	if page != nil {
		page.setPage(v)
	}
	req, err := c.newRequest(ctx, "GET", "me/getcoinins", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Coinins
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** ビットコイン・イーサ送付履歴
type Coinouts []struct {
	ID            int     `json:"id"`
	OrderID       string  `json:"order_id"`
	CurrencyCode  string  `json:"currency_code"`
	Amount        float64 `json:"amount"`
	Address       string  `json:"address"`
	TxHash        string  `json:"tx_hash"`
	Fee           float64 `json:"fee"`
	AdditionalFee float64 `json:"additional_fee"`
	Status        string  `json:"status"`
	EventDate     string  `json:"event_date"`
}

func (c *Client) GetMyCoinouts(ctx context.Context, page *Page, messageID string) (*Coinouts, error) {
	v := url.Values{}
	if page != nil {
		page.setPage(v)
	}
	if messageID != "" {
		v.Set("message_id", messageID)
	}
	req, err := c.newRequest(ctx, "GET", "me/getcoinouts", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Coinouts
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 銀行口座一覧取得
type BankAccounts []struct {
	ID            int    `json:"id"`
	IsVerified    bool   `json:"is_verified"`
	BankName      string `json:"bank_name"`
	BranchName    string `json:"branch_name"`
	AccountType   string `json:"account_type"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
}

func (c *Client) GetMyBankAccounts(ctx context.Context) (*BankAccounts, error) {
	req, err := c.newRequest(ctx, "GET", "me/getcoinins", nil, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data BankAccounts
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 入金履歴
type Deposits []struct {
	ID           int    `json:"id"`
	OrderID      string `json:"order_id"`
	CurrencyCode string `json:"currency_code"`
	Amount       int    `json:"amount"`
	Status       string `json:"status"`
	EventDate    string `json:"event_date"`
}

func (c *Client) GetMyDeposits(ctx context.Context, page *Page) (*Deposits, error) {
	v := url.Values{}
	if page != nil {
		page.setPage(v)
	}
	req, err := c.newRequest(ctx, "GET", "me/getdeposits", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Deposits
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 出金
type Withdraw struct {
	CurrencyCode  string `json:"currency_code"`
	BankAccountID int    `json:"bank_account_id"`
	Amount        int    `json:"amount"`
	Code          string `json:"code"`
}
type WithdrawResponse struct {
	MessageID    string      `json:"message_id"`
	Status       int         `json:"status"`
	ErrorMessage string      `json:"error_message"`
	Data         interface{} `json:"data"`
}

func (c *Client) Withdraw(ctx context.Context, wd *Withdraw) (*WithdrawResponse, error) {
	body, err := json.Marshal(&wd)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, "POST", "me/withdraw", nil, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data WithdrawResponse
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 出金履歴
type Withdrawals []struct {
	ID           int    `json:"id"`
	OrderID      string `json:"order_id"`
	CurrencyCode string `json:"currency_code"`
	Amount       int    `json:"amount"`
	Status       string `json:"status"`
	EventDate    string `json:"event_date"`
}

func (c *Client) GetMyWithdrawals(ctx context.Context, page *Page, messageID string) (*Withdrawals, error) {
	v := url.Values{}
	if page != nil {
		page.setPage(v)
	}
	if messageID != "" {
		v.Set("message_id", messageID)
	}
	req, err := c.newRequest(ctx, "GET", "me/getwithdrawals", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Withdrawals
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// ** トレード
// *** 新規注文を出す
type Childorder struct {
	ProductCode            string  `json:"product_code"`
	ChildOrderType         string  `json:"child_order_type"`
	Side                   string  `json:"side"`
	Price                  float64 `json:"price"`
	Size                   float64 `json:"size"`
	MinuteToExpire         int     `json:"minute_to_expire"`
	TimeInForce            string  `json:"time_in_force"`
	ChildOrderID           string  `json:"child_order_id"`
	ChildOrderAcceptanceID string  `json:"child_order_acceptance_id"`
}
type ChildOrderAcceptanceID struct {
	ChildOrderAcceptanceID string `json:"child_order_acceptance_id"`
}

func (c *Client) SendChildorder(ctx context.Context, ch *Childorder) (*ChildOrderAcceptanceID, error) {
	body, err := json.Marshal(&ch)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, "POST", "me/sendchildorder", nil, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data ChildOrderAcceptanceID
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 注文をキャンセルする
func (c *Client) CancelChildorder(ctx context.Context, ch *Childorder) error {
	body, err := json.Marshal(&ch)
	if err != nil {
		return err
	}
	req, err := c.newRequest(ctx, "POST", "me/cancelchildorder", nil, bytes.NewReader(body))
	if err != nil {
		return err
	}

	_, err = c.getResponse(req)

	return err
}

// *** 新規の親注文を出す（特殊注文）
type Parentorder struct {
	ID             int    `json:"id"`
	ParentOrderID  string `json:"parent_order_id"`
	OrderMethod    string `json:"order_method"`
	MinuteToExpire int    `json:"minute_to_expire"`
	TimeInForce    string `json:"time_in_force"`
	Parameters     []struct {
		ProductCode   string  `json:"product_code"`
		ConditionType string  `json:"condition_type"`
		Side          string  `json:"side"`
		Price         float64 `json:"price"`
		Size          float64 `json:"size"`
		TriggerPrice  int     `json:"trigger_price,omitempty"`
		Offset        int     `json:"offset"`
	} `json:"parameters"`
	ParentOrderAcceptanceID string `json:"parent_order_acceptance_id"`
}
type ParentOrderAcceptanceID struct {
	ParentOrderAcceptanceID string `json:"parent_order_acceptance_id"`
}

func (c *Client) SendParentrder(ctx context.Context, pa *Parentorder) (*ParentOrderAcceptanceID, error) {
	body, err := json.Marshal(&pa)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, "POST", "me/sendparentorder", nil, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data ParentOrderAcceptanceID
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 親注文をキャンセルする
func (c *Client) CancelParentorder(ctx context.Context, pa *Parentorder) error {
	body, err := json.Marshal(&pa)
	if err != nil {
		return err
	}
	req, err := c.newRequest(ctx, "POST", "me/cancelparentorder", nil, bytes.NewReader(body))
	if err != nil {
		return err
	}

	_, err = c.getResponse(req)

	return err
}

// *** すべての注文をキャンセルする
func (c *Client) CancelAllChildorder(ctx context.Context, productCode string) error {
	body := `{"product_code": "' + productCode + '"}`
	req, err := c.newRequest(ctx, "POST", "me/cancelallchildorder", nil, strings.NewReader(body))
	if err != nil {
		return err
	}

	_, err = c.getResponse(req)

	return err
}

// *** 注文の一覧を取得
type Childorders []struct {
	ID                     int     `json:"id"`
	ChildOrderID           string  `json:"child_order_id"`
	ProductCode            string  `json:"product_code"`
	Side                   string  `json:"side"`
	ChildOrderType         string  `json:"child_order_type"`
	Price                  float64 `json:"price"`
	AveragePrice           float64 `json:"average_price"`
	Size                   float64 `json:"size"`
	ChildOrderState        string  `json:"child_order_state"`
	ExpireDate             string  `json:"expire_date"`
	ChildOrderDate         string  `json:"child_order_date"`
	ChildOrderAcceptanceID string  `json:"child_order_acceptance_id"`
	OutstandingSize        float64 `json:"outstanding_size"`
	CancelSize             float64 `json:"cancel_size"`
	ExecutedSize           float64 `json:"executed_size"`
	TotalCommission        int     `json:"total_commission"`
}

func (c *Client) GetMyChildorders(ctx context.Context, productCode string, page *Page, childOrderState, parentOrderID string) (*Childorders, error) {
	v := url.Values{}
	if productCode != "" {
		v.Set("product_code", productCode)
	}
	if page != nil {
		page.setPage(v)
	}
	if childOrderState != "" {
		v.Set("child_order_state", childOrderState)
	}
	if parentOrderID != "" {
		v.Set("parent_order_id", parentOrderID)
	}
	req, err := c.newRequest(ctx, "GET", "me/getchildorders", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Childorders
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 親注文の一覧を取得
type Parentorders []struct {
	ID                      int     `json:"id"`
	ParentOrderID           string  `json:"parent_order_id"`
	ProductCode             string  `json:"product_code"`
	Side                    string  `json:"side"`
	ParentOrderType         string  `json:"parent_order_type"`
	Price                   float64 `json:"price"`
	AveragePrice            float64 `json:"average_price"`
	Size                    float64 `json:"size"`
	ParentOrderState        string  `json:"parent_order_state"`
	ExpireDate              string  `json:"expire_date"`
	ParentOrderDate         string  `json:"parent_order_date"`
	ParentOrderAcceptanceID string  `json:"parent_order_acceptance_id"`
	OutstandingSize         float64 `json:"outstanding_size"`
	CancelSize              float64 `json:"cancel_size"`
	ExecutedSize            float64 `json:"executed_size"`
	TotalCommission         int     `json:"total_commission"`
}

func (c *Client) GetMyParentorders(ctx context.Context, productCode string, page *Page, parentOrderState string) (*Parentorders, error) {
	v := url.Values{}
	if productCode != "" {
		v.Set("product_code", productCode)
	}
	if page != nil {
		page.setPage(v)
	}
	if parentOrderState != "" {
		v.Set("parent_order_state", parentOrderState)
	}
	req, err := c.newRequest(ctx, "GET", "me/getparentorders", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Parentorders
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 親注文の詳細を取得
/*
type Parentorder struct {
	ID             int    `json:"id"`
	ParentOrderID  string `json:"parent_order_id"`
	OrderMethod    string `json:"order_method"`
	MinuteToExpire int    `json:"minute_to_expire"`
	Parameters     []struct {
		ProductCode   string  `json:"product_code"`
		ConditionType string  `json:"condition_type"`
		Side          string  `json:"side"`
		Price         float64 `json:"price"`
		Size          float64 `json:"size"`
		TriggerPrice  float64 `json:"trigger_price"`
		Offset        int     `json:"offset"`
	} `json:"parameters"`
	ParentOrderAcceptanceID string `json:"parent_order_acceptance_id"`
}
*/

func (c *Client) GetMyParentorder(ctx context.Context, parentOrderID, parentOrderAcceptanceID string) (*Parentorders, error) {
	v := url.Values{}
	if parentOrderID != "" {
		v.Set("parent_order_id", parentOrderID)
	} else if parentOrderAcceptanceID != "" {
		v.Set("parent_order_acceptance_id", parentOrderAcceptanceID)
	} else {
		return nil, errors.New("nothing parameter")
	}
	req, err := c.newRequest(ctx, "GET", "me/getparentorder", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Parentorders
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 約定の一覧を取得
/*
type Executions []struct {
	ID                     int     `json:"id"`
	ChildOrderID           string  `json:"child_order_id"`
	Side                   string  `json:"side"`
	Price                  float64 `json:"price"`
	Size                   float64 `json:"size"`
	Commission             int     `json:"commission"`
	ExecDate               string  `json:"exec_date"`
	ChildOrderAcceptanceID string  `json:"child_order_acceptance_id"`
}
*/

func (c *Client) GetMyExecutions(ctx context.Context, productCode string, page *Page, childOrderID, childOrderAcceptanceID string) (*Executions, error) {
	v := url.Values{}
	if productCode != "" {
		v.Set("product_code", productCode)
	}
	if page != nil {
		page.setPage(v)
	}
	if childOrderID != "" {
		v.Set("child_order_id", childOrderID)
	}
	if childOrderAcceptanceID != "" {
		v.Set("child_order_acceptance_id", childOrderAcceptanceID)
	}
	req, err := c.newRequest(ctx, "GET", "me/getexecutions", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Executions
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 建玉の一覧を取得
type Positions []struct {
	ProductCode         string  `json:"product_code"`
	Side                string  `json:"side"`
	Price               float64 `json:"price"`
	Size                float64 `json:"size"`
	Commission          int     `json:"commission"`
	SwapPointAccumulate int     `json:"swap_point_accumulate"`
	RequireCollateral   int     `json:"require_collateral"`
	OpenDate            string  `json:"open_date"`
	Leverage            int     `json:"leverage"`
	Pnl                 int     `json:"pnl"`
}

func (c *Client) GetMyPositions(ctx context.Context, productCode string) (*Positions, error) {
	v := url.Values{}
	if productCode != "" {
		v.Set("product_code", productCode)
	}
	req, err := c.newRequest(ctx, "GET", "me/getpositions", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data Positions
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

// *** 取引手数料を取得
type TradingCommission struct {
	CommissionRate float64 `json:"commission_rate"`
}

func (c *Client) GetMyTradingCommission(ctx context.Context, productCode string) (*TradingCommission, error) {
	v := url.Values{}
	if productCode != "" {
		v.Set("product_code", productCode)
	}
	req, err := c.newRequest(ctx, "GET", "me/gettradingcommission", v, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.getResponse(req)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	var data TradingCommission
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
