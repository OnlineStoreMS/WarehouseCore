package peers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	order   string
	after   string
	ship    string
	self    string
	http    *http.Client
}

func New(orderURL, afterURL, shipURL, selfURL string) *Client {
	return &Client{
		order: strings.TrimRight(orderURL, "/"),
		after: strings.TrimRight(afterURL, "/"),
		ship:  strings.TrimRight(shipURL, "/"),
		self:  strings.TrimRight(selfURL, "/"),
		http:  &http.Client{Timeout: 12 * time.Second},
	}
}

type Line struct {
	SkuCode string
	Name    string
	Qty     float64
}

type Fulfillment struct {
	Source    string
	OrderNo   string
	SourceNo  string
	Buyer     string
	Address   string
	Cancelled bool
	Refunded  bool
	Lines     []Line
}

type ReturnNotice struct {
	LogisticsNo string
	AftersaleNo string
	OrderNo     string
	Lines       []Line
}

func (c *Client) LookupFulfillment(ctx context.Context, bearer, code string) (*Fulfillment, error) {
	if c == nil {
		return nil, nil
	}
	code = strings.TrimSpace(code)
	if order, err := c.matchOrder(ctx, bearer, code); err != nil {
		return nil, err
	} else if order != nil {
		return order, nil
	}
	if selfOrder, err := c.matchSelf(ctx, bearer, code); err != nil {
		return nil, err
	} else if selfOrder != nil {
		return selfOrder, nil
	}
	shipOrderNo, err := c.matchShipmentOrderNo(ctx, bearer, code)
	if err != nil || shipOrderNo == "" {
		return nil, err
	}
	order, err := c.matchOrder(ctx, bearer, shipOrderNo)
	if err != nil || order == nil {
		return nil, err
	}
	order.Source = "shipping"
	order.SourceNo = code
	return order, nil
}

func (c *Client) LookupReturn(ctx context.Context, bearer, code string) (*ReturnNotice, error) {
	if c == nil || c.after == "" {
		return nil, nil
	}
	rows, err := c.getList(ctx, bearer, c.after+"/api/v1/admin/return-packages?keyword="+url.QueryEscape(code)+"&pageSize=20")
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	type row struct {
		PlatformAftersaleID string `json:"platformAftersaleId"`
		OrderNo             string `json:"orderNo"`
		LogisticsNo         string `json:"logisticsNo"`
		SKU                 string `json:"sku"`
		ProductTitle        string `json:"productTitle"`
		Qty                 int    `json:"qty"`
	}
	var exact *row
	for i := range rows {
		var item row
		if json.Unmarshal(rows[i], &item) != nil {
			continue
		}
		if strings.EqualFold(item.LogisticsNo, code) || strings.EqualFold(item.OrderNo, code) || strings.EqualFold(item.PlatformAftersaleID, code) {
			exact = &item
			break
		}
	}
	if exact == nil {
		return nil, nil
	}
	qty := exact.Qty
	if qty <= 0 {
		qty = 1
	}
	return &ReturnNotice{
		LogisticsNo: exact.LogisticsNo,
		AftersaleNo: exact.PlatformAftersaleID,
		OrderNo:     exact.OrderNo,
		Lines:       []Line{{SkuCode: exact.SKU, Name: exact.ProductTitle, Qty: float64(qty)}},
	}, nil
}

func (c *Client) matchOrder(ctx context.Context, bearer, code string) (*Fulfillment, error) {
	if c.order == "" {
		return nil, nil
	}
	rows, err := c.getList(ctx, bearer, c.order+"/api/v1/admin/orders?keyword="+url.QueryEscape(code)+"&pageSize=10")
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	type item struct {
		SkuCode           string  `json:"skuCode"`
		ProductName       string  `json:"productName"`
		Quantity          float64 `json:"quantity"`
		AllocType         string  `json:"allocType"`
		ParentOrderItemID uint64  `json:"parentOrderItemId"`
		AfterSaleStatus   string  `json:"afterSaleStatus"`
	}
	type order struct {
		ID              uint64 `json:"id"`
		OrderNo         string `json:"orderNo"`
		PlatformOrderID string `json:"platformOrderId"`
		Status          string `json:"status"`
		ShipStatus      string `json:"shipStatus"`
		AllocType       string `json:"allocType"`
		SelfOrderNo     string `json:"selfOrderNo"`
		AfterSaleStatus string `json:"afterSaleStatus"`
		BuyerName       string `json:"buyerName"`
		Address         *struct {
			FullText string `json:"fullText"`
			Address  string `json:"address"`
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
		} `json:"address"`
		Items []item `json:"items"`
	}
	for _, raw := range rows {
		var row order
		if json.Unmarshal(raw, &row) != nil {
			continue
		}
		if !strings.EqualFold(row.OrderNo, code) && !strings.EqualFold(row.PlatformOrderID, code) && !strings.EqualFold(row.SelfOrderNo, code) {
			continue
		}
		if len(row.Items) == 0 && row.ID > 0 {
			if raw, e := c.getData(ctx, bearer, fmt.Sprintf("%s/api/v1/admin/orders/%d", c.order, row.ID)); e == nil {
				_ = json.Unmarshal(raw, &row)
			}
		}
		self := row.AllocType == "self_ship" || row.SelfOrderNo != ""
		lines := make([]Line, 0)
		for _, it := range row.Items {
			if it.ParentOrderItemID > 0 || it.Quantity <= 0 {
				continue
			}
			if it.AllocType == "self_ship" || (it.AllocType == "" && self) {
				self = true
				lines = append(lines, Line{SkuCode: it.SkuCode, Name: it.ProductName, Qty: it.Quantity})
			}
			if strings.Contains(strings.ToUpper(it.AfterSaleStatus), "FINISH") {
				row.AfterSaleStatus = it.AfterSaleStatus
			}
		}
		if !self || len(lines) == 0 {
			continue
		}
		address := ""
		if row.Address != nil {
			address = strings.TrimSpace(row.Address.FullText)
			if address == "" {
				address = strings.TrimSpace(row.Address.Province + row.Address.City + row.Address.District + row.Address.Address)
			}
		}
		return &Fulfillment{
			Source:    "order",
			OrderNo:   row.OrderNo,
			SourceNo:  row.OrderNo,
			Buyer:     row.BuyerName,
			Address:   address,
			Cancelled: isClosed(row.Status) || isClosed(row.ShipStatus),
			Refunded:  strings.Contains(strings.ToUpper(row.AfterSaleStatus), "FINISH"),
			Lines:     lines,
		}, nil
	}
	return nil, nil
}

