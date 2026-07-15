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

func (s *DocumentService) ListOtherIn(keyword, status string, page, pageSize int) ([]model.OtherInboundOrder, int64, error) {
	q := s.db().Model(&model.OtherInboundOrder{})
	if keyword != "" {
		q = q.Where("doc_no ILIKE ?", "%"+keyword+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OtherInboundOrder
	err := q.Preload("Items").Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *DocumentService) GetOtherIn(id uint64) (*model.OtherInboundOrder, error) {
	var item model.OtherInboundOrder
	if err := s.db().Preload("Items").First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
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

func (s *DocumentService) ListOtherOut(keyword, status string, page, pageSize int) ([]model.OtherOutboundOrder, int64, error) {
	q := s.db().Model(&model.OtherOutboundOrder{})
	if keyword != "" {
		q = q.Where("doc_no ILIKE ?", "%"+keyword+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OtherOutboundOrder
	err := q.Preload("Items").Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *DocumentService) GetOtherOut(id uint64) (*model.OtherOutboundOrder, error) {
	var item model.OtherOutboundOrder
	if err := s.db().Preload("Items").First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
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

func (s *DocumentService) ListStocktakes(keyword, status string, page, pageSize int) ([]model.StocktakeOrder, int64, error) {
	q := s.db().Model(&model.StocktakeOrder{})
	if keyword != "" {
		q = q.Where("doc_no ILIKE ?", "%"+keyword+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.StocktakeOrder
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *DocumentService) GetStocktake(id uint64) (*model.StocktakeOrder, error) {
	var item model.StocktakeOrder
	if err := s.db().Preload("Items").First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &item, nil
}

func (s *DocumentService) CreateStocktake(in *dto.StocktakeCreateDTO, userID uint64) (*model.StocktakeOrder, error) {
	order := &model.StocktakeOrder{
		TenantID:    s.tenantID,
		DocNo:       GenDocNo("STK"),
		WarehouseID: in.WarehouseID,
		LocationID:  in.LocationID,
		Status:      model.DocStatusDraft,
		Remark:      in.Remark,
		CreatedBy:   userID,
	}
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(order).Error; e != nil {
			return e
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

func (s *DocumentService) ListStocktakeDetails(keyword string, page, pageSize int) ([]model.StocktakeItem, int64, error) {
	q := s.repos.DB.Table("stocktake_items AS i").
		Joins("JOIN stocktake_orders o ON o.id = i.order_id").
		Where("i.tenant_id = ? AND o.status = ?", s.tenantID, model.DocStatusPosted)
	if keyword != "" {
		q = q.Where("o.doc_no ILIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.StocktakeItem
	err := q.Select("i.*").Order("i.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&list).Error
	return list, total, err
}

// ── Transfer ──

func (s *DocumentService) ListTransfers(keyword, status string, page, pageSize int) ([]model.TransferOrder, int64, error) {
	q := s.db().Model(&model.TransferOrder{})
	if keyword != "" {
		q = q.Where("doc_no ILIKE ?", "%"+keyword+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.TransferOrder
	err := q.Preload("Items").Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *DocumentService) GetTransfer(id uint64) (*model.TransferOrder, error) {
	var item model.TransferOrder
	if err := s.db().Preload("Items").First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
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
