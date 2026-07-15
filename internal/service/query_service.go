package service

import (
	"time"

	"warehousecore/internal/dto"
	"warehousecore/internal/repo"

	"gorm.io/gorm"
)

type QueryService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewQueryService(repos *repo.Repos) *QueryService {
	return &QueryService{repos: repos}
}

func (s *QueryService) ForTenant(tenantID uint64) *QueryService {
	return &QueryService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *QueryService) QueryBalances(q dto.StockQuery) ([]dto.BalanceRow, int64, error) {
	db := s.repos.DB.Table("inv_balances AS b").
		Select(`b.id, b.warehouse_id, w.name AS warehouse_name, b.location_id, l.code AS location_code,
			b.inv_sku_id, s.sku_code, COALESCE(s.pic,'') AS pic, s.pick_name, p.name AS product_name,
			COALESCE(c.name,'') AS category_name, s.status AS sku_status, b.on_hand,
			s.retail_price, s.last_purchase_price AS last_cost, s.last_purchase_price AS unit_cost,
			s.min_purchase_price, s.weight_g, COALESCE(p.brand,'') AS brand,
			COALESCE(p.spec_class,'') AS spec_class, COALESCE(p.model,'') AS model,
			COALESCE(p.material,'') AS material, s.style1, s.style2, s.style3, b.updated_at`).
		Joins("JOIN warehouses w ON w.id = b.warehouse_id").
		Joins("JOIN warehouse_locations l ON l.id = b.location_id").
		Joins("JOIN inv_skus s ON s.id = b.inv_sku_id").
		Joins("JOIN inv_products p ON p.id = s.parent_id").
		Joins("LEFT JOIN inv_categories c ON c.id = p.category_id").
		Where("b.tenant_id = ?", s.tenantID)
	if q.WarehouseID > 0 {
		db = db.Where("b.warehouse_id = ?", q.WarehouseID)
	}
	if q.LocationID > 0 {
		db = db.Where("b.location_id = ?", q.LocationID)
	}
	if q.InvSkuID > 0 {
		db = db.Where("b.inv_sku_id = ?", q.InvSkuID)
	}
	if q.SkuCode != "" {
		db = db.Where("s.sku_code ILIKE ?", "%"+q.SkuCode+"%")
	}
	if q.CategoryID > 0 {
		db = db.Where("p.category_id = ?", q.CategoryID)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		db = db.Where("s.sku_code ILIKE ? OR s.pick_name ILIKE ? OR p.name ILIKE ?", like, like, like)
	}
	if q.HideZero {
		db = db.Where("b.on_hand <> 0")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := q.Page, q.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var list []dto.BalanceRow
	if err := db.Order("b.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&list).Error; err != nil {
		return nil, 0, err
	}
	for i := range list {
		list[i].ReservedQty = 0
		list[i].AvailableQty = list[i].OnHand
		if list[i].UnitCost == 0 {
			list[i].UnitCost = list[i].LastCost
		}
		list[i].StockAmount = list[i].OnHand * list[i].UnitCost
	}
	return list, total, nil
}

func (s *QueryService) QuerySummary(warehouseID uint64, skuCode string, from, to *time.Time, page, pageSize int) ([]dto.SummaryRow, int64, error) {
	mq := s.repos.DB.Table("stock_movements AS m").
		Select(`m.warehouse_id, m.inv_sku_id,
			COALESCE(SUM(CASE WHEN m.qty > 0 THEN m.qty ELSE 0 END),0) AS inbound,
			COALESCE(SUM(CASE WHEN m.qty < 0 THEN -m.qty ELSE 0 END),0) AS outbound,
			COALESCE(SUM(CASE WHEN m.qty > 0 THEN m.qty * COALESCE(s.last_purchase_price,0) ELSE 0 END),0) AS inbound_amount,
			COALESCE(SUM(CASE WHEN m.qty < 0 THEN (-m.qty) * COALESCE(s.last_purchase_price,0) ELSE 0 END),0) AS outbound_amount`).
		Joins("JOIN inv_skus s ON s.id = m.inv_sku_id").
		Where("m.tenant_id = ?", s.tenantID)
	if warehouseID > 0 {
		mq = mq.Where("m.warehouse_id = ?", warehouseID)
	}
	if from != nil {
		mq = mq.Where("m.created_at >= ?", *from)
	}
	if to != nil {
		mq = mq.Where("m.created_at <= ?", *to)
	}
	mq = mq.Group("m.warehouse_id, m.inv_sku_id")

	bq := s.repos.DB.Table("inv_balances AS b").
		Select(`b.warehouse_id, w.name AS warehouse_name, b.inv_sku_id, s.sku_code, s.pick_name,
			p.name AS product_name, s.style1, s.style2, s.style3, COALESCE(p.purchaser,'') AS purchaser,
			s.last_purchase_price AS cost_price,
			COALESCE(SUM(b.on_hand),0) AS closing,
			COALESCE(a.inbound,0) AS inbound,
			COALESCE(a.outbound,0) AS outbound,
			COALESCE(a.inbound_amount,0) AS inbound_amount,
			COALESCE(a.outbound_amount,0) AS outbound_amount`).
		Joins("JOIN warehouses w ON w.id = b.warehouse_id").
		Joins("JOIN inv_skus s ON s.id = b.inv_sku_id").
		Joins("JOIN inv_products p ON p.id = s.parent_id").
		Joins("LEFT JOIN (?) AS a ON a.warehouse_id = b.warehouse_id AND a.inv_sku_id = b.inv_sku_id", mq).
		Where("b.tenant_id = ?", s.tenantID).
		Group(`b.warehouse_id, w.name, b.inv_sku_id, s.sku_code, s.pick_name, p.name,
			s.style1, s.style2, s.style3, p.purchaser, s.last_purchase_price,
			a.inbound, a.outbound, a.inbound_amount, a.outbound_amount`)
	if warehouseID > 0 {
		bq = bq.Where("b.warehouse_id = ?", warehouseID)
	}
	if skuCode != "" {
		bq = bq.Where("s.sku_code ILIKE ?", "%"+skuCode+"%")
	}

	var total int64
	if err := s.repos.DB.Table("(?) AS t", bq).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var raw []struct {
		WarehouseID    uint64
		WarehouseName  string
		InvSkuID       uint64
		SkuCode        string
		PickName       string
		ProductName    string
		Style1         string
		Style2         string
		Style3         string
		Purchaser      string
		CostPrice      float64
		Closing        float64
		Inbound        float64
		Outbound       float64
		InboundAmount  float64
		OutboundAmount float64
	}
	if err := bq.Offset((page - 1) * pageSize).Limit(pageSize).Scan(&raw).Error; err != nil {
		return nil, 0, err
	}
	list := make([]dto.SummaryRow, 0, len(raw))
	for _, r := range raw {
		opening := r.Closing - r.Inbound + r.Outbound
		cost := r.CostPrice
		closingAmt := r.Closing * cost
		openingAmt := opening * cost
		avg := cost
		if r.Closing > 0 && closingAmt > 0 {
			avg = closingAmt / r.Closing
		}
		list = append(list, dto.SummaryRow{
			WarehouseID:    r.WarehouseID,
			WarehouseName:  r.WarehouseName,
			InvSkuID:       r.InvSkuID,
			SkuCode:        r.SkuCode,
			PickName:       r.PickName,
			ProductName:    r.ProductName,
			Style1:         r.Style1,
			Style2:         r.Style2,
			Style3:         r.Style3,
			Purchaser:      r.Purchaser,
			CostPrice:      cost,
			Opening:        opening,
			OpeningAmount:  openingAmt,
			Inbound:        r.Inbound,
			InboundAmount:  r.InboundAmount,
			Outbound:       r.Outbound,
			OutboundAmount: r.OutboundAmount,
			Closing:        r.Closing,
			AvgUnitCost:    avg,
			ClosingAmount:  closingAmt,
		})
	}
	return list, total, nil
}

func (s *QueryService) QueryMovements(warehouseID, invSkuID uint64, skuCode, moveType, docNo string, from, to *time.Time, page, pageSize int) ([]dto.MovementRow, int64, error) {
	db := s.repos.DB.Table("stock_movements AS m").
		Select(`m.id, m.created_at, m.warehouse_id, w.name AS warehouse_name, l.code AS location_code,
			m.inv_sku_id, s.sku_code, s.pick_name, p.name AS product_name, m.move_type, m.qty, m.balance_after,
			COALESCE(s.last_purchase_price,0) AS unit_cost, m.doc_type, m.doc_no, m.remark`).
		Joins("JOIN warehouses w ON w.id = m.warehouse_id").
		Joins("JOIN warehouse_locations l ON l.id = m.location_id").
		Joins("JOIN inv_skus s ON s.id = m.inv_sku_id").
		Joins("JOIN inv_products p ON p.id = s.parent_id").
		Where("m.tenant_id = ?", s.tenantID)
	if warehouseID > 0 {
		db = db.Where("m.warehouse_id = ?", warehouseID)
	}
	if invSkuID > 0 {
		db = db.Where("m.inv_sku_id = ?", invSkuID)
	}
	if skuCode != "" {
		db = db.Where("s.sku_code ILIKE ?", "%"+skuCode+"%")
	}
	if moveType != "" {
		db = db.Where("m.move_type = ?", moveType)
	}
	if docNo != "" {
		db = db.Where("m.doc_no ILIKE ?", "%"+docNo+"%")
	}
	if from != nil {
		db = db.Where("m.created_at >= ?", *from)
	}
	if to != nil {
		db = db.Where("m.created_at <= ?", *to)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var list []dto.MovementRow
	if err := db.Order("m.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&list).Error; err != nil {
		return nil, 0, err
	}
	for i := range list {
		if list[i].Qty > 0 {
			list[i].InboundQty = list[i].Qty
		} else if list[i].Qty < 0 {
			list[i].OutboundQty = -list[i].Qty
		}
		q := list[i].Qty
		if q < 0 {
			q = -q
		}
		list[i].Amount = q * list[i].UnitCost
	}
	return list, total, nil
}

func (s *QueryService) QuerySlowMoving(q dto.SlowMovingQuery) ([]dto.SlowMovingRow, int64, error) {
	days := q.Days
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	// 最后一次入库（qty>0）
	lastIn := s.repos.DB.Table("stock_movements AS m").
		Select(`DISTINCT ON (m.warehouse_id, m.inv_sku_id) m.warehouse_id, m.inv_sku_id,
			m.created_at AS last_inbound_at, m.qty AS last_inbound_qty`).
		Where("m.tenant_id = ? AND m.qty > 0", s.tenantID).
		Order("m.warehouse_id, m.inv_sku_id, m.created_at DESC")

	db := s.repos.DB.Table("inv_balances AS b").
		Select(`b.warehouse_id, w.name AS warehouse_name, b.inv_sku_id, s.sku_code, s.pick_name,
			p.name AS product_name, SUM(b.on_hand) AS on_hand, s.last_purchase_price AS unit_cost,
			li.last_inbound_at, COALESCE(li.last_inbound_qty,0) AS last_inbound_qty, p.created_at`).
		Joins("JOIN warehouses w ON w.id = b.warehouse_id").
		Joins("JOIN inv_skus s ON s.id = b.inv_sku_id").
		Joins("JOIN inv_products p ON p.id = s.parent_id").
		Joins("LEFT JOIN (?) AS li ON li.warehouse_id = b.warehouse_id AND li.inv_sku_id = b.inv_sku_id", lastIn).
		Where("b.tenant_id = ? AND b.on_hand > ?", s.tenantID, q.MinOnHand).
		Where("(li.last_inbound_at IS NULL OR li.last_inbound_at < ?)", cutoff).
		Group(`b.warehouse_id, w.name, b.inv_sku_id, s.sku_code, s.pick_name, p.name,
			s.last_purchase_price, li.last_inbound_at, li.last_inbound_qty, p.created_at`)
	if q.WarehouseID > 0 {
		db = db.Where("b.warehouse_id = ?", q.WarehouseID)
	}

	var total int64
	if err := s.repos.DB.Table("(?) AS t", db).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := q.Page, q.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var raw []struct {
		WarehouseID    uint64
		WarehouseName  string
		InvSkuID       uint64
		SkuCode        string
		PickName       string
		ProductName    string
		OnHand         float64
		UnitCost       float64
		LastInboundAt  *time.Time
		LastInboundQty float64
		CreatedAt      *time.Time
	}
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Scan(&raw).Error; err != nil {
		return nil, 0, err
	}
	now := time.Now()
	list := make([]dto.SlowMovingRow, 0, len(raw))
	for _, r := range raw {
		idle := days
		if r.LastInboundAt != nil {
			idle = int(now.Sub(*r.LastInboundAt).Hours() / 24)
		}
		list = append(list, dto.SlowMovingRow{
			WarehouseID:    r.WarehouseID,
			WarehouseName:  r.WarehouseName,
			InvSkuID:       r.InvSkuID,
			SkuCode:        r.SkuCode,
			PickName:       r.PickName,
			ProductName:    r.ProductName,
			OnHand:         r.OnHand,
			AvailableQty:   r.OnHand,
			UnitCost:       r.UnitCost,
			StockAmount:    r.OnHand * r.UnitCost,
			LastInboundAt:  r.LastInboundAt,
			LastInboundQty: r.LastInboundQty,
			LastMoveAt:     r.LastInboundAt,
			IdleDays:       idle,
			CreatedAt:      r.CreatedAt,
		})
	}
	return list, total, nil
}

var _ = gorm.ErrRecordNotFound
