package service

import (
	"strings"
	"time"

	"warehousecore/internal/dto"
	"warehousecore/internal/model"
	"warehousecore/internal/repo"

	"gorm.io/gorm"
)

type IntegrationService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewIntegrationService(repos *repo.Repos) *IntegrationService {
	return &IntegrationService{repos: repos}
}

func (s *IntegrationService) ForTenant(tenantID uint64) *IntegrationService {
	return &IntegrationService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *IntegrationService) db() *gorm.DB {
	return s.repos.ForTenant(s.tenantID)
}

// ── PIM mapping / 商品库映射 ──

func (s *IntegrationService) ListPimMappings(keyword string, page, pageSize int) ([]model.PimSkuMapping, int64, error) {
	q := s.db().Model(&model.PimSkuMapping{})
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(`pim_sku_code ILIKE ? OR CAST(pim_sku_id AS TEXT) ILIKE ? OR CAST(inv_sku_id AS TEXT) ILIKE ? OR remark ILIKE ?
			OR inv_sku_id IN (SELECT id FROM inv_skus WHERE tenant_id = ? AND (sku_code ILIKE ? OR pick_name ILIKE ?))`,
			like, like, like, like, s.tenantID, like, like)
	}
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
	var list []model.PimSkuMapping
	if err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	s.enrichPimMappings(list)
	return list, total, nil
}

func (s *IntegrationService) enrichPimMappings(list []model.PimSkuMapping) {
	if len(list) == 0 {
		return
	}
	ids := make([]uint64, 0, len(list))
	for _, m := range list {
		ids = append(ids, m.InvSkuID)
	}
	var skus []model.InvSku
	_ = s.db().Where("id IN ?", ids).Find(&skus).Error
	byID := make(map[uint64]model.InvSku, len(skus))
	parentIDs := make([]uint64, 0, len(skus))
	for _, sku := range skus {
		byID[sku.ID] = sku
		if sku.ParentID > 0 {
			parentIDs = append(parentIDs, sku.ParentID)
		}
	}
	var products []model.InvProduct
	if len(parentIDs) > 0 {
		_ = s.db().Where("id IN ?", parentIDs).Find(&products).Error
	}
	prodByID := make(map[uint64]model.InvProduct, len(products))
	for _, p := range products {
		prodByID[p.ID] = p
	}
	for i := range list {
		if sku, ok := byID[list[i].InvSkuID]; ok {
			list[i].InvSkuCode = sku.SkuCode
			list[i].PickName = sku.PickName
			if p, ok := prodByID[sku.ParentID]; ok {
				list[i].ProductName = p.Name
			}
		}
	}
}

func (s *IntegrationService) UpsertPimMapping(in *dto.PimMappingDTO) (*model.PimSkuMapping, error) {
	var sku model.InvSku
	if err := s.db().First(&sku, in.InvSkuID).Error; err != nil {
		return nil, mapNotFound(err)
	}
	var m model.PimSkuMapping
	err := s.db().Where("inv_sku_id = ?", in.InvSkuID).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		m = model.PimSkuMapping{
			TenantID:   s.tenantID,
			InvSkuID:   in.InvSkuID,
			PimSkuID:   in.PimSkuID,
			PimSkuCode: in.PimSkuCode,
			Remark:     in.Remark,
		}
		if e := s.repos.DB.Create(&m).Error; e != nil {
			if isUniqueViolation(e) {
				return nil, ErrDuplicateCode
			}
			return nil, e
		}
	} else if err != nil {
		return nil, err
	} else {
		m.PimSkuID = in.PimSkuID
		m.PimSkuCode = in.PimSkuCode
		m.Remark = in.Remark
		if e := s.repos.DB.Save(&m).Error; e != nil {
			if isUniqueViolation(e) {
				return nil, ErrDuplicateCode
			}
			return nil, e
		}
	}
	_ = s.repos.DB.Model(&sku).Update("pim_sku_id", in.PimSkuID)
	return &m, nil
}

func (s *IntegrationService) DeletePimMapping(id uint64) error {
	res := s.db().Delete(&model.PimSkuMapping{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// PurchaseInbound posts stock from SupplyCore GRN (M4 reserved API).
func (s *IntegrationService) PurchaseInbound(in *dto.PurchaseInboundDTO, userID uint64) (*model.OtherInboundOrder, error) {
	doc := DocumentService{repos: s.repos, tenantID: s.tenantID}
	order, err := doc.CreateOtherIn(&dto.OtherInboundDTO{
		WarehouseID: in.WarehouseID,
		LocationID:  in.LocationID,
		Reason:      "purchase",
		Remark:      in.Remark,
		Items:       in.Items,
	}, userID)
	if err != nil {
		return nil, err
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
				MoveType:    model.MovePurchaseIn,
				DocType:     "purchase_inbound",
				DocNo:       order.DocNo,
				DocID:       order.ID,
				RefDocType:  in.RefDocType,
				RefDocID:    in.RefDocID,
				Remark:      in.RefDocNo,
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
			"reason":    "purchase",
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return doc.GetOtherIn(order.ID)
}

// TransferToStore deducts central warehouse stock for store intake (M4 reserved; StoreCore confirms separately).
func (s *IntegrationService) TransferToStore(in *dto.StoreTransferDTO, userID uint64) (*model.TransferOrder, error) {
	doc := DocumentService{repos: s.repos, tenantID: s.tenantID}
	// Use a synthetic transit warehouse pattern: ship out of from-warehouse only; store side is external.
	order := &model.TransferOrder{
		TenantID:        s.tenantID,
		DocNo:           GenDocNo("STF"),
		FromWarehouseID: in.FromWarehouseID,
		FromLocationID:  in.FromLocationID,
		ToWarehouseID:   in.FromWarehouseID, // placeholder; external store
		ToLocationID:    0,
		Status:          model.DocStatusDraft,
		Remark:          in.Remark + " | store_id=" + itoa(in.StoreID),
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
	shipped, err := doc.ShipTransfer(order.ID, userID)
	if err != nil {
		return nil, err
	}
	// Mark received without inbound (stock already left central warehouse for store)
	now := time.Now()
	if err := s.repos.DB.Model(shipped).Updates(map[string]interface{}{
		"status":      model.DocStatusReceived,
		"received_at": now,
		"remark":      shipped.Remark + " | store_transfer_pending_confirm",
	}).Error; err != nil {
		return nil, err
	}
	return doc.GetTransfer(order.ID)
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
