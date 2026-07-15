package model

import "time"

// ── 仓配分类 ──

type InvCategory struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TenantID  uint64    `gorm:"index;not null" json:"tenantId"`
	Code      string    `gorm:"size:64;not null" json:"code"`
	Name      string    `gorm:"size:128;not null" json:"name"` // 商品类别
	AliasCn   string    `gorm:"size:128" json:"aliasCn"`       // 中文品名
	AliasEn   string    `gorm:"size:128" json:"aliasEn"`       // 英文品名
	ParentID  uint64    `gorm:"default:0" json:"parentId"`
	Sort      int       `gorm:"default:0" json:"sort"`
	Status    int8      `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (InvCategory) TableName() string { return "inv_categories" }

// ── 父SKU / 主SKU ──

type InvProduct struct {
	ID                  uint64     `gorm:"primaryKey" json:"id"`
	TenantID            uint64     `gorm:"index;not null" json:"tenantId"`
	ParentSku           string     `gorm:"size:64;not null" json:"parentSku"`
	Name                string     `gorm:"size:256;not null" json:"name"`
	CategoryID          uint64     `gorm:"index;default:0" json:"categoryId"`
	PackSpecID          uint64     `gorm:"index;default:0" json:"packSpecId"` // 外包装规格
	DevelopedAt         *time.Time `json:"developedAt"`
	DefaultWarehouseID  uint64     `gorm:"default:0" json:"defaultWarehouseId"`
	ScoreFactor         float64    `gorm:"type:numeric(10,4);default:1" json:"scoreFactor"`
	Remark              string     `gorm:"size:1024" json:"remark"`
	Pic                 string     `gorm:"size:512" json:"pic"`
	AlbumPics           string     `gorm:"type:text" json:"albumPics"`
	Status              int8       `gorm:"default:1" json:"status"`
	PimSpuID            *uint64    `json:"pimSpuId"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	Skus                []InvSku   `gorm:"foreignKey:ParentID" json:"skus,omitempty"`
}

func (InvProduct) TableName() string { return "inv_products" }

// ── 库存SKU ──

const (
	ProductTypeNormal    = "normal"
	ProductTypeCombo     = "combo"
	ProductTypeAssembly  = "assembly"
)