func (c *Client) matchSelf(ctx context.Context, bearer, code string) (*Fulfillment, error) {
	if c.self == "" {
		return nil, nil
	}
	rows, err := c.getList(ctx, bearer, c.self+"/api/v1/admin/self-orders?keyword="+url.QueryEscape(code)+"&pageSize=10")
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	type item struct {
		SkuCode     string  `json:"skuCode"`
		InvSkuCode  string  `json:"invSkuCode"`
		ProductName string  `json:"productName"`
		Qty         float64 `json:"qty"`
	}
	type order struct {
		ID      uint64 `json:"id"`
		SoNo    string `json:"soNo"`
		Status  string `json:"status"`
		Address string `json:"address"`
		Buyer   string `json:"buyerName"`
		Items   []item `json:"items"`
	}
	for _, raw := range rows {
		var row order
		if json.Unmarshal(raw, &row) != nil || !strings.EqualFold(row.SoNo, code) {
			continue
		}
		if len(row.Items) == 0 && row.ID > 0 {
			if raw, e := c.getData(ctx, bearer, fmt.Sprintf("%s/api/v1/admin/self-orders/%d", c.self, row.ID)); e == nil {
				_ = json.Unmarshal(raw, &row)
			}
		}
		lines := make([]Line, 0, len(row.Items))
		for _, it := range row.Items {
			sku := it.InvSkuCode
			if sku == "" {
				sku = it.SkuCode
			}
			if sku == "" || it.Qty <= 0 {
				continue
			}
			lines = append(lines, Line{SkuCode: sku, Name: it.ProductName, Qty: it.Qty})
		}
		if len(lines) == 0 {
			return nil, nil
		}
		return &Fulfillment{
			Source: "self", OrderNo: row.SoNo, SourceNo: row.SoNo, Buyer: row.Buyer, Address: row.Address,
			Cancelled: isClosed(row.Status), Lines: lines,
		}, nil
	}
	return nil, nil
}

func (c *Client) matchShipmentOrderNo(ctx context.Context, bearer, code string) (string, error) {
	if c.ship == "" {
		return "", nil
	}
	q := url.Values{}
	q.Set("keyword", code)
	q.Set("mail_no", code)
	q.Set("pageSize", "10")
	rows, err := c.getList(ctx, bearer, c.ship+"/api/v1/admin/shipments?"+q.Encode())
	if err != nil || len(rows) == 0 {
		return "", err
	}
	type row struct {
		MailNo  string `json:"mailNo"`
		OrderNo string `json:"orderNo"`
		Status  string `json:"status"`
	}
	for _, raw := range rows {
		var item row
		if json.Unmarshal(raw, &item) != nil {
			continue
		}
		if strings.EqualFold(item.MailNo, code) || strings.EqualFold(item.OrderNo, code) {
			if isClosed(item.Status) {
				continue
			}
			return item.OrderNo, nil
		}
	}
	return "", nil
}

func (c *Client) getList(ctx context.Context, bearer, rawURL string) ([]json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if bearer != "" {
		req.Header.Set("Authorization", bearer)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	var wrapped struct {
		Code int `json:"code"`
		Data struct {
			List []json.RawMessage `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Data.List, nil
}

func (c *Client) getData(ctx context.Context, bearer, rawURL string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if bearer != "" {
		req.Header.Set("Authorization", bearer)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	var wrapped struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Data, nil
}

func isClosed(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "cancelled", "canceled", "closed", "refunded", "shipped", "completed":
		return true
	default:
		return false
	}
}
