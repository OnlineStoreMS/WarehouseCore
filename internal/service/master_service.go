package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"warehousecore/internal/dto"
	"warehousecore/internal/model"
	"warehousecore/internal/repo"

	"gorm.io/gorm"
)

type MasterService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewMasterService(repos *repo.Repos) *MasterService {
	return &MasterService{repos: repos}
}

func (s *MasterService) ForTenant(tenantID uint64) *MasterService {
	return &MasterService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *MasterService) db() *gorm.DB {
	return s.repos.ForTenant(s.tenantID)
}

// ── Categories ──

func (s *MasterService) ListCategories(keyword string, page, pageSize int) ([]model.InvCategory, int64, error) {
	q := s.db().Model(&model.InvCategory{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("code ILIKE ? OR name ILIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.InvCategory
	err := q.Order("sort asc, id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *MasterService) CreateCategory(in *dto.InvCategoryDTO) (*model.InvCategory, error) {
	item := &model.InvCategory{
		TenantID: s.tenantID,
		Code:     strings.TrimSpace(in.Code),
		Name:     strings.TrimSpace(in.Name),
		AliasCn:  strings.TrimSpace(in.AliasCn),
		AliasEn:  strings.TrimSpace(in.AliasEn),
		ParentID: in.ParentID,
		Sort:     in.Sort,
		Status:   statusOrDefault(in.Status),
	}
	if err := s.repos.DB.Create(item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return item, nil
}

func (s *MasterService) UpdateCategory(id uint64, in *dto.InvCategoryDTO) (*model.InvCategory, error) {
	var item model.InvCategory
	if err := s.db().First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	item.Code = strings.TrimSpace(in.Code)
	item.Name = strings.TrimSpace(in.Name)
	item.AliasCn = strings.TrimSpace(in.AliasCn)
	item.AliasEn = strings.TrimSpace(in.AliasEn)
	item.ParentID = in.ParentID
	item.Sort = in.Sort
	item.Status = statusOrDefault(in.Status)
	if err := s.repos.DB.Save(&item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return &item, nil
}

func (s *MasterService) DeleteCategory(id uint64) error {
	res := s.db().Delete(&model.InvCategory{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Products ──

func (s *MasterService) ListProducts(keyword string, categoryID uint64, uncategorized bool, productType string, page, pageSize int) ([]model.InvProduct, int64, error) {
	q := s.db().Model(&model.InvProduct{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("parent_sku ILIKE ? OR name ILIKE ?", like, like)
	}
	if uncategorized {
		q = q.Where("category_id = 0")
	} else if categoryID > 0 {
		q = q.Where("category_id = ?", categoryID)
	}
	if productType != "" {
		q = q.Where(
			"EXISTS (SELECT 1 FROM inv_skus s WHERE s.parent_id = inv_products.id AND s.tenant_id = inv_products.tenant_id AND s.product_type = ?)",
			productType,
		)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.InvProduct
	err := q.Preload("Skus").Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *MasterService) GetProduct(id uint64) (*model.InvProduct, error) {
	var item model.InvProduct
	if err := s.db().Preload("Skus").First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &item, nil
}

func (s *MasterService) CreateProduct(in *dto.InvProductDTO) (*model.InvProduct, error) {
	item := &model.InvProduct{
		TenantID:           s.tenantID,
		ParentSku:          strings.TrimSpace(in.ParentSku),
		Name:               strings.TrimSpace(in.Name),
		CategoryID:         in.CategoryID,
		PackSpecID:         in.PackSpecID,
		DefaultWarehouseID: in.DefaultWarehouseID,
		ScoreFactor:        in.ScoreFactor,
		Remark:             in.Remark,
		Pic:                in.Pic,
		AlbumPics:          in.AlbumPics,
		Status:             statusOrDefault(in.Status),
		PimSpuID:           in.PimSpuID,
	}
	if item.ScoreFactor == 0 {
		item.ScoreFactor = 1
	}
	if in.DevelopedAt != nil && *in.DevelopedAt != "" {
		if t, err := time.Parse("2006-01-02", *in.DevelopedAt); err == nil {
			item.DevelopedAt = &t
		}
	}
	if err := s.repos.DB.Create(item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return item, nil
}

func (s *MasterService) UpdateProduct(id uint64, in *dto.InvProductDTO) (*model.InvProduct, error) {
	var item model.InvProduct
	if err := s.db().First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	item.ParentSku = strings.TrimSpace(in.ParentSku)
	item.Name = strings.TrimSpace(in.Name)
	item.CategoryID = in.CategoryID
	item.PackSpecID = in.PackSpecID
	item.DefaultWarehouseID = in.DefaultWarehouseID
	item.ScoreFactor = in.ScoreFactor
	if item.ScoreFactor == 0 {
		item.ScoreFactor = 1
	}
	item.Remark = in.Remark
	item.Pic = in.Pic
	item.AlbumPics = in.AlbumPics
	item.Status = statusOrDefault(in.Status)
	item.PimSpuID = in.PimSpuID
	if in.DevelopedAt != nil {
		if *in.DevelopedAt == "" {
			item.DevelopedAt = nil
		} else if t, err := time.Parse("2006-01-02", *in.DevelopedAt); err == nil {
			item.DevelopedAt = &t
		}
	}
	if err := s.repos.DB.Save(&item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return &item, nil
}

func (s *MasterService) CreateProductWithSkus(in *dto.ProductWithSkusDTO) (*model.InvProduct, error) {
	if len(in.Skus) == 0 {
		return nil, ErrBadRequest
	}
	codes := map[string]struct{}{}
	for _, sk := range in.Skus {
		code := strings.TrimSpace(sk.SkuCode)
		if code == "" {
			return nil, ErrBadRequest
		}
		if _, ok := codes[code]; ok {
			return nil, ErrDuplicateCode
		}
		codes[code] = struct{}{}
	}
	var createdID uint64
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		prod, err := s.createProductTx(tx, &in.InvProductDTO)
		if err != nil {
			return err
		}
		defType := in.DefaultProductType
		if defType == "" {
			defType = model.ProductTypeNormal
		}
		for i := range in.Skus {
			sk := &in.Skus[i]
			pt := sk.ProductType
			if pt == "" {
				pt = defType
			}
			item := skuFromItem(s.tenantID, prod.ID, pt, sk)
			if e := tx.Create(item).Error; e != nil {
				if isUniqueViolation(e) {
					return ErrDuplicateCode
				}
				return e
			}
		}
		createdID = prod.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetProduct(createdID)
}

func (s *MasterService) UpdateProductWithSkus(id uint64, in *dto.ProductWithSkusDTO) (*model.InvProduct, error) {
	if len(in.Skus) == 0 {
		return nil, ErrBadRequest
	}
	codes := map[string]struct{}{}
	for _, sk := range in.Skus {
		code := strings.TrimSpace(sk.SkuCode)
		if code == "" {
			return nil, ErrBadRequest
		}
		if _, ok := codes[code]; ok {
			return nil, ErrDuplicateCode
		}
		codes[code] = struct{}{}
	}
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var prod model.InvProduct
		if e := tx.Where("tenant_id = ?", s.tenantID).First(&prod, id).Error; e != nil {
			return mapNotFound(e)
		}
		s.applyProductFields(&prod, &in.InvProductDTO)
		if e := tx.Save(&prod).Error; e != nil {
			if isUniqueViolation(e) {
				return ErrDuplicateCode
			}
			return e
		}
		var existing []model.InvSku
		if e := tx.Where("tenant_id = ? AND parent_id = ?", s.tenantID, id).Find(&existing).Error; e != nil {
			return e
		}
		keep := map[uint64]struct{}{}
		defType := in.DefaultProductType
		if defType == "" {
			defType = model.ProductTypeNormal
		}
		for i := range in.Skus {
			sk := &in.Skus[i]
			pt := sk.ProductType
			if pt == "" {
				pt = defType
			}
			if sk.ID > 0 {
				var item model.InvSku
				if e := tx.Where("tenant_id = ? AND parent_id = ?", s.tenantID, id).First(&item, sk.ID).Error; e != nil {
					return mapNotFound(e)
				}
				applySkuFields(&item, pt, sk)
				if e := tx.Save(&item).Error; e != nil {
					if isUniqueViolation(e) {
						return ErrDuplicateCode
					}
					return e
				}
				keep[item.ID] = struct{}{}
				continue
			}
			item := skuFromItem(s.tenantID, id, pt, sk)
			if e := tx.Create(item).Error; e != nil {
				if isUniqueViolation(e) {
					return ErrDuplicateCode
				}
				return e
			}
			keep[item.ID] = struct{}{}
		}
		for _, old := range existing {
			if _, ok := keep[old.ID]; ok {
				continue
			}
			var cnt int64
			tx.Model(&model.StockMovement{}).Where("tenant_id = ? AND inv_sku_id = ?", s.tenantID, old.ID).Count(&cnt)
			if cnt > 0 {
				return ErrHasMovements
			}
			if e := tx.Where("tenant_id = ?", s.tenantID).Delete(&model.InvSku{}, old.ID).Error; e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetProduct(id)
}

func (s *MasterService) createProductTx(tx *gorm.DB, in *dto.InvProductDTO) (*model.InvProduct, error) {
	item := &model.InvProduct{TenantID: s.tenantID}
	s.applyProductFields(item, in)
	if err := tx.Create(item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return item, nil
}

func (s *MasterService) applyProductFields(item *model.InvProduct, in *dto.InvProductDTO) {
	item.ParentSku = strings.TrimSpace(in.ParentSku)
	item.Name = strings.TrimSpace(in.Name)
	item.CategoryID = in.CategoryID
	item.PackSpecID = in.PackSpecID
	item.DefaultWarehouseID = in.DefaultWarehouseID
	item.ScoreFactor = in.ScoreFactor
	if item.ScoreFactor == 0 {
		item.ScoreFactor = 1
	}
	item.Remark = in.Remark
	item.Pic = in.Pic
	item.AlbumPics = in.AlbumPics
	item.Status = statusOrDefault(in.Status)
	item.PimSpuID = in.PimSpuID
	if in.DevelopedAt != nil {
		if *in.DevelopedAt == "" {
			item.DevelopedAt = nil
		} else if t, err := time.Parse("2006-01-02", *in.DevelopedAt); err == nil {
			item.DevelopedAt = &t
		}
	}

	item.Features = in.Features
	item.AliasCn = in.AliasCn
	item.AliasEn = in.AliasEn
	item.DeclareWeightG = in.DeclareWeightG
	item.DeclaredValue = in.DeclaredValue
	item.OriginCountryCode = in.OriginCountryCode
	item.HSCode = in.HSCode
	item.ExportDeclaredValue = in.ExportDeclaredValue

	item.PurchaseChannel = in.PurchaseChannel
	item.Purchaser = in.Purchaser
	item.MinPurchasePrice = in.MinPurchasePrice
	item.StockMinAmount = in.StockMinAmount

	item.PackFee = in.PackFee
	item.PackageCount = in.PackageCount
	item.OutLong = in.OutLong
	item.OutWide = in.OutWide
	item.OutHigh = in.OutHigh
	item.OutGrossWeight = in.OutGrossWeight
	item.OutNetWeight = in.OutNetWeight
	item.InLong = in.InLong
	item.InWide = in.InWide
	item.InHigh = in.InHigh
	item.InGrossWeight = in.InGrossWeight
	item.InNetWeight = in.InNetWeight
	item.PackMsg = in.PackMsg

	item.ShopTitle = in.ShopTitle
	item.Brand = in.Brand
	item.SpecClass = in.SpecClass
	item.Model = in.Model
	item.Material = in.Material
	item.Style = in.Style
	item.Season = in.Season
	item.Unit = in.Unit
	item.RetailPrice = in.RetailPrice
	item.BatchPrice = in.BatchPrice
	item.MaxSalePrice = in.MaxSalePrice
	item.MinSalePrice = in.MinSalePrice
	item.MarketPrice = in.MarketPrice
}

func skuFromItem(tenantID, parentID uint64, productType string, in *dto.ProductSkuItemDTO) *model.InvSku {
	item := &model.InvSku{
		TenantID:    tenantID,
		ParentID:    parentID,
		ProductType: productType,
	}
	applySkuFields(item, productType, in)
	return item
}

func applySkuFields(item *model.InvSku, productType string, in *dto.ProductSkuItemDTO) {
	item.SkuCode = strings.TrimSpace(in.SkuCode)
	item.Pic = in.Pic
	item.Status = in.Status
	if item.Status == "" {
		item.Status = "active"
	}
	item.ProductType = productType
	if item.ProductType == "" {
		item.ProductType = model.ProductTypeNormal
	}
	item.GoodsKind = in.GoodsKind
	if item.GoodsKind == "" {
		item.GoodsKind = model.GoodsKindNormal
	}
	item.PickName = in.PickName
	item.Style1 = in.Style1
	item.Style2 = in.Style2
	item.Style3 = in.Style3
	item.WeightG = in.WeightG
	item.LastPurchasePrice = in.LastPurchasePrice
	item.MinPurchasePrice = in.MinPurchasePrice
	item.RetailPrice = in.RetailPrice
	item.Description = in.Description
	item.UPC = in.UPC
	item.ASIN = in.ASIN
	item.SupplierItemNo = in.SupplierItemNo
}

func (s *MasterService) DeleteProduct(id uint64) error {
	var skuIDs []uint64
	s.db().Model(&model.InvSku{}).Where("parent_id = ?", id).Pluck("id", &skuIDs)
	if len(skuIDs) > 0 {
		var cnt int64
		s.repos.DB.Model(&model.StockMovement{}).Where("tenant_id = ? AND inv_sku_id IN ?", s.tenantID, skuIDs).Count(&cnt)
		if cnt > 0 {
			return ErrHasMovements
		}
	}
	return s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND parent_id = ?", s.tenantID, id).Delete(&model.InvSku{}).Error; err != nil {
			return err
		}
		res := tx.Where("tenant_id = ?", s.tenantID).Delete(&model.InvProduct{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// ── SKUs ──

func (s *MasterService) ListSkus(keyword, productType string, categoryID uint64, uncategorized bool, page, pageSize int) ([]dto.SkuListRow, int64, error) {
	q := s.repos.DB.Table("inv_skus AS s").
		Select("s.*, p.parent_sku AS parent_sku_code, p.name AS product_name, p.category_id AS category_id").
		Joins("JOIN inv_products p ON p.id = s.parent_id AND p.tenant_id = s.tenant_id").
		Where("s.tenant_id = ?", s.tenantID)
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("s.sku_code ILIKE ? OR s.pick_name ILIKE ? OR s.upc ILIKE ? OR s.asin ILIKE ? OR p.parent_sku ILIKE ? OR p.name ILIKE ?",
			like, like, like, like, like, like)
	}
	if productType != "" {
		q = q.Where("s.product_type = ?", productType)
	}
	if uncategorized {
		q = q.Where("p.category_id = 0")
	} else if categoryID > 0 {
		q = q.Where("p.category_id = ?", categoryID)
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []dto.SkuListRow
	err := q.Order("s.id desc").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&list).Error
	return list, total, err
}

func (s *MasterService) GetSku(id uint64) (*model.InvSku, error) {
	var item model.InvSku
	if err := s.db().First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &item, nil
}

func (s *MasterService) CreateSku(in *dto.InvSkuDTO) (*model.InvSku, error) {
	var parent model.InvProduct
	if err := s.db().First(&parent, in.ParentID).Error; err != nil {
		return nil, mapNotFound(err)
	}
	pt := in.ProductType
	if pt == "" {
		pt = model.ProductTypeNormal
	}
	gk := in.GoodsKind
	if gk == "" {
		gk = model.GoodsKindNormal
	}
	st := in.Status
	if st == "" {
		st = "active"
	}
	item := &model.InvSku{
		TenantID:          s.tenantID,
		ParentID:          in.ParentID,
		SkuCode:           strings.TrimSpace(in.SkuCode),
		Pic:               in.Pic,
		Status:            st,
		ProductType:       pt,
		GoodsKind:         gk,
		PickName:          in.PickName,
		Style1:            in.Style1,
		Style2:            in.Style2,
		Style3:            in.Style3,
		WeightG:           in.WeightG,
		LastPurchasePrice: in.LastPurchasePrice,
		MinPurchasePrice:  in.MinPurchasePrice,
		RetailPrice:       in.RetailPrice,
		Description:       in.Description,
		UPC:               in.UPC,
		ASIN:              in.ASIN,
		SupplierItemNo:    in.SupplierItemNo,
		PimSkuID:          in.PimSkuID,
	}
	if err := s.repos.DB.Create(item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return item, nil
}

func (s *MasterService) UpdateSku(id uint64, in *dto.InvSkuDTO) (*model.InvSku, error) {
	var item model.InvSku
	if err := s.db().First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	item.SkuCode = strings.TrimSpace(in.SkuCode)
	item.Pic = in.Pic
	if in.Status != "" {
		item.Status = in.Status
	}
	if in.ProductType != "" {
		item.ProductType = in.ProductType
	}
	if in.GoodsKind != "" {
		item.GoodsKind = in.GoodsKind
	}
	item.PickName = in.PickName
	item.Style1 = in.Style1
	item.Style2 = in.Style2
	item.Style3 = in.Style3
	item.WeightG = in.WeightG
	item.LastPurchasePrice = in.LastPurchasePrice
	item.MinPurchasePrice = in.MinPurchasePrice
	item.RetailPrice = in.RetailPrice
	item.Description = in.Description
	item.UPC = in.UPC
	item.ASIN = in.ASIN
	item.SupplierItemNo = in.SupplierItemNo
	item.PimSkuID = in.PimSkuID
	if in.ParentID > 0 {
		item.ParentID = in.ParentID
	}
	if err := s.repos.DB.Save(&item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return &item, nil
}

func (s *MasterService) DeleteSku(id uint64) error {
	var cnt int64
	s.repos.DB.Model(&model.StockMovement{}).Where("tenant_id = ? AND inv_sku_id = ?", s.tenantID, id).Count(&cnt)
	if cnt > 0 {
		return ErrHasMovements
	}
	res := s.db().Delete(&model.InvSku{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Pack specs ──

func (s *MasterService) ListPackSpecs(keyword string, page, pageSize int) ([]model.InvPackSpec, int64, error) {
	q := s.db().Model(&model.InvPackSpec{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name ILIKE ? OR remark ILIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.InvPackSpec
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *MasterService) CreatePackSpec(in *dto.InvPackSpecDTO) (*model.InvPackSpec, error) {
	item := &model.InvPackSpec{
		TenantID: s.tenantID,
		Name:     strings.TrimSpace(in.Name),
		Cost:     in.Cost,
		WeightG:  in.WeightG,
		Remark:   in.Remark,
		Status:   statusOrDefault(in.Status),
	}
	if item.Name == "" {
		return nil, ErrBadRequest
	}
	if err := s.repos.DB.Create(item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return item, nil
}

func (s *MasterService) UpdatePackSpec(id uint64, in *dto.InvPackSpecDTO) (*model.InvPackSpec, error) {
	var item model.InvPackSpec
	if err := s.db().First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	item.Name = strings.TrimSpace(in.Name)
	item.Cost = in.Cost
	item.WeightG = in.WeightG
	item.Remark = in.Remark
	item.Status = statusOrDefault(in.Status)
	if item.Name == "" {
		return nil, ErrBadRequest
	}
	if err := s.repos.DB.Save(&item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return &item, nil
}

func (s *MasterService) DeletePackSpec(id uint64) error {
	return s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND pack_spec_id = ?", s.tenantID, id).Delete(&model.InvPackSpecSku{}).Error; err != nil {
			return err
		}
		res := tx.Where("tenant_id = ?", s.tenantID).Delete(&model.InvPackSpec{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (s *MasterService) ListPackSpecSkus(packSpecID uint64) ([]dto.PackSpecSkuRow, error) {
	var list []dto.PackSpecSkuRow
	err := s.repos.DB.Table("inv_pack_spec_skus AS ps").
		Select("ps.id, ps.pack_spec_id, ps.inv_sku_id, s.sku_code, s.pick_name, ps.qty_min, ps.qty_max, ps.remark").
		Joins("JOIN inv_skus s ON s.id = ps.inv_sku_id AND s.tenant_id = ps.tenant_id").
		Where("ps.tenant_id = ? AND ps.pack_spec_id = ?", s.tenantID, packSpecID).
		Order("ps.id asc").
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].NumRange = formatQtyRange(list[i].QtyMin, list[i].QtyMax)
	}
	return list, nil
}

func formatQtyRange(min, max float64) string {
	if max > 0 {
		return fmt.Sprintf("%g~%g", min, max)
	}
	if min > 0 {
		return fmt.Sprintf("%g+", min)
	}
	return "-"
}

func (s *MasterService) BindPackSpecSku(in *dto.InvPackSpecSkuDTO) (*model.InvPackSpecSku, error) {
	var pack model.InvPackSpec
	if err := s.db().First(&pack, in.PackSpecID).Error; err != nil {
		return nil, mapNotFound(err)
	}
	var sku model.InvSku
	if err := s.db().First(&sku, in.InvSkuID).Error; err != nil {
		return nil, mapNotFound(err)
	}
	item := &model.InvPackSpecSku{
		TenantID:   s.tenantID,
		PackSpecID: in.PackSpecID,
		InvSkuID:   in.InvSkuID,
		QtyMin:     in.QtyMin,
		QtyMax:     in.QtyMax,
		Remark:     in.Remark,
	}
	if err := s.repos.DB.Create(item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return item, nil
}

func (s *MasterService) UpdatePackSpecSku(id uint64, in *dto.InvPackSpecSkuDTO) (*model.InvPackSpecSku, error) {
	var item model.InvPackSpecSku
	if err := s.db().First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	if in.InvSkuID > 0 {
		item.InvSkuID = in.InvSkuID
	}
	item.QtyMin = in.QtyMin
	item.QtyMax = in.QtyMax
	item.Remark = in.Remark
	if err := s.repos.DB.Save(&item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return &item, nil
}

func (s *MasterService) UnbindPackSpecSku(id uint64) error {
	res := s.db().Delete(&model.InvPackSpecSku{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ── BOM ──

func (s *MasterService) ListBoms(page, pageSize int) ([]model.InvBomHeader, int64, error) {
	q := s.db().Model(&model.InvBomHeader{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.InvBomHeader
	err := q.Preload("Items").Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *MasterService) GetBom(id uint64) (*model.InvBomHeader, error) {
	var item model.InvBomHeader
	if err := s.db().Preload("Items").First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &item, nil
}

func (s *MasterService) SaveBom(in *dto.BomDTO) (*model.InvBomHeader, error) {
	var sku model.InvSku
	if err := s.db().First(&sku, in.ParentSkuID).Error; err != nil {
		return nil, mapNotFound(err)
	}
	if in.BomType != model.ProductTypeCombo && in.BomType != model.ProductTypeAssembly {
		return nil, ErrBadRequest
	}
	var header model.InvBomHeader
	findErr := s.db().Where("parent_sku_id = ?", in.ParentSkuID).First(&header).Error
	if err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			header = model.InvBomHeader{
				TenantID:    s.tenantID,
				ParentSkuID: in.ParentSkuID,
				BomType:     in.BomType,
				Remark:      in.Remark,
				Status:      statusOrDefault(in.Status),
			}
			if e := tx.Create(&header).Error; e != nil {
				return e
			}
		} else if findErr != nil {
			return findErr
		} else {
			header.BomType = in.BomType
			header.Remark = in.Remark
			header.Status = statusOrDefault(in.Status)
			if e := tx.Save(&header).Error; e != nil {
				return e
			}
			if e := tx.Where("bom_id = ?", header.ID).Delete(&model.InvBomItem{}).Error; e != nil {
				return e
			}
		}
		for _, it := range in.Items {
			if it.Qty <= 0 {
				return ErrBadRequest
			}
			row := model.InvBomItem{
				TenantID:   s.tenantID,
				BomID:      header.ID,
				ChildSkuID: it.ChildSkuID,
				Qty:        it.Qty,
				Remark:     it.Remark,
			}
			if e := tx.Create(&row).Error; e != nil {
				return e
			}
		}
		if sku.ProductType != in.BomType {
			if e := tx.Model(&sku).Update("product_type", in.BomType).Error; e != nil {
				return e
			}
		}
		return nil
	}); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return s.GetBom(header.ID)
}

func (s *MasterService) DeleteBom(id uint64) error {
	return s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND bom_id = ?", s.tenantID, id).Delete(&model.InvBomItem{}).Error; err != nil {
			return err
		}
		res := tx.Where("tenant_id = ?", s.tenantID).Delete(&model.InvBomHeader{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// ── Warehouses ──

func (s *MasterService) ListWarehouses(keyword string, page, pageSize int) ([]model.Warehouse, int64, error) {
	q := s.db().Model(&model.Warehouse{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("code ILIKE ? OR name ILIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Warehouse
	err := q.Order("is_default desc, id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *MasterService) CreateWarehouse(in *dto.WarehouseDTO) (*model.Warehouse, error) {
	wt := in.Type
	if wt == "" {
		wt = model.WarehouseTypeCentral
	}
	item := &model.Warehouse{
		TenantID:  s.tenantID,
		Code:      strings.TrimSpace(in.Code),
		Name:      strings.TrimSpace(in.Name),
		Type:      wt,
		Address:   in.Address,
		Contact:   in.Contact,
		Phone:     in.Phone,
		Status:    statusOrDefault(in.Status),
		IsDefault: in.IsDefault,
	}
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if item.IsDefault == 1 {
			if e := tx.Model(&model.Warehouse{}).Where("tenant_id = ?", s.tenantID).Update("is_default", 0).Error; e != nil {
				return e
			}
		}
		if e := tx.Create(item).Error; e != nil {
			return e
		}
		loc := model.WarehouseLocation{
			TenantID:    s.tenantID,
			WarehouseID: item.ID,
			Code:        model.DefaultLocationCode,
			Status:      1,
		}
		return tx.Create(&loc).Error
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return item, nil
}

func (s *MasterService) UpdateWarehouse(id uint64, in *dto.WarehouseDTO) (*model.Warehouse, error) {
	var item model.Warehouse
	if err := s.db().First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	item.Code = strings.TrimSpace(in.Code)
	item.Name = strings.TrimSpace(in.Name)
	if in.Type != "" {
		item.Type = in.Type
	}
	item.Address = in.Address
	item.Contact = in.Contact
	item.Phone = in.Phone
	item.Status = statusOrDefault(in.Status)
	item.IsDefault = in.IsDefault
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if item.IsDefault == 1 {
			if e := tx.Model(&model.Warehouse{}).Where("tenant_id = ? AND id <> ?", s.tenantID, id).Update("is_default", 0).Error; e != nil {
				return e
			}
		}
		return tx.Save(&item).Error
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return &item, nil
}

func (s *MasterService) DeleteWarehouse(id uint64) error {
	var cnt int64
	s.repos.DB.Model(&model.InvBalance{}).Where("tenant_id = ? AND warehouse_id = ? AND on_hand <> 0", s.tenantID, id).Count(&cnt)
	if cnt > 0 {
		return ErrBadRequest
	}
	return s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND warehouse_id = ?", s.tenantID, id).Delete(&model.WarehouseLocation{}).Error; err != nil {
			return err
		}
		res := tx.Where("tenant_id = ?", s.tenantID).Delete(&model.Warehouse{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// ── Locations ──

func (s *MasterService) ListLocations(warehouseID uint64, keyword string, page, pageSize int) ([]model.WarehouseLocation, int64, error) {
	q := s.db().Model(&model.WarehouseLocation{})
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("code ILIKE ?", like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.WarehouseLocation
	err := q.Order("warehouse_id asc, code asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *MasterService) CreateLocation(in *dto.LocationDTO) (*model.WarehouseLocation, error) {
	var wh model.Warehouse
	if err := s.db().First(&wh, in.WarehouseID).Error; err != nil {
		return nil, mapNotFound(err)
	}
	item := &model.WarehouseLocation{
		TenantID:    s.tenantID,
		WarehouseID: in.WarehouseID,
		Code:        strings.TrimSpace(in.Code),
		Zone:        in.Zone,
		Aisle:       in.Aisle,
		Shelf:       in.Shelf,
		Bin:         in.Bin,
		Status:      statusOrDefault(in.Status),
	}
	if err := s.repos.DB.Create(item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return item, nil
}

func (s *MasterService) UpdateLocation(id uint64, in *dto.LocationDTO) (*model.WarehouseLocation, error) {
	var item model.WarehouseLocation
	if err := s.db().First(&item, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	item.Code = strings.TrimSpace(in.Code)
	item.Zone = in.Zone
	item.Aisle = in.Aisle
	item.Shelf = in.Shelf
	item.Bin = in.Bin
	item.Status = statusOrDefault(in.Status)
	if err := s.repos.DB.Save(&item).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateCode
		}
		return nil, err
	}
	return &item, nil
}

func (s *MasterService) DeleteLocation(id uint64) error {
	var loc model.WarehouseLocation
	if err := s.db().First(&loc, id).Error; err != nil {
		return mapNotFound(err)
	}
	if loc.Code == model.DefaultLocationCode {
		return ErrBadRequest
	}
	var cnt int64
	s.repos.DB.Model(&model.InvBalance{}).Where("tenant_id = ? AND location_id = ? AND on_hand <> 0", s.tenantID, id).Count(&cnt)
	if cnt > 0 {
		return ErrBadRequest
	}
	res := s.db().Delete(&model.WarehouseLocation{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func statusOrDefault(s int8) int8 {
	if s == 0 {
		return 1
	}
	return s
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}