type InvSku struct {
	ID                uint64    `gorm:"primaryKey" json:"id"`
	TenantID          uint64    `gorm:"index;not null" json:"tenantId"`
	ParentID          uint64    `gorm:"index;not null" json:"parentId"`
	SkuCode           string    `gorm:"size:64;not null" json:"skuCode"`
	Pic               string    `gorm:"size:512" json:"pic"`
	Status            string    `gorm:"size:32;default:active" json:"status"` // active/inactive/clearance
	ProductType       string    `gorm:"size:32;default:normal" json:"productType"`
	PickName          string    `gorm:"size:256" json:"pickName"`
	Style1            string    `gorm:"size:128" json:"style1"`
	Style2            string    `gorm:"size:128" json:"style2"`
	Style3            string    `gorm:"size:128" json:"style3"`
	WeightG           float64   `gorm:"type:numeric(12,3);default:0" json:"weightG"`
	LastPurchasePrice float64   `gorm:"type:numeric(14,4);default:0" json:"lastPurchasePrice"`
	MinPurchasePrice  float64   `gorm:"type:numeric(14,4);default:0" json:"minPurchasePrice"`
	RetailPrice       float64   `gorm:"type:numeric(14,4);default:0" json:"retailPrice"`
	Description       string    `gorm:"type:text" json:"description"`
	UPC               string    `gorm:"size:64" json:"upc"`
	ASIN              string    `gorm:"size:64" json:"asin"`
	SupplierItemNo    string    `gorm:"size:128" json:"supplierItemNo"`
	PimSkuID          *uint64   `json:"pimSkuId"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (InvSku) TableName() string { return "inv_skus" }

// ── 包装规格（对齐普源 goodspack）──

type InvPackSpec struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TenantID  uint64    `gorm:"index;not null" json:"tenantId"`
	Name      string    `gorm:"size:128;not null" json:"name"` // 包装规格名称
	Cost      float64   `gorm:"type:numeric(14,4);default:0" json:"cost"` // 成本价
	WeightG   float64   `gorm:"type:numeric(12,3);default:0" json:"weightG"`
	Remark    string    `gorm:"size:512" json:"remark"`
	Status    int8      `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (InvPackSpec) TableName() string { return "inv_pack_specs" }

// InvPackSpecSku 包装规格绑定库存SKU（数量范围）
type InvPackSpecSku struct {
	ID         uint64  `gorm:"primaryKey" json:"id"`
	TenantID   uint64  `gorm:"index;not null" json:"tenantId"`
	PackSpecID uint64  `gorm:"index;not null" json:"packSpecId"`
	InvSkuID   uint64  `gorm:"index;not null" json:"invSkuId"`
	QtyMin     float64 `gorm:"type:numeric(14,4);default:0" json:"qtyMin"`
	QtyMax     float64 `gorm:"type:numeric(14,4);default:0" json:"qtyMax"` // 0=不限
	Remark     string  `gorm:"size:256" json:"remark"`
}

func (InvPackSpecSku) TableName() string { return "inv_pack_spec_skus" }

// ── BOM ──

type InvBomHeader struct {
	ID          uint64        `gorm:"primaryKey" json:"id"`
	TenantID    uint64        `gorm:"index;not null" json:"tenantId"`
	ParentSkuID uint64        `gorm:"index;not null" json:"parentSkuId"` // 组合/组装品库存SKU
	BomType     string        `gorm:"size:32;not null" json:"bomType"`   // combo / assembly
	Remark      string        `gorm:"size:512" json:"remark"`
	Status      int8          `gorm:"default:1" json:"status"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	Items       []InvBomItem  `gorm:"foreignKey:BomID" json:"items,omitempty"`
}

func (InvBomHeader) TableName() string { return "inv_bom_headers" }

type InvBomItem struct {
	ID         uint64  `gorm:"primaryKey" json:"id"`
	TenantID   uint64  `gorm:"index;not null" json:"tenantId"`
	BomID      uint64  `gorm:"index;not null" json:"bomId"`
	ChildSkuID uint64  `gorm:"index;not null" json:"childSkuId"`
	Qty        float64 `gorm:"type:numeric(14,4);not null" json:"qty"`
	Remark     string  `gorm:"size:256" json:"remark"`
}

func (InvBomItem) TableName() string { return "inv_bom_items" }

// ── 仓库 / 库位 ──

const (
	WarehouseTypeCentral = "central"
	WarehouseTypeReturn  = "return"
	WarehouseTypeTransit = "transit"
	DefaultLocationCode  = "DEFAULT"
)

type Warehouse struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TenantID  uint64    `gorm:"index;not null" json:"tenantId"`
	Code      string    `gorm:"size:64;not null" json:"code"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Type      string    `gorm:"size:32;default:central" json:"type"`
	Address   string    `gorm:"size:512" json:"address"`
	Contact   string    `gorm:"size:128" json:"contact"`
	Phone     string    `gorm:"size:64" json:"phone"`
	Status    int8      `gorm:"default:1" json:"status"`
	IsDefault int8      `gorm:"default:0" json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Warehouse) TableName() string { return "warehouses" }

type WarehouseLocation struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TenantID    uint64    `gorm:"index;not null" json:"tenantId"`
	WarehouseID uint64    `gorm:"index;not null" json:"warehouseId"`
	Code        string    `gorm:"size:64;not null" json:"code"`
	Zone        string    `gorm:"size:64" json:"zone"`
	Aisle       string    `gorm:"size:64" json:"aisle"`
	Shelf       string    `gorm:"size:64" json:"shelf"`
	Bin         string    `gorm:"size:64" json:"bin"`
	Status      int8      `gorm:"default:1" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (WarehouseLocation) TableName() string { return "warehouse_locations" }

// ── 库存结存 / 流水 ──

type InvBalance struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TenantID    uint64    `gorm:"index;not null" json:"tenantId"`
	WarehouseID uint64    `gorm:"index;not null" json:"warehouseId"`
	LocationID  uint64    `gorm:"index;not null" json:"locationId"`
	InvSkuID    uint64    `gorm:"index;not null" json:"invSkuId"`
	OnHand      float64   `gorm:"type:numeric(14,4);default:0" json:"onHand"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (InvBalance) TableName() string { return "inv_balances" }

const (
	MoveOtherIn        = "other_in"
	MoveOtherOut       = "other_out"
	MoveTransferIn     = "transfer_in"
	MoveTransferOut    = "transfer_out"
	MoveStocktakeGain  = "stocktake_gain"
	MoveStocktakeLoss  = "stocktake_loss"
	MovePurchaseIn     = "purchase_in"
	MoveSaleOut        = "sale_out"
)

type StockMovement struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	TenantID     uint64    `gorm:"index;not null" json:"tenantId"`
	WarehouseID  uint64    `gorm:"index;not null" json:"warehouseId"`
	LocationID   uint64    `gorm:"index;not null" json:"locationId"`
	InvSkuID     uint64    `gorm:"index;not null" json:"invSkuId"`
	MoveType     string    `gorm:"size:32;not null" json:"moveType"`
	Qty          float64   `gorm:"type:numeric(14,4);not null" json:"qty"` // 正=入，负=出
	BalanceAfter float64   `gorm:"type:numeric(14,4);not null" json:"balanceAfter"`
	DocType      string    `gorm:"size:32" json:"docType"`
	DocNo        string    `gorm:"size:64;index" json:"docNo"`
	DocID        uint64    `gorm:"default:0" json:"docId"`
	RefDocType   string    `gorm:"size:32" json:"refDocType"`
	RefDocID     uint64    `gorm:"default:0" json:"refDocId"`
	Remark       string    `gorm:"size:512" json:"remark"`
	CreatedBy    uint64    `gorm:"default:0" json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (StockMovement) TableName() string { return "stock_movements" }

// ── 其他入库 / 出库 ──

const (
	DocStatusDraft     = "draft"
	DocStatusPosted    = "posted"
	DocStatusCancelled = "cancelled"
	DocStatusCounting  = "counting"
	DocStatusReview    = "review"
	DocStatusInTransit = "in_transit"
	DocStatusReceived  = "received"
)

type OtherInboundOrder struct {
	ID          uint64              `gorm:"primaryKey" json:"id"`
	TenantID    uint64              `gorm:"index;not null" json:"tenantId"`
	DocNo       string              `gorm:"size:64;not null" json:"docNo"`
	WarehouseID uint64              `gorm:"index;not null" json:"warehouseId"`
	LocationID  uint64              `gorm:"default:0" json:"locationId"`
	Reason      string              `gorm:"size:64" json:"reason"` // opening/gift/return/adjust/...
	Status      string              `gorm:"size:32;default:draft" json:"status"`
	Remark      string              `gorm:"size:512" json:"remark"`
	PostedAt    *time.Time          `json:"postedAt"`
	CreatedBy   uint64              `gorm:"default:0" json:"createdBy"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
	Items       []OtherInboundItem  `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

func (OtherInboundOrder) TableName() string { return "other_inbound_orders" }

type OtherInboundItem struct {
	ID       uint64  `gorm:"primaryKey" json:"id"`
	TenantID uint64  `gorm:"index;not null" json:"tenantId"`
	OrderID  uint64  `gorm:"index;not null" json:"orderId"`
	InvSkuID uint64  `gorm:"index;not null" json:"invSkuId"`
	Qty      float64 `gorm:"type:numeric(14,4);not null" json:"qty"`
	Cost     float64 `gorm:"type:numeric(14,4);default:0" json:"cost"`
	Remark   string  `gorm:"size:256" json:"remark"`
}

func (OtherInboundItem) TableName() string { return "other_inbound_items" }

type OtherOutboundOrder struct {
	ID          uint64               `gorm:"primaryKey" json:"id"`
	TenantID    uint64               `gorm:"index;not null" json:"tenantId"`
	DocNo       string               `gorm:"size:64;not null" json:"docNo"`
	WarehouseID uint64               `gorm:"index;not null" json:"warehouseId"`
	LocationID  uint64               `gorm:"default:0" json:"locationId"`
	Reason      string               `gorm:"size:64" json:"reason"` // damage/sample/usage/adjust/...
	Status      string               `gorm:"size:32;default:draft" json:"status"`
	Remark      string               `gorm:"size:512" json:"remark"`
	PostedAt    *time.Time           `json:"postedAt"`
	CreatedBy   uint64               `gorm:"default:0" json:"createdBy"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
	Items       []OtherOutboundItem  `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

func (OtherOutboundOrder) TableName() string { return "other_outbound_orders" }

type OtherOutboundItem struct {
	ID       uint64  `gorm:"primaryKey" json:"id"`
	TenantID uint64  `gorm:"index;not null" json:"tenantId"`
	OrderID  uint64  `gorm:"index;not null" json:"orderId"`
	InvSkuID uint64  `gorm:"index;not null" json:"invSkuId"`
	Qty      float64 `gorm:"type:numeric(14,4);not null" json:"qty"`
	Remark   string  `gorm:"size:256" json:"remark"`
}

func (OtherOutboundItem) TableName() string { return "other_outbound_items" }

// ── 盘点 ──

type StocktakeOrder struct {
	ID          uint64           `gorm:"primaryKey" json:"id"`
	TenantID    uint64           `gorm:"index;not null" json:"tenantId"`
	DocNo       string           `gorm:"size:64;not null" json:"docNo"`
	WarehouseID uint64           `gorm:"index;not null" json:"warehouseId"`
	LocationID  uint64           `gorm:"default:0" json:"locationId"` // 0=全仓
	Status      string           `gorm:"size:32;default:draft" json:"status"`
	Remark      string           `gorm:"size:512" json:"remark"`
	PostedAt    *time.Time       `json:"postedAt"`
	CreatedBy   uint64           `gorm:"default:0" json:"createdBy"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	Items       []StocktakeItem  `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

func (StocktakeOrder) TableName() string { return "stocktake_orders" }

type StocktakeItem struct {
	ID          uint64  `gorm:"primaryKey" json:"id"`
	TenantID    uint64  `gorm:"index;not null" json:"tenantId"`
	OrderID     uint64  `gorm:"index;not null" json:"orderId"`
	LocationID  uint64  `gorm:"index;not null" json:"locationId"`
	InvSkuID    uint64  `gorm:"index;not null" json:"invSkuId"`
	BookQty     float64 `gorm:"type:numeric(14,4);default:0" json:"bookQty"`
	CountQty    float64 `gorm:"type:numeric(14,4);default:0" json:"countQty"`
	DiffQty     float64 `gorm:"type:numeric(14,4);default:0" json:"diffQty"`
	Remark      string  `gorm:"size:256" json:"remark"`
}

func (StocktakeItem) TableName() string { return "stocktake_items" }

// ── 调拨 ──

type TransferOrder struct {
	ID              uint64          `gorm:"primaryKey" json:"id"`
	TenantID        uint64          `gorm:"index;not null" json:"tenantId"`
	DocNo           string          `gorm:"size:64;not null" json:"docNo"`
	FromWarehouseID uint64          `gorm:"index;not null" json:"fromWarehouseId"`
	FromLocationID  uint64          `gorm:"default:0" json:"fromLocationId"`
	ToWarehouseID   uint64          `gorm:"index;not null" json:"toWarehouseId"`
	ToLocationID    uint64          `gorm:"default:0" json:"toLocationId"`
	Status          string          `gorm:"size:32;default:draft" json:"status"`
	Remark          string          `gorm:"size:512" json:"remark"`
	ShippedAt       *time.Time      `json:"shippedAt"`
	ReceivedAt      *time.Time      `json:"receivedAt"`
	CreatedBy       uint64          `gorm:"default:0" json:"createdBy"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
	Items           []TransferItem  `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

func (TransferOrder) TableName() string { return "transfer_orders" }

type TransferItem struct {
	ID       uint64  `gorm:"primaryKey" json:"id"`
	TenantID uint64  `gorm:"index;not null" json:"tenantId"`
	OrderID  uint64  `gorm:"index;not null" json:"orderId"`
	InvSkuID uint64  `gorm:"index;not null" json:"invSkuId"`
	Qty      float64 `gorm:"type:numeric(14,4);not null" json:"qty"`
	Remark   string  `gorm:"size:256" json:"remark"`
}

func (TransferItem) TableName() string { return "transfer_items" }

// ── PIM 映射（M4）──

type PimSkuMapping struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	TenantID   uint64    `gorm:"index;not null" json:"tenantId"`
	InvSkuID   uint64    `gorm:"index;not null" json:"invSkuId"`
	PimSkuID   uint64    `gorm:"index;not null" json:"pimSkuId"`
	PimSkuCode string    `gorm:"size:64" json:"pimSkuCode"`
	Remark     string    `gorm:"size:256" json:"remark"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (PimSkuMapping) TableName() string { return "pim_sku_mappings" }
