package dto

import "time"

type InvCategoryDTO struct {
	Code     string `json:"code"` // 留空则创建时自动生成
	Name     string `json:"name" binding:"required"`
	AliasCn  string `json:"aliasCn"`
	AliasEn  string `json:"aliasEn"`
	ParentID uint64 `json:"parentId"`
	Sort     int    `json:"sort"`
	Status   int8   `json:"status"`
}

type InvProductDTO struct {
	ParentSku          string  `json:"parentSku" binding:"required"`
	Name               string  `json:"name" binding:"required"`
	CategoryID         uint64  `json:"categoryId"`
	PackSpecID         uint64  `json:"packSpecId"`
	DevelopedAt        *string `json:"developedAt"`
	DefaultWarehouseID uint64  `json:"defaultWarehouseId"`
	ScoreFactor        float64 `json:"scoreFactor"`
	Remark             string  `json:"remark"`
	Pic                string  `json:"pic"`
	AlbumPics          string  `json:"albumPics"`
	Status             int8    `json:"status"`
	PimSpuID           *uint64 `json:"pimSpuId"`

	Features            string  `json:"features"`
	AliasCn             string  `json:"aliasCn"`
	AliasEn             string  `json:"aliasEn"`
	DeclareWeightG      float64 `json:"declareWeightG"`
	DeclaredValue       float64 `json:"declaredValue"`
	OriginCountryCode   string  `json:"originCountryCode"`
	HSCode              string  `json:"hsCode"`
	ExportDeclaredValue float64 `json:"exportDeclaredValue"`

	PurchaseChannel  string  `json:"purchaseChannel"`
	Purchaser        string  `json:"purchaser"`
	MinPurchasePrice float64 `json:"minPurchasePrice"`
	StockMinAmount   float64 `json:"stockMinAmount"`

	PackFee        float64 `json:"packFee"`
	PackageCount   float64 `json:"packageCount"`
	OutLong        float64 `json:"outLong"`
	OutWide        float64 `json:"outWide"`
	OutHigh        float64 `json:"outHigh"`
	OutGrossWeight float64 `json:"outGrossWeight"`
	OutNetWeight   float64 `json:"outNetWeight"`
	InLong         float64 `json:"inLong"`
	InWide         float64 `json:"inWide"`
	InHigh         float64 `json:"inHigh"`
	InGrossWeight  float64 `json:"inGrossWeight"`
	InNetWeight    float64 `json:"inNetWeight"`
	PackMsg        string  `json:"packMsg"`

	ShopTitle    string  `json:"shopTitle"`
	Brand        string  `json:"brand"`
	SpecClass    string  `json:"specClass"`
	Model        string  `json:"model"`
	Material     string  `json:"material"`
	Style        string  `json:"style"`
	Season       string  `json:"season"`
	Unit         string  `json:"unit"`
	RetailPrice  float64 `json:"retailPrice"`
	BatchPrice   float64 `json:"batchPrice"`
	MaxSalePrice float64 `json:"maxSalePrice"`
	MinSalePrice float64 `json:"minSalePrice"`
	MarketPrice  float64 `json:"marketPrice"`
}

// ProductWithSkusDTO 对齐普源「新增普通商品」：一次提交父SKU + 库存SKU明细 + 多供应商 + 多语言描述
type ProductWithSkusDTO struct {
	InvProductDTO
	DefaultProductType string                      `json:"defaultProductType"` // normal/combo/assembly
	Skus               []ProductSkuItemDTO         `json:"skus" binding:"required,min=1"`
	Suppliers          []ProductSupplierItemDTO    `json:"suppliers"`
	Descriptions       []ProductDescriptionItemDTO `json:"descriptions"`
}

