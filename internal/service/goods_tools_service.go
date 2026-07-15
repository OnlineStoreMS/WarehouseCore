package service

import (
	"math"
	"strings"

	"warehousecore/internal/dto"
	"warehousecore/internal/model"

	"gorm.io/gorm"
)

// ── 商品费用设置 ──

func (s *MasterService) GetGoodsFeeSettings() (*dto.GoodsFeeSettingsDTO, error) {
	base, err := s.ensureFeeBase()
	if err != nil {
		return nil, err
	}
	var scores []model.InvScoreWeightRule
	if e := s.db().Order("sort asc, id asc").Find(&scores).Error; e != nil {
		return nil, e
	}
	var qtys []model.InvOrderQtyCoeff
	if e := s.db().Order("sort asc, id asc").Find(&qtys).Error; e != nil {
		return nil, e
	}
	out := &dto.GoodsFeeSettingsDTO{
		StoreFee:      base.StoreFee,
		FixedStoreFee: base.FixedStoreFee,
		PackFee:       base.PackFee,
		ScoreRules:    make([]dto.ScoreWeightRuleDTO, 0, len(scores)),
		QtyCoeffs:     make([]dto.OrderQtyCoeffDTO, 0, len(qtys)),
	}
	for _, r := range scores {
		out.ScoreRules = append(out.ScoreRules, dto.ScoreWeightRuleDTO{
			ID: r.ID, WeightMinG: r.WeightMinG, WeightMaxG: r.WeightMaxG, ScoreFactor: r.ScoreFactor, Sort: r.Sort,
		})
	}
	for _, r := range qtys {
		out.QtyCoeffs = append(out.QtyCoeffs, dto.OrderQtyCoeffDTO{
			ID: r.ID, QtyMin: r.QtyMin, QtyMax: r.QtyMax, Coeff: r.Coeff, Sort: r.Sort,
		})
	}
	return out, nil
}

