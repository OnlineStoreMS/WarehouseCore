package service

import (
	"time"

	"warehousecore/internal/dto"
	"warehousecore/internal/model"
	"warehousecore/internal/repo"

	"gorm.io/gorm"
)

type DocumentService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewDocumentService(repos *repo.Repos) *DocumentService {
	return &DocumentService{repos: repos}
}

func (s *DocumentService) ForTenant(tenantID uint64) *DocumentService {
	return &DocumentService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *DocumentService) db() *gorm.DB {
	return s.repos.ForTenant(s.tenantID)
}

func (s *DocumentService) expandOutboundLines(tx *gorm.DB, warehouseID, locationID uint64, skuID uint64, qty float64, docType, docNo string, docID, userID uint64) ([]MoveLine, error) {
	var sku model.InvSku
	if err := tx.Where("tenant_id = ?", s.tenantID).First(&sku, skuID).Error; err != nil {
		return nil, mapNotFound(err)
	}
	base := MoveLine{
		WarehouseID: warehouseID,
		LocationID:  locationID,
		DocType:     docType,
		DocNo:       docNo,
		DocID:       docID,
		CreatedBy:   userID,
	}
	if sku.ProductType == model.ProductTypeCombo {
		var bom model.InvBomHeader
		if err := tx.Where("tenant_id = ? AND parent_sku_id = ? AND status = 1", s.tenantID, skuID).
			Preload("Items").First(&bom).Error; err != nil {
			return nil, ErrBadRequest
		}
		var lines []MoveLine
		for _, it := range bom.Items {
			line := base
			line.InvSkuID = it.ChildSkuID
			line.Qty = -(it.Qty * qty)
			line.MoveType = model.MoveOtherOut
			line.Remark = "combo expand"
			lines = append(lines, line)
		}
		return lines, nil
	}
	line := base
	line.InvSkuID = skuID
	line.Qty = -qty
	line.MoveType = model.MoveOtherOut
	return []MoveLine{line}, nil
}

// ── Other Inbound ──

func (s *DocumentService) ListOtherIn(keyword, status, reason string, warehouseID uint64, page, pageSize int) ([]model.OtherInboundOrder, int64, error) {
	q := s.db().Model(&model.OtherInboundOrder{})
	if keyword != "" {
		q = q.Where("doc_no ILIKE ?", "%"+keyword+"%")
	}
	q = applyDocStatusFilter(q, status)
	if reason != "" {
		q = q.Where("reason = ?", reason)
	}
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OtherInboundOrder
	err := q.Preload("Items").Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	whNames := s.warehouseNameMap(collectWarehouseIDsFromOtherIn(list))
	for i := range list {
		s.enrichSkuOnItems(list[i].Items)
		list[i].WarehouseName = whNames[list[i].WarehouseID]
		var qty, amt float64
		for _, it := range list[i].Items {
			qty += it.Qty
			amt += it.Qty * it.Cost
		}
		list[i].TotalQty = qty
		list[i].TotalAmount = amt
	}
	return list, total, nil
}

func (s *DocumentService) GetOtherIn(id uint64) (*model.OtherInboundOrder, error) {
	var item model.OtherInboundOrder
	if err := s.db().Preload("Items").First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	s.enrichSkuOnItems(item.Items)
	item.WarehouseName = s.warehouseNameMap([]uint64{item.WarehouseID})[item.WarehouseID]
	var qty, amt float64
	for _, it := range item.Items {
		qty += it.Qty
		amt += it.Amount
	}
	item.TotalQty, item.TotalAmount = qty, amt
	return &item, nil
}