// ProductDescriptionItemDTO 多语言商品描述行（空行由服务端过滤，languageCode 非必填以便前端可提交空数组）
type ProductDescriptionItemDTO struct {
	ID           uint64 `json:"id"`
	LanguageCode string `json:"languageCode" binding:"omitempty,max=16"`
	LanguageName string `json:"languageName" binding:"omitempty,max=64"`
	Title        string `json:"title" binding:"omitempty,max=512"`
	Description  string `json:"description" binding:"omitempty"`
	Sort         int    `json:"sort"`
}

// ProductSupplierItemDTO 商品多供应商行（供应商选自 SupplyCore VMS）
type ProductSupplierItemDTO struct {
	ID           uint64  `json:"id"`
	SupplierID   uint64  `json:"supplierId" binding:"required"`
	SupplierCode string  `json:"supplierCode"`
	SupplierName string  `json:"supplierName"`
	PurchaseURL  string  `json:"purchaseUrl"`
	Price        float64 `json:"price"`
	Remark       string  `json:"remark"`
	ContactName  string  `json:"contactName"`
	Phone        string  `json:"phone"`
	IsDefault    int8    `json:"isDefault"`
	Sort         int     `json:"sort"`
}

type ProductSkuItemDTO struct {
	ID                uint64  `json:"id"`
	SkuCode           string  `json:"skuCode" binding:"required"`
	Pic               string  `json:"pic"`
	Status            string  `json:"status"`
	ProductType       string  `json:"productType"`
	GoodsKind         string  `json:"goodsKind"`
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
}

type InvPackSpecDTO struct {
	Name    string  `json:"name" binding:"required"`
	Cost    float64 `json:"cost"`
	WeightG float64 `json:"weightG"`
	Remark  string  `json:"remark"`
	Status  int8    `json:"status"`
}

type InvPackSpecSkuDTO struct {
	PackSpecID uint64  `json:"packSpecId" binding:"required"`
	InvSkuID   uint64  `json:"invSkuId" binding:"required"`
	QtyMin     float64 `json:"qtyMin"`
	QtyMax     float64 `json:"qtyMax"`
	Remark     string  `json:"remark"`
}

type PackSpecSkuRow struct {
	ID         uint64  `json:"id"`
	PackSpecID uint64  `json:"packSpecId"`
	InvSkuID   uint64  `json:"invSkuId"`
	SkuCode    string  `json:"skuCode"`
	PickName   string  `json:"pickName"`
	QtyMin     float64 `json:"qtyMin"`
	QtyMax     float64 `json:"qtyMax"`
	Remark     string  `json:"remark"`
	NumRange   string  `json:"numRange"`
}

type InvSkuDTO struct {
	ParentID          uint64  `json:"parentId" binding:"required"`
	SkuCode           string  `json:"skuCode" binding:"required"`
	Pic               string  `json:"pic"`
	Status            string  `json:"status"`
	ProductType       string  `json:"productType"`
	GoodsKind         string  `json:"goodsKind"`
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
	GoodsKind         string    `json:"goodsKind"`
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
	Code               string `json:"code" binding:"required"`
	Name               string `json:"name" binding:"required"`
	Type               string `json:"type"`
	Address            string `json:"address"`
	Contact            string `json:"contact"`
	Phone              string `json:"phone"`
	Country            string `json:"country"`
	Remark             string `json:"remark"`
	Status             int8   `json:"status"`
	IsDefault          int8   `json:"isDefault"`
	AllowCalcFee       int8   `json:"allowCalcFee"`
	AllowNegativeStock int8   `json:"allowNegativeStock"`
}

type GoodsFeeSettingsDTO struct {
	StoreFee      float64                  `json:"storeFee"`
	FixedStoreFee float64                  `json:"fixedStoreFee"`
	PackFee       float64                  `json:"packFee"`
	ScoreRules    []ScoreWeightRuleDTO     `json:"scoreRules"`
	QtyCoeffs     []OrderQtyCoeffDTO       `json:"qtyCoeffs"`
}

type ScoreWeightRuleDTO struct {
	ID          uint64  `json:"id"`
	WeightMinG  float64 `json:"weightMinG"`
	WeightMaxG  float64 `json:"weightMaxG"`
	ScoreFactor float64 `json:"scoreFactor"`
	Sort        int     `json:"sort"`
}

