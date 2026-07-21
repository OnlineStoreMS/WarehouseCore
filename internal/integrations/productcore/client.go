package productcore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

type ProductItem struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	ProductSn     string `json:"productSn"`
	MaterialCode  string `json:"materialCode"`
	Pic           string `json:"pic"`
	SkuCount      int    `json:"skuCount"`
	PublishStatus int8   `json:"publishStatus"`
	BrandName     string `json:"brandName"`
	CategoryName  string `json:"categoryName"`
}

type SkuItem struct {
	ID      uint64            `json:"id"`
	SkuCode string            `json:"skuCode"`
	Specs   map[string]string `json:"specs"`
	Price   float64           `json:"price"`
	Stock   int               `json:"stock"`
	Pic     string            `json:"pic"`
}

type ProductSkus struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	MaterialCode string    `json:"materialCode"`
	SkuCount     int       `json:"skuCount"`
	Skus         []SkuItem `json:"skus"`
}

type pagePayload struct {
	List     []ProductItem `json:"list"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

type apiBody struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) get(ctx context.Context, bearerToken, path string, q url.Values) (json.RawMessage, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("productcore 未配置")
	}
	reqURL := c.baseURL + path
	if len(q) > 0 {
		reqURL += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", bearerToken)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("productcore request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("productcore http %d", resp.StatusCode)
	}
	var wrapped apiBody
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, err
	}
	if wrapped.Code != 200 {
		msg := wrapped.Message
		if msg == "" {
			msg = "productcore error"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return wrapped.Data, nil
}

func (c *Client) ListProducts(ctx context.Context, bearerToken, keyword string, page, pageSize int) ([]ProductItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := url.Values{}
	if keyword != "" {
		q.Set("keyword", keyword)
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(pageSize))
	raw, err := c.get(ctx, bearerToken, "/api/v1/admin/products", q)
	if err != nil {
		return nil, 0, err
	}
	var pageData pagePayload
	if err := json.Unmarshal(raw, &pageData); err != nil {
		return nil, 0, err
	}
	if pageData.List == nil {
		pageData.List = []ProductItem{}
	}
	return pageData.List, pageData.Total, nil
}

func (c *Client) GetProductSkus(ctx context.Context, bearerToken string, productID uint64) (*ProductSkus, error) {
	if productID == 0 {
		return nil, fmt.Errorf("product id required")
	}
	path := fmt.Sprintf("/api/v1/admin/products/%d/skus", productID)
	raw, err := c.get(ctx, bearerToken, path, nil)
	if err != nil {
		return nil, err
	}
	var out ProductSkus
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.Skus == nil {
		out.Skus = []SkuItem{}
	}
	return &out, nil
}