func (s *DocumentService) CreateOtherIn(in *dto.OtherInboundDTO, userID uint64) (*model.OtherInboundOrder, error) {
	order := &model.OtherInboundOrder{
		TenantID:    s.tenantID,
		DocNo:       GenDocNo("OIN"),
		WarehouseID: in.WarehouseID,
		LocationID:  in.LocationID,
		Reason:      in.Reason,
		Status:      model.DocStatusDraft,
		Remark:      in.Remark,
		CreatedBy:   userID,
	}
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(order).Error; e != nil {
			return e
		}
		for _, it := range in.Items {
			if it.Qty <= 0 {
				return ErrBadRequest
			}
			row := model.OtherInboundItem{
				TenantID: s.tenantID,
				OrderID:  order.ID,
				InvSkuID: it.InvSkuID,
				Qty:      it.Qty,
				Cost:     it.Cost,
				Remark:   it.Remark,
			}
			if e := tx.Create(&row).Error; e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetOtherIn(order.ID)
}

func (s *DocumentService) PostOtherIn(id, userID uint64) (*model.OtherInboundOrder, error) {
	order, err := s.GetOtherIn(id)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusDraft {
		return nil, ErrInvalidStatus
	}
	engine := NewStockEngine(s.repos.DB, s.tenantID)
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var lines []MoveLine
		for _, it := range order.Items {
			lines = append(lines, MoveLine{
				WarehouseID: order.WarehouseID,
				LocationID:  order.LocationID,
				InvSkuID:    it.InvSkuID,
				Qty:         it.Qty,
				MoveType:    model.MoveOtherIn,
				DocType:     "other_inbound",
				DocNo:       order.DocNo,
				DocID:       order.ID,
				Remark:      order.Reason,
				CreatedBy:   userID,
			})
			if it.Cost > 0 {
				if e := tx.Model(&model.InvSku{}).Where("id = ? AND tenant_id = ?", it.InvSkuID, s.tenantID).
					Update("last_purchase_price", it.Cost).Error; e != nil {
					return e
				}
			}
		}
		if e := engine.ApplyMoves(tx, lines); e != nil {
			return e
		}
		now := time.Now()
		return tx.Model(order).Updates(map[string]interface{}{
			"status":    model.DocStatusPosted,
			"posted_at": now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetOtherIn(id)
}

func (s *DocumentService) CancelOtherIn(id uint64) error {
	var order model.OtherInboundOrder
	if err := s.db().First(&order, id).Error; err != nil {
		return mapNotFound(err)
	}
	if order.Status != model.DocStatusDraft {
		return ErrInvalidStatus
	}
	return s.db().Model(&order).Update("status", model.DocStatusCancelled).Error
}

// ── Other Outbound ──

func (s *DocumentService) ListOtherOut(keyword, status, reason string, warehouseID uint64, page, pageSize int) ([]model.OtherOutboundOrder, int64, error) {
	q := s.db().Model(&model.OtherOutboundOrder{})
	if keyword != "" {
		q = q.Where("doc_no ILIKE ?", "%"+keyword+"%")
	}
	q = applyDocStatusFilter(q, status)
	if reason != "" {
		q = q.Where("reason = ?", reason)
	}
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OtherOutboundOrder
	err := q.Preload("Items").Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	whNames := s.warehouseNameMap(collectWarehouseIDsFromOtherOut(list))
	for i := range list {
		s.enrichSkuOnOutItems(list[i].Items)
		list[i].WarehouseName = whNames[list[i].WarehouseID]
		var qty, amt float64
		for _, it := range list[i].Items {
			qty += it.Qty
			amt += it.Amount
		}
		list[i].TotalQty = qty
		list[i].TotalAmount = amt
	}
	return list, total, nil
}

func (s *DocumentService) GetOtherOut(id uint64) (*model.OtherOutboundOrder, error) {
	var item model.OtherOutboundOrder
	if err := s.db().Preload("Items").First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	s.enrichSkuOnOutItems(item.Items)
	item.WarehouseName = s.warehouseNameMap([]uint64{item.WarehouseID})[item.WarehouseID]
	var qty, amt float64
	for _, it := range item.Items {
		qty += it.Qty
		amt += it.Amount
	}
	item.TotalQty, item.TotalAmount = qty, amt
	return &item, nil
}

func (s *DocumentService) CreateOtherOut(in *dto.OtherOutboundDTO, userID uint64) (*model.OtherOutboundOrder, error) {
	order := &model.OtherOutboundOrder{
		TenantID:    s.tenantID,
		DocNo:       GenDocNo("OOUT"),
		WarehouseID: in.WarehouseID,
		LocationID:  in.LocationID,
		Reason:      in.Reason,
		Status:      model.DocStatusDraft,
		Remark:      in.Remark,
		CreatedBy:   userID,
	}
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(order).Error; e != nil {
			return e
		}
		for _, it := range in.Items {
			if it.Qty <= 0 {
				return ErrBadRequest
			}
			row := model.OtherOutboundItem{
				TenantID: s.tenantID,
				OrderID:  order.ID,
				InvSkuID: it.InvSkuID,
				Qty:      it.Qty,
				Remark:   it.Remark,
			}
			if e := tx.Create(&row).Error; e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetOtherOut(order.ID)
}

func (s *DocumentService) PostOtherOut(id, userID uint64) (*model.OtherOutboundOrder, error) {
	order, err := s.GetOtherOut(id)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusDraft {
		return nil, ErrInvalidStatus
	}
	engine := NewStockEngine(s.repos.DB, s.tenantID)
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var lines []MoveLine
		for _, it := range order.Items {
			expanded, e := s.expandOutboundLines(tx, order.WarehouseID, order.LocationID, it.InvSkuID, it.Qty, "other_outbound", order.DocNo, order.ID, userID)
			if e != nil {
				return e
			}
			for i := range expanded {
				expanded[i].MoveType = model.MoveOtherOut
				expanded[i].Remark = order.Reason
			}
			lines = append(lines, expanded...)
		}
		if e := engine.ApplyMoves(tx, lines); e != nil {
			return e
		}
		now := time.Now()
		return tx.Model(order).Updates(map[string]interface{}{
			"status":    model.DocStatusPosted,
			"posted_at": now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetOtherOut(id)
}

func (s *DocumentService) CancelOtherOut(id uint64) error {
	var order model.OtherOutboundOrder
	if err := s.db().First(&order, id).Error; err != nil {
		return mapNotFound(err)
	}
	if order.Status != model.DocStatusDraft {
		return ErrInvalidStatus
	}
	return s.db().Model(&order).Update("status", model.DocStatusCancelled).Error
}

// ── Stocktake ──

func (s *DocumentService) ListStocktakes(keyword, status string, warehouseID uint64, page, pageSize int) ([]model.StocktakeOrder, int64, error) {
	q := s.db().Model(&model.StocktakeOrder{})
	if keyword != "" {
		q = q.Where("doc_no ILIKE ?", "%"+keyword+"%")
	}
	q = applyStocktakeStatusFilter(q, status)
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.StocktakeOrder
	err := q.Preload("Items").Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, len(list))
	for i := range list {
		ids[i] = list[i].WarehouseID
	}
	whNames := s.warehouseNameMap(ids)
	for i := range list {
		list[i].WarehouseName = whNames[list[i].WarehouseID]
		s.enrichStocktakeItems(list[i].Items)
		var book, count, diff float64
		for _, it := range list[i].Items {
			book += it.BookQty
			count += it.CountQty
			diff += it.DiffQty
		}
		list[i].TotalBookQty = book
		list[i].TotalCountQty = count
		list[i].TotalDiffQty = diff
	}
	return list, total, nil
}

func (s *DocumentService) GetStocktake(id uint64) (*model.StocktakeOrder, error) {
	var item model.StocktakeOrder
	if err := s.db().Preload("Items").First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	s.enrichStocktakeItems(item.Items)
	item.WarehouseName = s.warehouseNameMap([]uint64{item.WarehouseID})[item.WarehouseID]
	var book, count, diff float64
	for _, it := range item.Items {
		book += it.BookQty
		count += it.CountQty
		diff += it.DiffQty
	}
	item.TotalBookQty, item.TotalCountQty, item.TotalDiffQty = book, count, diff
	return &item, nil
}

func (s *DocumentService) CreateStocktake(in *dto.StocktakeCreateDTO, userID uint64) (*model.StocktakeOrder, error) {
	order := &model.StocktakeOrder{
		TenantID:    s.tenantID,
		DocNo:       GenDocNo("STK"),
		WarehouseID: in.WarehouseID,
		LocationID:  in.LocationID,
		CheckerName: in.CheckerName,
		Status:      model.DocStatusDraft,
		Remark:      in.Remark,
		CreatedBy:   userID,
	}
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(order).Error; e != nil {
			return e
		}
		if !in.FillAllBalances {
			return nil
		}
		bq := tx.Model(&model.InvBalance{}).Where("tenant_id = ? AND warehouse_id = ?", s.tenantID, in.WarehouseID)
		if in.LocationID > 0 {
			bq = bq.Where("location_id = ?", in.LocationID)
		}
		var bals []model.InvBalance
		if e := bq.Find(&bals).Error; e != nil {
			return e
		}
		for _, b := range bals {
			row := model.StocktakeItem{
				TenantID:   s.tenantID,
				OrderID:    order.ID,
				LocationID: b.LocationID,
				InvSkuID:   b.InvSkuID,
				BookQty:    b.OnHand,
				CountQty:   b.OnHand,
				DiffQty:    0,
			}
			if e := tx.Create(&row).Error; e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetStocktake(order.ID)
}

func (s *DocumentService) UpdateStocktake(id uint64, in *dto.StocktakeUpdateDTO) (*model.StocktakeOrder, error) {
	order, err := s.GetStocktake(id)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusDraft && order.Status != model.DocStatusCounting {
		return nil, ErrInvalidStatus
	}
	updates := map[string]interface{}{
		"checker_name": in.CheckerName,
		"remark":       in.Remark,
	}
	if err := s.db().Model(&model.StocktakeOrder{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetStocktake(id)
}

func (s *DocumentService) AddStocktakeItems(id uint64, in *dto.StocktakeAddItemsDTO) (*model.StocktakeOrder, error) {
	order, err := s.GetStocktake(id)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusDraft && order.Status != model.DocStatusCounting {
		return nil, ErrInvalidStatus
	}
	engine := NewStockEngine(s.repos.DB, s.tenantID)
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		for _, it := range in.Items {
			if it.InvSkuID == 0 {
				return ErrBadRequest
			}
			locID := it.LocationID
			if locID == 0 {
				id, e := engine.EnsureDefaultLocation(tx, order.WarehouseID)
				if e != nil {
					return e
				}
				locID = id
			}
			var exist int64
			if e := tx.Model(&model.StocktakeItem{}).
				Where("tenant_id = ? AND order_id = ? AND inv_sku_id = ? AND location_id = ?",
					s.tenantID, id, it.InvSkuID, locID).Count(&exist).Error; e != nil {
				return e
			}
			if exist > 0 {
				continue // 已存在则跳过
			}
			bookQty := 0.0
			var bal model.InvBalance
			if e := tx.Where("tenant_id = ? AND warehouse_id = ? AND location_id = ? AND inv_sku_id = ?",
				s.tenantID, order.WarehouseID, locID, it.InvSkuID).First(&bal).Error; e == nil {
				bookQty = bal.OnHand
			}
			countQty := bookQty
			if it.CountQty != nil {
				countQty = *it.CountQty
			}
			row := model.StocktakeItem{
				TenantID:   s.tenantID,
				OrderID:    id,
				LocationID: locID,
				InvSkuID:   it.InvSkuID,
				BookQty:    bookQty,
				CountQty:   countQty,
				DiffQty:    countQty - bookQty,
				Remark:     it.Remark,
			}
			if e := tx.Create(&row).Error; e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetStocktake(id)
}

func (s *DocumentService) DeleteStocktakeItem(orderID, itemID uint64) (*model.StocktakeOrder, error) {
	order, err := s.GetStocktake(orderID)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusDraft && order.Status != model.DocStatusCounting {
		return nil, ErrInvalidStatus
	}
	res := s.db().Where("tenant_id = ? AND order_id = ? AND id = ?", s.tenantID, orderID, itemID).
		Delete(&model.StocktakeItem{})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return s.GetStocktake(orderID)
}

func (s *DocumentService) StartCounting(id uint64) (*model.StocktakeOrder, error) {
	order, err := s.GetStocktake(id)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusDraft {
		return nil, ErrInvalidStatus
	}
	if err := s.db().Model(order).Update("status", model.DocStatusCounting).Error; err != nil {
		return nil, err
	}
	return s.GetStocktake(id)
}

func (s *DocumentService) SubmitCount(id uint64, in *dto.StocktakeCountDTO) (*model.StocktakeOrder, error) {
	order, err := s.GetStocktake(id)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusDraft && order.Status != model.DocStatusCounting {
		return nil, ErrInvalidStatus
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		for _, it := range in.Items {
			diff := it.CountQty
			var row model.StocktakeItem
			if e := tx.Where("tenant_id = ? AND order_id = ? AND id = ?", s.tenantID, id, it.ID).First(&row).Error; e != nil {
				return mapNotFound(e)
			}
			row.CountQty = it.CountQty
			row.DiffQty = it.CountQty - row.BookQty
			row.Remark = it.Remark
			_ = diff
			if e := tx.Save(&row).Error; e != nil {
				return e
			}
		}
		return tx.Model(&model.StocktakeOrder{}).Where("id = ?", id).Update("status", model.DocStatusReview).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetStocktake(id)
}

func (s *DocumentService) PostStocktake(id, userID uint64) (*model.StocktakeOrder, error) {
	order, err := s.GetStocktake(id)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusReview && order.Status != model.DocStatusCounting {
		return nil, ErrInvalidStatus
	}
	engine := NewStockEngine(s.repos.DB, s.tenantID)
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var lines []MoveLine
		for _, it := range order.Items {
			if it.DiffQty == 0 {
				continue
			}
			mt := model.MoveStocktakeGain
			if it.DiffQty < 0 {
				mt = model.MoveStocktakeLoss
			}
			lines = append(lines, MoveLine{
				WarehouseID: order.WarehouseID,
				LocationID:  it.LocationID,
				InvSkuID:    it.InvSkuID,
				Qty:         it.DiffQty,
				MoveType:    mt,
				DocType:     "stocktake",
				DocNo:       order.DocNo,
				DocID:       order.ID,
				CreatedBy:   userID,
			})
		}
		if e := engine.ApplyMoves(tx, lines); e != nil {
			return e
		}
		now := time.Now()
		return tx.Model(order).Updates(map[string]interface{}{
			"status":    model.DocStatusPosted,
			"posted_at": now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetStocktake(id)
}

func (s *DocumentService) CancelStocktake(id uint64) error {
	var order model.StocktakeOrder
	if err := s.db().First(&order, id).Error; err != nil {
		return mapNotFound(err)
	}
	if order.Status == model.DocStatusPosted {
		return ErrInvalidStatus
	}
	return s.db().Model(&order).Update("status", model.DocStatusCancelled).Error
}

// DeleteStocktake 物理删除盘点单及明细；已过账不可删（库存已调整）
func (s *DocumentService) DeleteStocktake(id uint64) error {
	var order model.StocktakeOrder
	if err := s.db().First(&order, id).Error; err != nil {
		return mapNotFound(err)
	}
	if order.Status == model.DocStatusPosted {
		return ErrInvalidStatus
	}
	return s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("tenant_id = ? AND order_id = ?", s.tenantID, id).Delete(&model.StocktakeItem{}).Error; e != nil {
			return e
		}
		res := tx.Where("tenant_id = ? AND id = ?", s.tenantID, id).Delete(&model.StocktakeOrder{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (s *DocumentService) ListStocktakeDetails(keyword, status string, warehouseID, stocktakeID uint64, from, to *time.Time, page, pageSize int) ([]model.StocktakeItem, int64, error) {
	q := s.repos.DB.Table("stocktake_items AS i").
		Select(`i.id, i.tenant_id, i.order_id, i.location_id, i.inv_sku_id, i.book_qty, i.count_qty, i.diff_qty, i.remark,
			o.doc_no, o.warehouse_id, o.status, o.remark AS order_remark, o.created_at, o.posted_at,
			w.name AS warehouse_name, s.sku_code, s.pick_name, s.style1, s.style2, s.style3,
			s.last_purchase_price AS unit_cost, COALESCE(p.spec_class,'') AS spec_class,
			COALESCE(p.model,'') AS model, COALESCE(p.unit,'') AS unit, l.code AS location_code`).
		Joins("JOIN stocktake_orders o ON o.id = i.order_id").
		Joins("JOIN warehouses w ON w.id = o.warehouse_id").
		Joins("JOIN inv_skus s ON s.id = i.inv_sku_id").
		Joins("JOIN inv_products p ON p.id = s.parent_id").
		Joins("LEFT JOIN warehouse_locations l ON l.id = i.location_id").
		Where("i.tenant_id = ?", s.tenantID)
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("o.doc_no ILIKE ? OR s.sku_code ILIKE ? OR s.pick_name ILIKE ?", like, like, like)
	}
	if warehouseID > 0 {
		q = q.Where("o.warehouse_id = ?", warehouseID)
	}
	if stocktakeID > 0 {
		q = q.Where("i.order_id = ?", stocktakeID)
	}
	if from != nil {
		q = q.Where("o.created_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("o.created_at <= ?", *to)
	}
	q = applyStocktakeStatusFilterCol(q, "o.status", status)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var list []model.StocktakeItem
	if err := q.Order("i.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&list).Error; err != nil {
		return nil, 0, err
	}
	for i := range list {
		list[i].BookAmount = list[i].BookQty * list[i].UnitCost
		list[i].CountAmount = list[i].CountQty * list[i].UnitCost
		list[i].DiffAmount = list[i].DiffQty * list[i].UnitCost
	}
	return list, total, nil
}

// ── Transfer ──

func (s *DocumentService) ListTransfers(keyword, status string, fromWarehouseID, toWarehouseID uint64, page, pageSize int) ([]model.TransferOrder, int64, error) {
	q := s.db().Model(&model.TransferOrder{})
	if keyword != "" {
		q = q.Where("doc_no ILIKE ?", "%"+keyword+"%")
	}
	q = applyDocStatusFilter(q, status)
	if fromWarehouseID > 0 {
		q = q.Where("from_warehouse_id = ?", fromWarehouseID)
	}
	if toWarehouseID > 0 {
		q = q.Where("to_warehouse_id = ?", toWarehouseID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.TransferOrder
	err := q.Preload("Items").Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(list)*2)
	for i := range list {
		ids = append(ids, list[i].FromWarehouseID, list[i].ToWarehouseID)
	}
	whNames := s.warehouseNameMap(ids)
	for i := range list {
		s.enrichSkuOnXferItems(list[i].Items)
		list[i].FromWarehouseName = whNames[list[i].FromWarehouseID]
		list[i].ToWarehouseName = whNames[list[i].ToWarehouseID]
		var qty, amt float64
		for _, it := range list[i].Items {
			qty += it.Qty
			amt += it.Amount
		}
		list[i].TotalQty = qty
		list[i].TotalAmount = amt
	}
	return list, total, nil
}

func (s *DocumentService) GetTransfer(id uint64) (*model.TransferOrder, error) {
	var item model.TransferOrder
	if err := s.db().Preload("Items").First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	s.enrichSkuOnXferItems(item.Items)
	wh := s.warehouseNameMap([]uint64{item.FromWarehouseID, item.ToWarehouseID})
	item.FromWarehouseName = wh[item.FromWarehouseID]
	item.ToWarehouseName = wh[item.ToWarehouseID]
	var qty, amt float64
	for _, it := range item.Items {
		qty += it.Qty
		amt += it.Amount
	}
	item.TotalQty, item.TotalAmount = qty, amt
	return &item, nil
}

func (s *DocumentService) CreateTransfer(in *dto.TransferDTO, userID uint64) (*model.TransferOrder, error) {
	toLoc := in.ResolveToLocationID()
	order := &model.TransferOrder{
		TenantID:        s.tenantID,
		DocNo:           GenDocNo("XFER"),
		FromWarehouseID: in.FromWarehouseID,
		FromLocationID:  in.FromLocationID,
		ToWarehouseID:   in.ToWarehouseID,
		ToLocationID:    toLoc,
		Status:          model.DocStatusDraft,
		Remark:          in.Remark,
		CreatedBy:       userID,
	}
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(order).Error; e != nil {
			return e
		}
		for _, it := range in.Items {
			if it.Qty <= 0 {
				return ErrBadRequest
			}
			row := model.TransferItem{
				TenantID: s.tenantID,
				OrderID:  order.ID,
				InvSkuID: it.InvSkuID,
				Qty:      it.Qty,
				Remark:   it.Remark,
			}
			if e := tx.Create(&row).Error; e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetTransfer(order.ID)
}

func (s *DocumentService) ShipTransfer(id, userID uint64) (*model.TransferOrder, error) {
	order, err := s.GetTransfer(id)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusDraft {
		return nil, ErrInvalidStatus
	}
	engine := NewStockEngine(s.repos.DB, s.tenantID)
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var lines []MoveLine
		for _, it := range order.Items {
			lines = append(lines, MoveLine{
				WarehouseID: order.FromWarehouseID,
				LocationID:  order.FromLocationID,
				InvSkuID:    it.InvSkuID,
				Qty:         -it.Qty,
				MoveType:    model.MoveTransferOut,
				DocType:     "transfer",
				DocNo:       order.DocNo,
				DocID:       order.ID,
				CreatedBy:   userID,
			})
		}
		if e := engine.ApplyMoves(tx, lines); e != nil {
			return e
		}
		now := time.Now()
		return tx.Model(order).Updates(map[string]interface{}{
			"status":     model.DocStatusInTransit,
			"shipped_at": now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetTransfer(id)
}

func (s *DocumentService) ReceiveTransfer(id, userID uint64) (*model.TransferOrder, error) {
	order, err := s.GetTransfer(id)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusInTransit {
		return nil, ErrInvalidStatus
	}
	engine := NewStockEngine(s.repos.DB, s.tenantID)
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var lines []MoveLine
		for _, it := range order.Items {
			lines = append(lines, MoveLine{
				WarehouseID: order.ToWarehouseID,
				LocationID:  order.ToLocationID,
				InvSkuID:    it.InvSkuID,
				Qty:         it.Qty,
				MoveType:    model.MoveTransferIn,
				DocType:     "transfer",
				DocNo:       order.DocNo,
				DocID:       order.ID,
				CreatedBy:   userID,
			})
		}
		if e := engine.ApplyMoves(tx, lines); e != nil {
			return e
		}
		now := time.Now()
		return tx.Model(order).Updates(map[string]interface{}{
			"status":      model.DocStatusReceived,
			"received_at": now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetTransfer(id)
}

func (s *DocumentService) CancelTransfer(id uint64) error {
	var order model.TransferOrder
	if err := s.db().First(&order, id).Error; err != nil {
		return mapNotFound(err)
	}
	if order.Status != model.DocStatusDraft {
		return ErrInvalidStatus
	}
	return s.db().Model(&order).Update("status", model.DocStatusCancelled).Error
}

func (s *DocumentService) skuCodeMap(ids []uint64) map[uint64][2]string {
	info := s.skuInfoMap(ids)
	out := map[uint64][2]string{}
	for id, v := range info {
		out[id] = [2]string{v.SkuCode, v.PickName}
	}
	return out
}

type skuEnrich struct {
	SkuCode   string
	PickName  string
	UnitCost  float64
	WeightG   float64
	Style1    string
	Style2    string
	Style3    string
	Brand     string
	SpecClass string
	Model     string
	Material  string
	Unit      string
}

func (s *DocumentService) skuInfoMap(ids []uint64) map[uint64]skuEnrich {
	out := map[uint64]skuEnrich{}
	if len(ids) == 0 {
		return out
	}
	uniq := make([]uint64, 0, len(ids))
	seen := map[uint64]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	type row struct {
		ID                uint64
		SkuCode           string
		PickName          string
		LastPurchasePrice float64
		WeightG           float64
		Style1            string
		Style2            string
		Style3            string
		Brand             string
		SpecClass         string
		Model             string
		Material          string
		Unit              string
	}
	var rows []row
	_ = s.repos.DB.Table("inv_skus AS s").
		Select(`s.id, s.sku_code, s.pick_name, s.last_purchase_price, s.weight_g, s.style1, s.style2, s.style3,
			COALESCE(p.brand,'') AS brand, COALESCE(p.spec_class,'') AS spec_class,
			COALESCE(p.model,'') AS model, COALESCE(p.material,'') AS material, COALESCE(p.unit,'') AS unit`).
		Joins("JOIN inv_products p ON p.id = s.parent_id").
		Where("s.tenant_id = ? AND s.id IN ?", s.tenantID, uniq).
		Scan(&rows)
	for _, r := range rows {
		out[r.ID] = skuEnrich{
			SkuCode: r.SkuCode, PickName: r.PickName, UnitCost: r.LastPurchasePrice, WeightG: r.WeightG,
			Style1: r.Style1, Style2: r.Style2, Style3: r.Style3,
			Brand: r.Brand, SpecClass: r.SpecClass, Model: r.Model, Material: r.Material, Unit: r.Unit,
		}
	}
	return out
}

func (s *DocumentService) locationCodeMap(ids []uint64) map[uint64]string {
	out := map[uint64]string{}
	if len(ids) == 0 {
		return out
	}
	uniq := make([]uint64, 0, len(ids))
	seen := map[uint64]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	var rows []model.WarehouseLocation
	_ = s.db().Select("id, code").Where("tenant_id = ? AND id IN ?", s.tenantID, uniq).Find(&rows)
	for _, r := range rows {
		out[r.ID] = r.Code
	}
	return out
}

func (s *DocumentService) warehouseNameMap(ids []uint64) map[uint64]string {
	out := map[uint64]string{}
	if len(ids) == 0 {
		return out
	}
	uniq := make([]uint64, 0, len(ids))
	seen := map[uint64]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	var rows []model.Warehouse
	_ = s.db().Select("id, name").Where("tenant_id = ? AND id IN ?", s.tenantID, uniq).Find(&rows)
	for _, r := range rows {
		out[r.ID] = r.Name
	}
	return out
}

func (s *DocumentService) enrichSkuOnItems(items []model.OtherInboundItem) {
	ids := make([]uint64, len(items))
	for i := range items {
		ids[i] = items[i].InvSkuID
	}
	m := s.skuInfoMap(ids)
	for i := range items {
		if v, ok := m[items[i].InvSkuID]; ok {
			items[i].SkuCode, items[i].PickName = v.SkuCode, v.PickName
			items[i].Style1, items[i].Style2, items[i].Style3 = v.Style1, v.Style2, v.Style3
			items[i].Brand, items[i].SpecClass, items[i].Model = v.Brand, v.SpecClass, v.Model
			items[i].Material, items[i].Unit = v.Material, v.Unit
			if items[i].Cost <= 0 {
				items[i].Cost = v.UnitCost
			}
		}
		items[i].Amount = items[i].Qty * items[i].Cost
	}
}

func (s *DocumentService) enrichSkuOnOutItems(items []model.OtherOutboundItem) {
	ids := make([]uint64, len(items))
	for i := range items {
		ids[i] = items[i].InvSkuID
	}
	m := s.skuInfoMap(ids)
	for i := range items {
		if v, ok := m[items[i].InvSkuID]; ok {
			items[i].SkuCode, items[i].PickName = v.SkuCode, v.PickName
			items[i].UnitCost = v.UnitCost
			items[i].Style1, items[i].Style2, items[i].Style3 = v.Style1, v.Style2, v.Style3
			items[i].Brand, items[i].SpecClass, items[i].Model = v.Brand, v.SpecClass, v.Model
			items[i].Material, items[i].Unit = v.Material, v.Unit
		}
		items[i].Amount = items[i].Qty * items[i].UnitCost
	}
}

func (s *DocumentService) enrichSkuOnXferItems(items []model.TransferItem) {
	ids := make([]uint64, len(items))
	for i := range items {
		ids[i] = items[i].InvSkuID
	}
	m := s.skuInfoMap(ids)
	for i := range items {
		if v, ok := m[items[i].InvSkuID]; ok {
			items[i].SkuCode, items[i].PickName = v.SkuCode, v.PickName
			items[i].WeightG, items[i].UnitCost = v.WeightG, v.UnitCost
			items[i].Style1, items[i].Style2, items[i].Style3 = v.Style1, v.Style2, v.Style3
			items[i].Brand, items[i].SpecClass, items[i].Model = v.Brand, v.SpecClass, v.Model
			items[i].Material, items[i].Unit = v.Material, v.Unit
		}
		items[i].Amount = items[i].Qty * items[i].UnitCost
	}
}

func (s *DocumentService) enrichStocktakeItems(items []model.StocktakeItem) {
	skuIDs := make([]uint64, len(items))
	locIDs := make([]uint64, len(items))
	for i := range items {
		skuIDs[i] = items[i].InvSkuID
		locIDs[i] = items[i].LocationID
	}
	sm := s.skuInfoMap(skuIDs)
	lm := s.locationCodeMap(locIDs)
	for i := range items {
		if v, ok := sm[items[i].InvSkuID]; ok {
			items[i].SkuCode, items[i].PickName = v.SkuCode, v.PickName
			items[i].UnitCost = v.UnitCost
			items[i].Style1, items[i].Style2, items[i].Style3 = v.Style1, v.Style2, v.Style3
			items[i].SpecClass, items[i].Model, items[i].Unit = v.SpecClass, v.Model, v.Unit
			items[i].BookAmount = items[i].BookQty * v.UnitCost
			items[i].CountAmount = items[i].CountQty * v.UnitCost
			items[i].DiffAmount = items[i].DiffQty * v.UnitCost
		}
		items[i].LocationCode = lm[items[i].LocationID]
	}
}

func applyDocStatusFilter(q *gorm.DB, status string) *gorm.DB {
	if status == "" || status == "all" {
		return q
	}
	return q.Where("status = ?", status)
}

func applyStocktakeStatusFilter(q *gorm.DB, status string) *gorm.DB {
	return applyStocktakeStatusFilterCol(q, "status", status)
}

func applyStocktakeStatusFilterCol(q *gorm.DB, col, status string) *gorm.DB {
	if status == "" || status == "all" {
		return q
	}
	// 普源「未审核/盘点中」
	if status == "open" || status == "pending" {
		return q.Where(col+" IN ?", []string{model.DocStatusDraft, model.DocStatusCounting})
	}
	return q.Where(col+" = ?", status)
}

func collectWarehouseIDsFromOtherIn(list []model.OtherInboundOrder) []uint64 {
	ids := make([]uint64, len(list))
	for i := range list {
		ids[i] = list[i].WarehouseID
	}
	return ids
}

func collectWarehouseIDsFromOtherOut(list []model.OtherOutboundOrder) []uint64 {
	ids := make([]uint64, len(list))
	for i := range list {
		ids[i] = list[i].WarehouseID
	}
	return ids
}