type OrderQtyCoeffDTO struct {
	ID     uint64  `json:"id"`
	QtyMin float64 `json:"qtyMin"`
	QtyMax float64 `json:"qtyMax"`
	Coeff  float64 `json:"coeff"`
	Sort   int     `json:"sort"`
}

type UpdateSkuWeightDTO struct {
	SkuCode string  `json:"skuCode" binding:"required"`
	WeightG float64 `json:"weightG" binding:"required"`
}

type ProfitTrialDTO struct {
	ID              uint64  `json:"id"`
	ParentSKU       string  `json:"parentSku"`
	SKU             string  `json:"sku" binding:"required"`
	ShopSKU         string  `json:"shopSku"`
	ShopName        string  `json:"shopName"`
	SKUName         string  `json:"skuName"`
	RetailPrice     float64 `json:"retailPrice"`
	PriceUS         float64 `json:"priceUs"`
	Price           float64 `json:"price"`
	CostPrice       float64 `json:"costPrice"`
	PlatformFreight float64 `json:"platformFreight"`
	HeadFreight     float64 `json:"headFreight"`
	Freight         float64 `json:"freight"`
	PackageFee      float64 `json:"packageFee"`
	Tariff          float64 `json:"tariff"`
	Profit          float64 `json:"profit"`
	ProfitMargin    float64 `json:"profitMargin"`
	ASIN            string  `json:"asin"`
	Remark          string  `json:"remark"`
}

type ProfitCalcMode string

const (
	ProfitCalcByCost   ProfitCalcMode = "by_cost"   // 按成本算利润
	ProfitCalcByMargin ProfitCalcMode = "by_margin" // 按利润率反推售价
)

type ProfitCalcRequest struct {
	IDs  []uint64       `json:"ids"`
	Mode ProfitCalcMode `json:"mode"`
	// Mode=by_margin 时用目标利润率（%）覆盖行上利润率后再反推售价
	TargetMargin *float64 `json:"targetMargin"`
}

type LocationDTO struct {
	WarehouseID  uint64 `json:"warehouseId" binding:"required"`
	Code         string `json:"code" binding:"required"`
	Zone         string `json:"zone"`
	Aisle        string `json:"aisle"`
	Shelf        string `json:"shelf"`
	Bin          string `json:"bin"`
	PickOrder    int    `json:"pickOrder"`
	PickPosition string `json:"pickPosition"`
	Remark       string `json:"remark"`
	Status       int8   `json:"status"`
}

type LocationSkuBindDTO struct {
	InvSkuID uint64 `json:"invSkuId" binding:"required"`
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
	CheckerName string `json:"checkerName"` // 盘点人
	Remark      string `json:"remark"`
	// FillAllBalances=true 时按仓库结存自动带出明细（兼容旧行为）；默认 false，对齐普源手工添加商品
	FillAllBalances bool `json:"fillAllBalances"`
}

type StocktakeUpdateDTO struct {
	CheckerName string `json:"checkerName"`
	Remark      string `json:"remark"`
}

type StocktakeAddItemsDTO struct {
	Items []StocktakeAddItemDTO `json:"items" binding:"required,min=1"`
}

