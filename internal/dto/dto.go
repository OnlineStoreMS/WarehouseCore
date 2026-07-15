package dto

import "time"

type InvCategoryDTO struct {
	Code     string `json:"code" binding:"required"`
	Name     string `json:"name" binding:"required"`
	ParentID uint64 `json:"parentId"`
	Sort     int    `json:"sort"`
	Status   int8   `json:"status"`
}

type InvProductDTO struct {
	ParentSku          string  `json:"parentSku" binding:"required"`
	Name               string  `json:"name" binding:"required"`
	CategoryID         uint64  `json:"categoryId"`
	DevelopedAt        *string `json:"developedAt"`
	DefaultWarehouseID uint64  `json:"defaultWarehouseId"`
	ScoreFactor        float64 `json:"scoreFactor"`
	Remark             string  `json:"remark"`
	Pic                string  `json:"pic"`
	AlbumPics          string  `json:"albumPics"`
	Status             int8    `json:"status"`
	PimSpuID           *uint64 `json:"pimSpuId"`
}

type InvSkuDTO struct {
	ParentID          uint64  `json:"parentId" binding:"required"`
	SkuCode           string  `json:"skuCode" binding:"required"`
	Pic               string  `json:"pic"`
	Status            string  `json:"status"`
	ProductType       string  `json:"productType"`
	PickName          string  `json:"pickName"`
	Style1            string  `json:"style1"`
	Style2            string  `json:"style2"`
	Style3            string  `json:"style3"`
	WeightG           float64 `json:"weightG"`
	LastPurchasePrice float64 `json:"lastPurchasePrice"`
	MinPurchasePrice  float64 `json:"minPurchasePrice"`
	RetailPrice       float64 `json:"retailPrice"`
	Description       string  `json:"description"`
	UPC               string  `json:"upc"`
	ASIN              string  `json:"asin"`
	SupplierItemNo    string  `json:"supplierItemNo"`
	PimSkuID          *uint64 `json:"pimSkuId"`
}

// SkuListRow 库存SKU 明细行（含父商品信息，对齐普源「库存SKU明细」）
type SkuListRow struct {
	ID                uint64    `json:"id"`
	TenantID          uint64    `json:"tenantId"`
	ParentID          uint64    `json:"parentId"`
	SkuCode           string    `json:"skuCode"`
	Pic               string    `json:"pic"`
	Status            string    `json:"status"`
	ProductType       string    `json:"productType"`
	PickName          string    `json:"pickName"`
	Style1            string    `json:"style1"`
	Style2            string    `json:"style2"`
	Style3            string    `json:"style3"`
	WeightG           float64   `json:"weightG"`
	LastPurchasePrice float64   `json:"lastPurchasePrice"`
	MinPurchasePrice  float64   `json:"minPurchasePrice"`
	RetailPrice       float64   `json:"retailPrice"`
	Description       string    `json:"description"`
	UPC               string    `json:"upc"`
	ASIN              string    `json:"asin"`
	SupplierItemNo    string    `json:"supplierItemNo"`
	PimSkuID          *uint64   `json:"pimSkuId"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	ParentSkuCode     string    `json:"parentSku"`
	ProductName       string    `json:"productName"`
	CategoryID        uint64    `json:"categoryId"`
}

type BomDTO struct {
	ParentSkuID uint64       `json:"parentSkuId" binding:"required"`
	BomType     string       `json:"bomType" binding:"required"`
	Remark      string       `json:"remark"`
	Status      int8         `json:"status"`
	Items       []BomItemDTO `json:"items"`
}

type BomItemDTO struct {
	ChildSkuID uint64  `json:"childSkuId" binding:"required"`
	Qty        float64 `json:"qty" binding:"required"`
	Remark     string  `json:"remark"`
}

type WarehouseDTO struct {
	Code      string `json:"code" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Type      string `json:"type"`
	Address   string `json:"address"`
	Contact   string `json:"contact"`
	Phone     string `json:"phone"`
	Status    int8   `json:"status"`
	IsDefault int8   `json:"isDefault"`
}

