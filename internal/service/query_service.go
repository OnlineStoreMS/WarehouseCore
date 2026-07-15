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
			b.inv_sku_id, s.sku_code, s.pick_name, p.name AS product_name, b.on_hand,
			s.retail_price, s.last_purchase_price AS last_cost, b.updated_at`).
		Joins("JOIN warehouses w ON w.id = b.warehouse_id").
		Joins("JOIN warehouse_locations l ON l.id = b.location_id").
		Joins("JOIN inv_skus s ON s.id = b.inv_sku_id").
		Joins("JOIN inv_products p ON p.id = s.parent_id").
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
	err := db.Order("b.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&list).Error
	return list, total, err
}

func (s *QueryService) QuerySummary(warehouseID uint64, from, to *time.Time, page, pageSize int) ([]dto.SummaryRow, int64, error) {
	// Closing = current on_hand; inbound/outbound from movements in range; opening = closing - in + out
	type agg struct {
		WarehouseID uint64
		InvSkuID    uint64
		Inbound     float64
		Outbound    float64
	}
	mq := s.repos.DB.Table("stock_movements").
		Select(`warehouse_id, inv_sku_id,
			COALESCE(SUM(CASE WHEN qty > 0 THEN qty ELSE 0 END),0) AS inbound,
			COALESCE(SUM(CASE WHEN qty < 0 THEN -qty ELSE 0 END),0) AS outbound`).
		Where("tenant_id = ?", s.tenantID)
	if warehouseID > 0 {
		mq = mq.Where("warehouse_id = ?", warehouseID)
	}
	if from != nil {
		mq = mq.Where("created_at >= ?", *from)
	}
	if to != nil {
		mq = mq.Where("created_at <= ?", *to)
	}
	mq = mq.Group("warehouse_id, inv_sku_id")

	bq := s.repos.DB.Table("inv_balances AS b").
		Select(`b.warehouse_id, w.name AS warehouse_name, b.inv_sku_id, s.sku_code, p.name AS product_name,
			COALESCE(SUM(b.on_hand),0) AS closing,
			COALESCE(a.inbound,0) AS inbound,
			COALESCE(a.outbound,0) AS outbound`).
		Joins("JOIN warehouses w ON w.id = b.warehouse_id").
		Joins("JOIN inv_skus s ON s.id = b.inv_sku_id").
		Joins("JOIN inv_products p ON p.id = s.parent_id").
		Joins("LEFT JOIN (?) AS a ON a.warehouse_id = b.warehouse_id AND a.inv_sku_id = b.inv_sku_id", mq).
		Where("b.tenant_id = ?", s.tenantID).
		Group("b.warehouse_id, w.name, b.inv_sku_id, s.sku_code, p.name, a.inbound, a.outbound")
	if warehouseID > 0 {
		bq = bq.Where("b.warehouse_id = ?", warehouseID)
	}

	var total int64
	countQ := s.repos.DB.Table("(?) AS t", bq).Count(&total)
	if countQ.Error != nil {
		return nil, 0, countQ.Error
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var raw []struct {
		WarehouseID   uint64
		WarehouseName string
		InvSkuID      uint64
		SkuCode       string
		ProductName   string
		Closing       float64
		Inbound       float64
		Outbound      float64
	}
	if err := bq.Offset((page - 1) * pageSize).Limit(pageSize).Scan(&raw).Error; err != nil {
		return nil, 0, err
	}
	list := make([]dto.SummaryRow, 0, len(raw))
	for _, r := range raw {
		list = append(list, dto.SummaryRow{
			WarehouseID:   r.WarehouseID,
			WarehouseName: r.WarehouseName,
			InvSkuID:      r.InvSkuID,
			SkuCode:       r.SkuCode,
			ProductName:   r.ProductName,
			Opening:       r.Closing - r.Inbound + r.Outbound,
			Inbound:       r.Inbound,
			Outbound:      r.Outbound,
			Closing:       r.Closing,
		})
	}
	return list, total, nil
}

func (s *QueryService) QueryMovements(warehouseID, invSkuID uint64, moveType, docNo string, from, to *time.Time, page, pageSize int) ([]dto.MovementRow, int64, error) {
	db := s.repos.DB.Table("stock_movements AS m").
		Select(`m.id, m.created_at, m.warehouse_id, w.name AS warehouse_name, l.code AS location_code,
			m.inv_sku_id, s.sku_code, m.move_type, m.qty, m.balance_after, m.doc_type, m.doc_no, m.remark`).
		Joins("JOIN warehouses w ON w.id = m.warehouse_id").
		Joins("JOIN warehouse_locations l ON l.id = m.location_id").
		Joins("JOIN inv_skus s ON s.id = m.inv_sku_id").
		Where("m.tenant_id = ?", s.tenantID)
	if warehouseID > 0 {
		db = db.Where("m.warehouse_id = ?", warehouseID)
	}
	if invSkuID > 0 {
		db = db.Where("m.inv_sku_id = ?", invSkuID)
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
	err := db.Order("m.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&list).Error
	return list, total, err
}

func (s *QueryService) QuerySlowMoving(q dto.SlowMovingQuery) ([]dto.SlowMovingRow, int64, error) {
	days := q.Days
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	sub := s.repos.DB.Table("stock_movements").
		Select("warehouse_id, inv_sku_id, MAX(created_at) AS last_move_at").
		Where("tenant_id = ?", s.tenantID).
		Group("warehouse_id, inv_sku_id")

	db := s.repos.DB.Table("inv_balances AS b").
		Select(`b.warehouse_id, w.name AS warehouse_name, b.inv_sku_id, s.sku_code, p.name AS product_name,
			SUM(b.on_hand) AS on_hand, lm.last_move_at`).
		Joins("JOIN warehouses w ON w.id = b.warehouse_id").
		Joins("JOIN inv_skus s ON s.id = b.inv_sku_id").
		Joins("JOIN inv_products p ON p.id = s.parent_id").
		Joins("LEFT JOIN (?) AS lm ON lm.warehouse_id = b.warehouse_id AND lm.inv_sku_id = b.inv_sku_id", sub).
		Where("b.tenant_id = ? AND b.on_hand > ?", s.tenantID, q.MinOnHand).
		Where("(lm.last_move_at IS NULL OR lm.last_move_at < ?)", cutoff).
		Group("b.warehouse_id, w.name, b.inv_sku_id, s.sku_code, p.name, lm.last_move_at")
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
		WarehouseID   uint64
		WarehouseName string
		InvSkuID      uint64
		SkuCode       string
		ProductName   string
		OnHand        float64
		LastMoveAt    *time.Time
	}
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Scan(&raw).Error; err != nil {
		return nil, 0, err
	}
	now := time.Now()
	list := make([]dto.SlowMovingRow, 0, len(raw))
	for _, r := range raw {
		idle := days
		if r.LastMoveAt != nil {
			idle = int(now.Sub(*r.LastMoveAt).Hours() / 24)
		}
		list = append(list, dto.SlowMovingRow{
			WarehouseID:   r.WarehouseID,
			WarehouseName: r.WarehouseName,
			InvSkuID:      r.InvSkuID,
			SkuCode:       r.SkuCode,
			ProductName:   r.ProductName,
			OnHand:        r.OnHand,
			LastMoveAt:    r.LastMoveAt,
			IdleDays:      idle,
		})
	}
	return list, total, nil
}

// silence unused import if gorm not used directly beyond Table
var _ = gorm.ErrRecordNotFound