type StocktakeAddItemDTO struct {
	InvSkuID   uint64   `json:"invSkuId" binding:"required"`
	LocationID uint64   `json:"locationId"`
	CountQty   *float64 `json:"countQty"` // nil=默认等于账存
	Remark     string   `json:"remark"`
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
	HideZero    bool
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
	ID               uint64    `json:"id"`
	WarehouseID      uint64    `json:"warehouseId"`
	WarehouseName    string    `json:"warehouseName"`
	LocationID       uint64    `json:"locationId"`
	LocationCode     string    `json:"locationCode"`
	InvSkuID         uint64    `json:"invSkuId"`
	SkuCode          string    `json:"skuCode"`
	Pic              string    `json:"pic"`
	PickName         string    `json:"pickName"`
	ProductName      string    `json:"productName"`
	CategoryName     string    `json:"categoryName"`
	SkuStatus        string    `json:"skuStatus"`
	OnHand           float64   `json:"onHand"`
	ReservedQty      float64   `json:"reservedQty"` // 一期无占用，固定 0
	AvailableQty     float64   `json:"availableQty"`
	StockAmount      float64   `json:"stockAmount"`
	UnitCost         float64   `json:"unitCost"`         // 库存单价（上次采购价）
	MinPurchasePrice float64   `json:"minPurchasePrice"`
	LastCost         float64   `json:"lastCost"` // 上次采购价
	WeightG          float64   `json:"weightG"`
	Brand            string    `json:"brand"`
	SpecClass        string    `json:"specClass"`
	Model            string    `json:"model"`
	Material         string    `json:"material"`
	Style1           string    `json:"style1"`
	Style2           string    `json:"style2"`
	Style3           string    `json:"style3"`
	RetailPrice      float64   `json:"retailPrice"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type SummaryRow struct {
	WarehouseID   uint64  `json:"warehouseId"`
	WarehouseName string  `json:"warehouseName"`
	InvSkuID      uint64  `json:"invSkuId"`
	SkuCode       string  `json:"skuCode"`
	PickName      string  `json:"pickName"`
	ProductName   string  `json:"productName"`
	Style1        string  `json:"style1"`
	Style2        string  `json:"style2"`
	Style3        string  `json:"style3"`
	Purchaser     string  `json:"purchaser"`
	CostPrice     float64 `json:"costPrice"`
	Opening       float64 `json:"opening"`
	OpeningAmount float64 `json:"openingAmount"`
	Inbound       float64 `json:"inbound"`
	InboundAmount float64 `json:"inboundAmount"`
	Outbound      float64 `json:"outbound"`
	OutboundAmount float64 `json:"outboundAmount"`
	Closing       float64 `json:"closing"`
	AvgUnitCost   float64 `json:"avgUnitCost"`
	ClosingAmount float64 `json:"closingAmount"`
}

type MovementRow struct {
	ID            uint64    `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	WarehouseID   uint64    `json:"warehouseId"`
	WarehouseName string    `json:"warehouseName"`
	LocationCode  string    `json:"locationCode"`
	InvSkuID      uint64    `json:"invSkuId"`
	SkuCode       string    `json:"skuCode"`
	PickName      string    `json:"pickName"`
	ProductName   string    `json:"productName"`
	MoveType      string    `json:"moveType"`
	Qty           float64   `json:"qty"`
	InboundQty    float64   `json:"inboundQty"`
	OutboundQty   float64   `json:"outboundQty"`
	BalanceAfter  float64   `json:"balanceAfter"`
	UnitCost      float64   `json:"unitCost"`
	Amount        float64   `json:"amount"`
	DocType       string    `json:"docType"`
	DocNo         string    `json:"docNo"`
	Remark        string    `json:"remark"`
}

type SlowMovingRow struct {
	WarehouseID     uint64     `json:"warehouseId"`
	WarehouseName   string     `json:"warehouseName"`
	InvSkuID        uint64     `json:"invSkuId"`
	SkuCode         string     `json:"skuCode"`
	PickName        string     `json:"pickName"`
	ProductName     string     `json:"productName"`
	OnHand          float64    `json:"onHand"`
	AvailableQty    float64    `json:"availableQty"`
	UnitCost        float64    `json:"unitCost"`
	StockAmount     float64    `json:"stockAmount"`
	LastInboundAt   *time.Time `json:"lastInboundAt"`
	LastInboundQty  float64    `json:"lastInboundQty"`
	LastMoveAt      *time.Time `json:"lastMoveAt"`
	IdleDays        int        `json:"idleDays"`
	CreatedAt       *time.Time `json:"createdAt"` // 商品创建时间
}