type LocationDTO struct {
	WarehouseID uint64 `json:"warehouseId" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Zone        string `json:"zone"`
	Aisle       string `json:"aisle"`
	Shelf       string `json:"shelf"`
	Bin         string `json:"bin"`
	Status      int8   `json:"status"`
}

type DocItemDTO struct {
	InvSkuID   uint64  `json:"invSkuId" binding:"required"`
	Qty        float64 `json:"qty" binding:"required"`
	Cost       float64 `json:"cost"`
	LocationID uint64  `json:"locationId"`
	Remark     string  `json:"remark"`
}

type OtherInboundDTO struct {
	WarehouseID uint64       `json:"warehouseId" binding:"required"`
	LocationID  uint64       `json:"locationId"`
	Reason      string       `json:"reason"`
	Remark      string       `json:"remark"`
	Items       []DocItemDTO `json:"items" binding:"required,min=1"`
}

type OtherOutboundDTO struct {
	WarehouseID uint64       `json:"warehouseId" binding:"required"`
	LocationID  uint64       `json:"locationId"`
	Reason      string       `json:"reason"`
	Remark      string       `json:"remark"`
	Items       []DocItemDTO `json:"items" binding:"required,min=1"`
}

type StocktakeCreateDTO struct {
	WarehouseID uint64 `json:"warehouseId" binding:"required"`
	LocationID  uint64 `json:"locationId"`
	Remark      string `json:"remark"`
}

type StocktakeCountDTO struct {
	Items []struct {
		ID       uint64  `json:"id" binding:"required"`
		CountQty float64 `json:"countQty"`
		Remark   string  `json:"remark"`
	} `json:"items" binding:"required"`
}

type TransferDTO struct {
	FromWarehouseID uint64       `json:"fromWarehouseId" binding:"required"`
	FromLocationID  uint64       `json:"fromLocationId"`
	ToWarehouseID   uint64       `json:"toWarehouseId" binding:"required"`
	ToLocationID    uint64       `json:"toLocationID"`
	ToLocationId    uint64       `json:"toLocationId"`
	Remark          string       `json:"remark"`
	Items           []DocItemDTO `json:"items" binding:"required,min=1"`
}

func (t *TransferDTO) ResolveToLocationID() uint64 {
	if t.ToLocationId > 0 {
		return t.ToLocationId
	}
	return t.ToLocationID
}

type PimMappingDTO struct {
	InvSkuID   uint64 `json:"invSkuId" binding:"required"`
	PimSkuID   uint64 `json:"pimSkuId" binding:"required"`
	PimSkuCode string `json:"pimSkuCode"`
	Remark     string `json:"remark"`
}

type PurchaseInboundDTO struct {
	WarehouseID uint64       `json:"warehouseId" binding:"required"`
	LocationID  uint64       `json:"locationId"`
	RefDocType  string       `json:"refDocType"` // purchase_order
	RefDocID    uint64       `json:"refDocId"`
	RefDocNo    string       `json:"refDocNo"`
	Remark      string       `json:"remark"`
	Items       []DocItemDTO `json:"items" binding:"required,min=1"`
}

type StoreTransferDTO struct {
	FromWarehouseID uint64       `json:"fromWarehouseId" binding:"required"`
	FromLocationID  uint64       `json:"fromLocationId"`
	StoreID         uint64       `json:"storeId" binding:"required"`
	Remark          string       `json:"remark"`
	Items           []DocItemDTO `json:"items" binding:"required,min=1"`
}

type StockQuery struct {
	WarehouseID uint64
	LocationID  uint64
	InvSkuID    uint64
	SkuCode     string
	CategoryID  uint64
	Keyword     string
	Page        int
	PageSize    int
}

type SlowMovingQuery struct {
	WarehouseID uint64
	Days        int
	MinOnHand   float64
	Page        int
	PageSize    int
}

type BalanceRow struct {
	ID            uint64    `json:"id"`
	WarehouseID   uint64    `json:"warehouseId"`
	WarehouseName string    `json:"warehouseName"`
	LocationID    uint64    `json:"locationId"`
	LocationCode  string    `json:"locationCode"`
	InvSkuID      uint64    `json:"invSkuId"`
	SkuCode       string    `json:"skuCode"`
	PickName      string    `json:"pickName"`
	ProductName   string    `json:"productName"`
	OnHand        float64   `json:"onHand"`
	RetailPrice   float64   `json:"retailPrice"`
	LastCost      float64   `json:"lastCost"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type SummaryRow struct {
	WarehouseID   uint64  `json:"warehouseId"`
	WarehouseName string  `json:"warehouseName"`
	InvSkuID      uint64  `json:"invSkuId"`
	SkuCode       string  `json:"skuCode"`
	ProductName   string  `json:"productName"`
	Opening       float64 `json:"opening"`
	Inbound       float64 `json:"inbound"`
	Outbound      float64 `json:"outbound"`
	Closing       float64 `json:"closing"`
}

type MovementRow struct {
	ID            uint64    `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	WarehouseID   uint64    `json:"warehouseId"`
	WarehouseName string    `json:"warehouseName"`
	LocationCode  string    `json:"locationCode"`
	InvSkuID      uint64    `json:"invSkuId"`
	SkuCode       string    `json:"skuCode"`
	MoveType      string    `json:"moveType"`
	Qty           float64   `json:"qty"`
	BalanceAfter  float64   `json:"balanceAfter"`
	DocType       string    `json:"docType"`
	DocNo         string    `json:"docNo"`
	Remark        string    `json:"remark"`
}

type SlowMovingRow struct {
	WarehouseID   uint64     `json:"warehouseId"`
	WarehouseName string     `json:"warehouseName"`
	InvSkuID      uint64     `json:"invSkuId"`
	SkuCode       string     `json:"skuCode"`
	ProductName   string     `json:"productName"`
	OnHand        float64    `json:"onHand"`
	LastMoveAt    *time.Time `json:"lastMoveAt"`
	IdleDays      int        `json:"idleDays"`
}