func (s *MasterService) SaveGoodsFeeSettings(in *dto.GoodsFeeSettingsDTO) (*dto.GoodsFeeSettingsDTO, error) {
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		base, e := s.ensureFeeBaseTx(tx)
		if e != nil {
			return e
		}
		base.StoreFee = in.StoreFee
		base.FixedStoreFee = in.FixedStoreFee
		base.PackFee = in.PackFee
		if e := tx.Save(base).Error; e != nil {
			return e
		}
		if e := tx.Where("tenant_id = ?", s.tenantID).Delete(&model.InvScoreWeightRule{}).Error; e != nil {
			return e
		}
		for i, r := range in.ScoreRules {
			item := model.InvScoreWeightRule{
				TenantID: s.tenantID, WeightMinG: r.WeightMinG, WeightMaxG: r.WeightMaxG,
				ScoreFactor: r.ScoreFactor, Sort: r.Sort,
			}
			if item.ScoreFactor == 0 {
				item.ScoreFactor = 1
			}
			if item.Sort == 0 {
				item.Sort = i + 1
			}
			if e := tx.Create(&item).Error; e != nil {
				return e
			}
		}
		if e := tx.Where("tenant_id = ?", s.tenantID).Delete(&model.InvOrderQtyCoeff{}).Error; e != nil {
			return e
		}
		for i, r := range in.QtyCoeffs {
			item := model.InvOrderQtyCoeff{
				TenantID: s.tenantID, QtyMin: r.QtyMin, QtyMax: r.QtyMax, Coeff: r.Coeff, Sort: r.Sort,
			}
			if item.Coeff == 0 {
				item.Coeff = 1
			}
			if item.Sort == 0 {
				item.Sort = i + 1
			}
			if e := tx.Create(&item).Error; e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetGoodsFeeSettings()
}

func (s *MasterService) ensureFeeBase() (*model.InvGoodsFeeBase, error) {
	return s.ensureFeeBaseTx(s.repos.DB)
}

func (s *MasterService) ensureFeeBaseTx(tx *gorm.DB) (*model.InvGoodsFeeBase, error) {
	var base model.InvGoodsFeeBase
	err := tx.Where("tenant_id = ?", s.tenantID).First(&base).Error
	if err == nil {
		return &base, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	base = model.InvGoodsFeeBase{TenantID: s.tenantID}
	if e := tx.Create(&base).Error; e != nil {
		return nil, e
	}
	return &base, nil
}

// ── 商品重量检测 ──

func (s *MasterService) GetSkuByCode(skuCode string) (*dto.SkuListRow, error) {
	code := strings.TrimSpace(skuCode)
	if code == "" {
		return nil, ErrBadRequest
	}
	var row dto.SkuListRow
	err := s.repos.DB.Table("inv_skus AS s").
		Select("s.*, p.parent_sku AS parent_sku_code, p.name AS product_name, p.category_id AS category_id").
		Joins("JOIN inv_products p ON p.id = s.parent_id AND p.tenant_id = s.tenant_id").
		Where("s.tenant_id = ? AND s.sku_code = ?", s.tenantID, code).
		Take(&row).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (s *MasterService) UpdateSkuWeightByCode(in *dto.UpdateSkuWeightDTO) (*dto.SkuListRow, error) {
	code := strings.TrimSpace(in.SkuCode)
	if code == "" || in.WeightG < 0 {
		return nil, ErrBadRequest
	}
	var sku model.InvSku
	if err := s.db().Where("sku_code = ?", code).First(&sku).Error; err != nil {
		return nil, mapNotFound(err)
	}
	sku.WeightG = in.WeightG
	if err := s.repos.DB.Save(&sku).Error; err != nil {
		return nil, err
	}
	return s.GetSkuByCode(code)
}

// ── 商品利润试算 ──

func (s *MasterService) ListProfitTrials(keyword string, page, pageSize int) ([]model.InvProfitTrial, int64, error) {
	q := s.db().Model(&model.InvProfitTrial{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("sku ILIKE ? OR parent_sku ILIKE ? OR shop_sku ILIKE ? OR sku_name ILIKE ? OR shop_name ILIKE ?",
			like, like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.InvProfitTrial
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *MasterService) UpsertProfitTrial(in *dto.ProfitTrialDTO) (*model.InvProfitTrial, error) {
	sku := strings.TrimSpace(in.SKU)
	if sku == "" {
		return nil, ErrBadRequest
	}
	item := &model.InvProfitTrial{}
	if in.ID > 0 {
		if err := s.db().First(item, in.ID).Error; err != nil {
			return nil, mapNotFound(err)
		}
	} else {
		item.TenantID = s.tenantID
	}
	applyProfitFields(item, in)
	calcProfit(item, dto.ProfitCalcByCost, nil)
	if in.ID > 0 {
		if err := s.repos.DB.Save(item).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.repos.DB.Create(item).Error; err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (s *MasterService) DeleteProfitTrials(ids []uint64) error {
	if len(ids) == 0 {
		return ErrBadRequest
	}
	res := s.db().Where("id IN ?", ids).Delete(&model.InvProfitTrial{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MasterService) CalcProfitTrials(in *dto.ProfitCalcRequest) ([]model.InvProfitTrial, error) {
	if len(in.IDs) == 0 {
		return nil, ErrBadRequest
	}
	mode := in.Mode
	if mode == "" {
		mode = dto.ProfitCalcByCost
	}
	var list []model.InvProfitTrial
	if err := s.db().Where("id IN ?", in.IDs).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		calcProfit(&list[i], mode, in.TargetMargin)
		if err := s.repos.DB.Save(&list[i]).Error; err != nil {
			return nil, err
		}
	}
	return list, nil
}

func applyProfitFields(item *model.InvProfitTrial, in *dto.ProfitTrialDTO) {
	item.ParentSKU = strings.TrimSpace(in.ParentSKU)
	item.SKU = strings.TrimSpace(in.SKU)
	item.ShopSKU = strings.TrimSpace(in.ShopSKU)
	item.ShopName = strings.TrimSpace(in.ShopName)
	item.SKUName = strings.TrimSpace(in.SKUName)
	item.RetailPrice = in.RetailPrice
	item.PriceUS = in.PriceUS
	item.Price = in.Price
	item.CostPrice = in.CostPrice
	item.PlatformFreight = in.PlatformFreight
	item.HeadFreight = in.HeadFreight
	item.Freight = in.Freight
	item.PackageFee = in.PackageFee
	item.Tariff = in.Tariff
	item.ProfitMargin = in.ProfitMargin
	item.ASIN = strings.TrimSpace(in.ASIN)
	item.Remark = strings.TrimSpace(in.Remark)
}

func calcProfit(item *model.InvProfitTrial, mode dto.ProfitCalcMode, targetMargin *float64) {
	fees := item.CostPrice + item.PlatformFreight + item.HeadFreight + item.Freight + item.PackageFee + item.Tariff
	if mode == dto.ProfitCalcByMargin {
		margin := item.ProfitMargin
		if targetMargin != nil {
			margin = *targetMargin
			item.ProfitMargin = margin
		}
		if margin >= 100 {
			return
		}
		denom := 1 - margin/100
		if denom <= 0 {
			return
		}
		item.Price = round4(fees / denom)
		item.Profit = round4(item.Price - fees)
		return
	}
	item.Profit = round4(item.Price - fees)
	if item.Price > 0 {
		item.ProfitMargin = round4(item.Profit / item.Price * 100)
	} else {
		item.ProfitMargin = 0
	}
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
