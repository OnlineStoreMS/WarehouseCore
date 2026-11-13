package service

import (
	"fmt"
	"strings"
	"time"

	"warehousecore/internal/integrations/peers"
	"warehousecore/internal/model"

	"gorm.io/gorm"
)

type PdaPickLine struct {
	SkuCode     string  `json:"skuCode"`
	ProductName string  `json:"productName"`
	SuggestCode string  `json:"suggestCode"`
	Qty         float64 `json:"qty"`
	PickedQty   float64 `json:"pickedQty"`
	CheckedQty  float64 `json:"checkedQty"`
}

type PdaPick struct {
	ID          uint64        `json:"id"`
	DocNo       string        `json:"docNo"`
	OrderNo     string        `json:"orderNo"`
	Source      string        `json:"source"`
	Buyer       string        `json:"buyer"`
	Address     string        `json:"address"`
	MailNo      string        `json:"mailNo"`
	PhotoURL    string        `json:"photoUrl"`
	Status      string        `json:"status"`
	StatusLabel string        `json:"statusLabel"`
	BlockReason string        `json:"blockReason"`
	Note        string        `json:"note"`
	Lines       []PdaPickLine `json:"lines"`
}

type PdaReturnLine struct {
	SkuCode      string  `json:"skuCode"`
	ProductName  string  `json:"productName"`
	ExpectedQty  float64 `json:"expectedQty"`
	GoodQty      float64 `json:"goodQty"`
	DefectQty    float64 `json:"defectQty"`
	PlacedGood   float64 `json:"placedGood"`
	PlacedDefect float64 `json:"placedDefect"`
	SuggestCode  string  `json:"suggestCode"`
}

type PdaReturn struct {
	ID          uint64          `json:"id"`
	DocNo       string          `json:"docNo"`
	LogisticsNo string          `json:"logisticsNo"`
	AftersaleNo string          `json:"aftersaleNo"`
	OrderNo     string          `json:"orderNo"`
	Status      string          `json:"status"`
	StatusLabel string          `json:"statusLabel"`
	Lines       []PdaReturnLine `json:"lines"`
}

type PdaCountLine struct {
	SkuCode  string  `json:"skuCode"`
	Name     string  `json:"productName"`
	CountQty float64 `json:"countQty"`
	BookQty  float64 `json:"bookQty,omitempty"`
	DiffQty  float64 `json:"diffQty,omitempty"`
}

type PdaCount struct {
	ID           uint64         `json:"id"`
	DocNo        string         `json:"docNo"`
	LocationCode string         `json:"locationCode"`
	Status       string         `json:"status"`
	StatusLabel  string         `json:"statusLabel"`
	Blind        bool           `json:"blind"`
	Lines        []PdaCountLine `json:"lines"`
}

type PdaTransferLine struct {
	ID          uint64  `json:"id"`
	SkuCode     string  `json:"skuCode"`
	Qty         float64 `json:"qty"`
	PickedQty   float64 `json:"pickedQty"`
	ReceivedQty float64 `json:"receivedQty"`
	FromCode    string  `json:"fromCode"`
	ToCode      string  `json:"toCode"`
}

type PdaTransfer struct {
	ID         uint64            `json:"id"`
	DocNo      string            `json:"docNo"`
	Status     string            `json:"status"`
	StatusLabel string           `json:"statusLabel"`
	FromName   string            `json:"fromName"`
	ToName     string            `json:"toName"`
	Lines      []PdaTransferLine `json:"lines"`
}

func (s *FlowService) lookupLocalCycle(code string) (*PdaHit, error) {
	var pick model.PickOrder
	err := s.repos.DB.Where("tenant_id = ? AND status <> ? AND (order_no = ? OR source_no = ? OR mail_no = ?)", s.tenantID, "done", code, code, code).
		Order("id DESC").First(&pick).Error
	if err == nil {
		view, e := s.viewPick(pick.ID)
		return &PdaHit{Kind: "pick", Code: code, Pick: view}, e
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	var ret model.ReturnReceipt
	err = s.repos.DB.Where("tenant_id = ? AND status <> ? AND (logistics_no = ? OR order_no = ? OR aftersale_no = ?)", s.tenantID, "done", code, code, code).
		Order("id DESC").First(&ret).Error
	if err == nil {
		view, e := s.viewReturn(ret.ID)
		return &PdaHit{Kind: "return", Code: code, Return: view}, e
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	var transfer model.TransferOrder
	err = s.repos.DB.Where("tenant_id = ? AND doc_no = ? AND status NOT IN ?", s.tenantID, code, []string{model.DocStatusReceived, model.DocStatusCancelled}).
		First(&transfer).Error
	if err == nil {
		view, e := s.viewTransfer(transfer.ID)
		return &PdaHit{Kind: "transfer", Code: code, Transfer: view}, e
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	var count model.StocktakeOrder
	err = s.repos.DB.Where("tenant_id = ? AND doc_no = ? AND status IN ?", s.tenantID, code, []string{model.DocStatusDraft, model.DocStatusCounting, model.DocStatusReview}).
		First(&count).Error
	if err == nil {
		view, e := s.viewCount(count.ID)
		return &PdaHit{Kind: "count", Code: code, Count: view}, e
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return nil, nil
}

func (s *FlowService) createPick(order *peers.Fulfillment, warehouseID uint64) (*PdaHit, error) {
	var existing model.PickOrder
	err := s.repos.DB.Where("tenant_id = ? AND order_no = ? AND status <> ?", s.tenantID, order.OrderNo, "done").First(&existing).Error
	if err == nil {
		view, e := s.viewPick(existing.ID)
		return &PdaHit{Kind: "pick", Code: order.OrderNo, Pick: view}, e
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	wh, err := s.pickWarehouse(warehouseID)
	if err != nil {
		return nil, err
	}
	row := model.PickOrder{
		TenantID: s.tenantID, WarehouseID: wh, DocNo: newDocNo("PICK"),
		Source: order.Source, SourceNo: order.SourceNo, OrderNo: order.OrderNo,
		Buyer: order.Buyer, Address: order.Address, Status: "picking",
	}
	var reason []string
	if order.Cancelled {
		reason = append(reason, "订单已取消或已发货")
	}
	if order.Refunded {
		reason = append(reason, "订单已退款")
	}
	if strings.TrimSpace(order.Address) == "" {
		reason = append(reason, "收货地址不完整")
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(&row).Error; e != nil {
			return e
		}
		for _, line := range order.Lines {
			sku, _ := s.findSku(line.SkuCode)
			item := model.PickLine{
				TenantID: s.tenantID, PickID: row.ID, SkuCode: line.SkuCode, ProductName: line.Name, Qty: line.Qty,
			}
			if sku != nil {
				item.InvSkuID = sku.ID
				code, qty, e := s.bestBin(tx, wh, sku.ID)
				if e != nil {
					return e
				}
				item.SuggestCode = code
				if qty+0.0001 < line.Qty {
					reason = append(reason, line.SkuCode+" 可用库存不足")
				}
			} else {
				reason = append(reason, line.SkuCode+" 不在商品资料里")
			}
			if e := tx.Create(&item).Error; e != nil {
				return e
			}
		}
		if len(reason) > 0 {
			return tx.Model(&row).Updates(map[string]interface{}{"status": "blocked", "block_reason": strings.Join(reason, "；")}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	view, err := s.viewPick(row.ID)
	if err != nil {
		return nil, err
	}
	msg := "已生成拣货任务"
	if view.Status == "blocked" {
		msg = "审单未通过，未进入拣货"
	}
	return &PdaHit{Kind: "pick", Code: order.OrderNo, Message: msg, Pick: view}, nil
}

func (s *FlowService) ScanPick(pickID uint64, locationCode, barcode string, qty float64) (*PdaHit, error) {
	if qty <= 0 {
		qty = 1
	}
	order, err := s.mustPick(pickID)
	if err != nil {
		return nil, err
	}
	if order.Status == "blocked" {
		return nil, fmt.Errorf("%w: 审单未通过，%s", ErrBadRequest, order.BlockReason)
	}
	if order.Status != "picking" {
		return nil, fmt.Errorf("%w: 当前不是拣货", ErrInvalidStatus)
	}
	sku, err := s.findSku(barcode)
	if err != nil || sku == nil {
		return nil, fmt.Errorf("%w: 商品不在资料里", ErrBadRequest)
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var lines []model.PickLine
		if e := tx.Where("tenant_id = ? AND pick_id = ?", s.tenantID, order.ID).Find(&lines).Error; e != nil {
			return e
		}
		var line *model.PickLine
		for i := range lines {
			if lines[i].InvSkuID == sku.ID || sameCode(lines[i].SkuCode, barcode) {
				line = &lines[i]
				break
			}
		}
		if line == nil {
			return fmt.Errorf("%w: 商品不在这张拣货单上", ErrBadRequest)
		}
		if line.SuggestCode != "" && !sameCode(line.SuggestCode, locationCode) {
			return fmt.Errorf("%w: 请扫推荐库位 %s", ErrBadRequest, line.SuggestCode)
		}
		loc, e := s.mustUsableLocation(tx, order.WarehouseID, locationCode)
		if e != nil {
			return e
		}
		remain := line.Qty - line.PickedQty
		if qty > remain {
			return fmt.Errorf("%w: 数量超过应拣", ErrBadRequest)
		}
		engine := NewStockEngine(s.repos.DB, s.tenantID)
		if e := engine.ApplyMoves(tx, []MoveLine{{
			WarehouseID: order.WarehouseID, LocationID: loc.ID, InvSkuID: sku.ID, Qty: -qty,
			MoveType: model.MoveSaleOut, DocType: "pick", DocNo: order.DocNo, DocID: order.ID,
		}}); e != nil {
			return e
		}
		line.PickedQty += qty
		if e := tx.Save(line).Error; e != nil {
			return e
		}
		var left int64
		if e := tx.Model(&model.PickLine{}).Where("tenant_id = ? AND pick_id = ? AND picked_qty + 0.0001 < qty", s.tenantID, order.ID).Count(&left).Error; e != nil {
			return e
		}
		if left == 0 {
			return tx.Model(order).Update("status", "reviewing").Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	view, err := s.viewPick(order.ID)
	return &PdaHit{Kind: "pick", Pick: view, Message: "已拣货扣账"}, err
}

func (s *FlowService) ReviewPick(pickID uint64, barcode string) (*PdaHit, error) {
	order, err := s.mustPick(pickID)
	if err != nil {
		return nil, err
	}
	if order.Status != "reviewing" {
		return nil, fmt.Errorf("%w: 拣完后才能复核", ErrInvalidStatus)
	}
	sku, _ := s.findSku(barcode)
	var matched bool
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var lines []model.PickLine
		if e := tx.Where("tenant_id = ? AND pick_id = ?", s.tenantID, order.ID).Find(&lines).Error; e != nil {
			return e
		}
		for i := range lines {
			if sameCode(lines[i].SkuCode, barcode) || (sku != nil && lines[i].InvSkuID == sku.ID) {
				if lines[i].CheckedQty+1 > lines[i].PickedQty+0.0001 {
					return fmt.Errorf("%w: 复核数量超过已拣", ErrBadRequest)
				}
				lines[i].CheckedQty++
				matched = true
				if e := tx.Save(&lines[i]).Error; e != nil {
					return e
				}
				break
			}
		}
		if !matched {
			return tx.Model(order).Updates(map[string]interface{}{"status": "picking", "note": "复核不一致：" + barcode}).Error
		}
		var left int64
		if e := tx.Model(&model.PickLine{}).Where("tenant_id = ? AND pick_id = ? AND checked_qty + 0.0001 < qty", s.tenantID, order.ID).Count(&left).Error; e != nil {
			return e
		}
		if left == 0 {
			return tx.Model(order).Update("status", "packed").Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	view, err := s.viewPick(order.ID)
	msg := "复核通过一件"
	if !matched {
		msg = "复核不一致，已退回拣货"
	}
	if view != nil && view.Status == "packed" {
		msg = "复核通过，可以交接"
	}
	return &PdaHit{Kind: "pick", Pick: view, Message: msg}, err
}

func (s *FlowService) HandoverPick(pickID uint64, mailNo, photoURL string) (*PdaHit, error) {
	mailNo = strings.TrimSpace(mailNo)
	if mailNo == "" {
		return nil, fmt.Errorf("%w: 请扫快递单", ErrBadRequest)
	}
	order, err := s.mustPick(pickID)
	if err != nil {
		return nil, err
	}
	if order.Status != "packed" {
		return nil, fmt.Errorf("%w: 复核通过后才能交接", ErrInvalidStatus)
	}
	if err := s.repos.DB.Model(order).Updates(map[string]interface{}{
		"status": "done", "mail_no": mailNo, "photo_url": strings.TrimSpace(photoURL),
	}).Error; err != nil {
		return nil, err
	}
	view, err := s.viewPick(order.ID)
	return &PdaHit{Kind: "pick", Pick: view, Message: "已记录出库交接"}, err
}

func (s *FlowService) RecheckPick(pickID uint64) (*PdaHit, error) {
	order, err := s.mustPick(pickID)
	if err != nil {
		return nil, err
	}
	if strings.Contains(order.BlockReason, "取消") || strings.Contains(order.BlockReason, "退款") || strings.Contains(order.BlockReason, "已发货") {
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, order.BlockReason)
	}
	var lines []model.PickLine
	if err := s.repos.DB.Where("tenant_id = ? AND pick_id = ?", s.tenantID, order.ID).Find(&lines).Error; err != nil {
		return nil, err
	}
	var reason []string
	if strings.TrimSpace(order.Address) == "" {
		reason = append(reason, "收货地址不完整")
	}
	for _, line := range lines {
		if line.InvSkuID == 0 {
			reason = append(reason, line.SkuCode+" 不在商品资料里")
			continue
		}
		_, qty, err := s.bestBin(s.repos.DB, order.WarehouseID, line.InvSkuID)
		if err != nil {
			return nil, err
		}
		if qty+0.0001 < line.Qty-line.PickedQty {
			reason = append(reason, line.SkuCode+" 可用库存不足")
		}
	}
	status := "picking"
	text := ""
	if len(reason) > 0 {
		status = "blocked"
		text = strings.Join(reason, "；")
	}
	if err := s.repos.DB.Model(order).Updates(map[string]interface{}{"status": status, "block_reason": text}).Error; err != nil {
		return nil, err
	}
	view, err := s.viewPick(order.ID)
	return &PdaHit{Kind: "pick", Pick: view, Message: "已重新审单"}, err
}

func (s *FlowService) createReturn(notice *peers.ReturnNotice, warehouseID uint64) (*PdaHit, error) {
	key := notice.LogisticsNo
	if key == "" {
		key = notice.AftersaleNo
	}
	var existing model.ReturnReceipt
	err := s.repos.DB.Where("tenant_id = ? AND status <> ? AND (logistics_no = ? OR aftersale_no = ?)", s.tenantID, "done", notice.LogisticsNo, notice.AftersaleNo).
		First(&existing).Error
	if err == nil {
		view, e := s.viewReturn(existing.ID)
		return &PdaHit{Kind: "return", Code: key, Return: view}, e
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	wh, err := s.pickWarehouse(warehouseID)
	if err != nil {
		return nil, err
	}
	row := model.ReturnReceipt{
		TenantID: s.tenantID, WarehouseID: wh, DocNo: newDocNo("RET"),
		LogisticsNo: notice.LogisticsNo, AftersaleNo: notice.AftersaleNo, OrderNo: notice.OrderNo, Status: "open",
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(&row).Error; e != nil {
			return e
		}
		for _, line := range notice.Lines {
			item := model.ReturnLine{
				TenantID: s.tenantID, ReturnID: row.ID, SkuCode: line.SkuCode, ProductName: line.Name, ExpectedQty: line.Qty,
			}
			if sku, e := s.findSku(line.SkuCode); e == nil && sku != nil {
				item.InvSkuID = sku.ID
				code, _, e := s.bestBin(tx, wh, sku.ID)
				if e != nil {
					return e
				}
				item.SuggestCode = code
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
	view, err := s.viewReturn(row.ID)
	return &PdaHit{Kind: "return", Code: key, Message: "已按售后单生成退货入库", Return: view}, err
}

func (s *FlowService) GradeReturn(returnID uint64, barcode, grade string) (*PdaHit, error) {
	if grade != "defect" {
		grade = "good"
	}
	order, err := s.mustReturn(returnID)
	if err != nil {
		return nil, err
	}
	if order.Status != "open" {
		return nil, fmt.Errorf("%w: 退货单已完成", ErrInvalidStatus)
	}
	sku, _ := s.findSku(barcode)
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var lines []model.ReturnLine
		if e := tx.Where("tenant_id = ? AND return_id = ?", s.tenantID, order.ID).Find(&lines).Error; e != nil {
			return e
		}
		for i := range lines {
			if !sameCode(lines[i].SkuCode, barcode) && (sku == nil || lines[i].InvSkuID != sku.ID) {
				continue
			}
			if lines[i].GoodQty+lines[i].DefectQty+1 > lines[i].ExpectedQty+0.0001 {
				return fmt.Errorf("%w: 超过退货数量", ErrBadRequest)
			}
			if grade == "defect" {
				lines[i].DefectQty++
			} else {
				lines[i].GoodQty++
			}
			return tx.Save(&lines[i]).Error
		}
		return fmt.Errorf("%w: 商品不在这张退货单上", ErrBadRequest)
	})
	if err != nil {
		return nil, err
	}
	view, err := s.viewReturn(order.ID)
	return &PdaHit{Kind: "return", Return: view}, err
}

func (s *FlowService) PlaceReturn(returnID uint64, locationCode, barcode, grade string, qty float64) (*PdaHit, error) {
	if qty <= 0 {
		qty = 1
	}
	if grade != "defect" {
		grade = "good"
	}
	order, err := s.mustReturn(returnID)
	if err != nil {
		return nil, err
	}
	sku, err := s.findSku(barcode)
	if err != nil || sku == nil {
		return nil, fmt.Errorf("%w: 商品不在资料里", ErrBadRequest)
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var lines []model.ReturnLine
		if e := tx.Where("tenant_id = ? AND return_id = ?", s.tenantID, order.ID).Find(&lines).Error; e != nil {
			return e
		}
		var line *model.ReturnLine
		for i := range lines {
			if lines[i].InvSkuID == sku.ID || sameCode(lines[i].SkuCode, barcode) {
				line = &lines[i]
				break
			}
		}
		if line == nil {
			return fmt.Errorf("%w: 商品不在这张退货单上", ErrBadRequest)
		}
		var loc *model.WarehouseLocation
		if grade == "defect" {
			id, e := s.ensureIsolate(tx, order.WarehouseID)
			if e != nil {
				return e
			}
			var row model.WarehouseLocation
			if e := tx.First(&row, id).Error; e != nil {
				return e
			}
			if !sameCode(row.Code, locationCode) {
				return fmt.Errorf("%w: 残次品请扫隔离库位 %s", ErrBadRequest, row.Code)
			}
			loc = &row
			if line.PlacedDefect+qty > line.DefectQty+0.0001 {
				return fmt.Errorf("%w: 超过已判定残次数量", ErrBadRequest)
			}
			line.PlacedDefect += qty
		} else {
			row, e := s.mustUsableLocation(tx, order.WarehouseID, locationCode)
			if e != nil {
				return e
			}
			if row.Code == model.IsolateLocationCode || (line.SuggestCode != "" && !sameCode(line.SuggestCode, row.Code)) {
				want := line.SuggestCode
				if want == "" {
					want = "可用库位"
				}
				return fmt.Errorf("%w: 良品请扫推荐库位 %s", ErrBadRequest, want)
			}
			loc = row
			if line.PlacedGood+qty > line.GoodQty+0.0001 {
				return fmt.Errorf("%w: 超过已判定良品数量", ErrBadRequest)
			}
			line.PlacedGood += qty
		}
		engine := NewStockEngine(s.repos.DB, s.tenantID)
		if e := engine.ApplyMoves(tx, []MoveLine{{
			WarehouseID: order.WarehouseID, LocationID: loc.ID, InvSkuID: sku.ID, Qty: qty,
			MoveType: model.MoveOtherIn, DocType: "return", DocNo: order.DocNo, DocID: order.ID,
		}}); e != nil {
			return e
		}
		if e := tx.Save(line).Error; e != nil {
			return e
		}
		var left int64
		if e := tx.Model(&model.ReturnLine{}).Where("tenant_id = ? AND return_id = ? AND (placed_good + 0.0001 < good_qty OR placed_defect + 0.0001 < defect_qty OR good_qty + defect_qty + 0.0001 < expected_qty)", s.tenantID, order.ID).Count(&left).Error; e != nil {
			return e
		}
		if left == 0 {
			return tx.Model(order).Update("status", "done").Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	view, err := s.viewReturn(order.ID)
	return &PdaHit{Kind: "return", Return: view, Message: "退货已入账"}, err
}

func (s *FlowService) StartCount(locationCode string) (*PdaHit, error) {
	locationCode = strings.TrimSpace(locationCode)
	var loc model.WarehouseLocation
	if err := s.repos.DB.Where("tenant_id = ? AND code = ?", s.tenantID, locationCode).First(&loc).Error; err != nil {
		return nil, fmt.Errorf("%w: 库位不存在", ErrBadRequest)
	}
	if loc.Code == model.StageLocationCode {
		return nil, fmt.Errorf("%w: 收货暂存位不参与盘点", ErrBadRequest)
	}
	var existing model.StocktakeOrder
	err := s.repos.DB.Where("tenant_id = ? AND location_id = ? AND status = ?", s.tenantID, loc.ID, model.DocStatusCounting).First(&existing).Error
	if err == nil {
		view, e := s.viewCount(existing.ID)
		return &PdaHit{Kind: "count", Count: view}, e
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	order := model.StocktakeOrder{
		TenantID: s.tenantID, DocNo: newDocNo("CNT"), WarehouseID: loc.WarehouseID, LocationID: loc.ID,
		Status: model.DocStatusCounting, Remark: "盲盘",
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&loc).Update("status", model.LocStatusLocked).Error; e != nil {
			return e
		}
		return tx.Create(&order).Error
	})
	if err != nil {
		return nil, err
	}
	view, err := s.viewCount(order.ID)
	return &PdaHit{Kind: "count", Count: view, Message: "库位已锁定，开始盲盘"}, err
}

func (s *FlowService) ScanCount(countID uint64, barcode string, qty float64) (*PdaHit, error) {
	if qty <= 0 {
		qty = 1
	}
	order, err := s.mustCount(countID)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusCounting {
		return nil, fmt.Errorf("%w: 盘点已结束", ErrInvalidStatus)
	}
	sku, err := s.findSku(barcode)
	if err != nil || sku == nil {
		return nil, fmt.Errorf("%w: 商品不在资料里", ErrBadRequest)
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var item model.StocktakeItem
		e := tx.Where("tenant_id = ? AND order_id = ? AND inv_sku_id = ?", s.tenantID, order.ID, sku.ID).First(&item).Error
		if e == gorm.ErrRecordNotFound {
			item = model.StocktakeItem{
				TenantID: s.tenantID, OrderID: order.ID, LocationID: order.LocationID, InvSkuID: sku.ID, CountQty: qty,
			}
			return tx.Create(&item).Error
		}
		if e != nil {
			return e
		}
		item.CountQty += qty
		return tx.Save(&item).Error
	})
	if err != nil {
		return nil, err
	}
	view, err := s.viewCount(order.ID)
	return &PdaHit{Kind: "count", Count: view}, err
}

func (s *FlowService) SubmitCount(countID uint64) (*PdaHit, error) {
	order, err := s.mustCount(countID)
	if err != nil {
		return nil, err
	}
	if order.Status != model.DocStatusCounting {
		return nil, fmt.Errorf("%w: 盘点已结束", ErrInvalidStatus)
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var items []model.StocktakeItem
		if e := tx.Where("tenant_id = ? AND order_id = ?", s.tenantID, order.ID).Find(&items).Error; e != nil {
			return e
		}
		if len(items) == 0 {
			return fmt.Errorf("%w: 还没有实盘", ErrBadRequest)
		}
		seen := map[uint64]bool{}
		for i := range items {
			var bal model.InvBalance
			e := tx.Where("tenant_id = ? AND location_id = ? AND inv_sku_id = ?", s.tenantID, order.LocationID, items[i].InvSkuID).First(&bal).Error
			book := 0.0
			if e == nil {
				book = bal.OnHand
			} else if e != gorm.ErrRecordNotFound {
				return e
			}
			items[i].BookQty = book
			items[i].DiffQty = items[i].CountQty - book
			items[i].Remark = "盘点差异"
			if e := tx.Save(&items[i]).Error; e != nil {
				return e
			}
			seen[items[i].InvSkuID] = true
		}
		var bals []model.InvBalance
		if e := tx.Where("tenant_id = ? AND location_id = ? AND on_hand <> 0", s.tenantID, order.LocationID).Find(&bals).Error; e != nil {
			return e
		}
		for _, bal := range bals {
			if seen[bal.InvSkuID] {
				continue
			}
			row := model.StocktakeItem{
				TenantID: s.tenantID, OrderID: order.ID, LocationID: order.LocationID, InvSkuID: bal.InvSkuID,
				BookQty: bal.OnHand, CountQty: 0, DiffQty: -bal.OnHand, Remark: "盘点差异",
			}
			if e := tx.Create(&row).Error; e != nil {
				return e
			}
		}
		if e := tx.Model(&model.WarehouseLocation{}).Where("id = ?", order.LocationID).Update("status", model.LocStatusOn).Error; e != nil {
			return e
		}
		return tx.Model(order).Update("status", model.DocStatusReview).Error
	})
	if err != nil {
		return nil, err
	}
	view, err := s.viewCount(order.ID)
	return &PdaHit{Kind: "count", Count: view, Message: "已生成差异单，库存未改"}, err
}

func (s *FlowService) ScanTransfer(transferID uint64, locationCode, barcode string, inbound bool, qty float64) (*PdaHit, error) {
	if qty <= 0 {
		qty = 1
	}
	order, err := s.mustTransfer(transferID)
	if err != nil {
		return nil, err
	}
	want := model.DocStatusDraft
	if inbound {
		want = model.DocStatusInTransit
	}
	if order.Status != want {
		return nil, fmt.Errorf("%w: 调拨单当前不能这样扫", ErrInvalidStatus)
	}
	sku, err := s.findSku(barcode)
	if err != nil || sku == nil {
		return nil, fmt.Errorf("%w: 商品不在资料里", ErrBadRequest)
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var items []model.TransferItem
		if e := tx.Where("tenant_id = ? AND order_id = ?", s.tenantID, order.ID).Find(&items).Error; e != nil {
			return e
		}
		var line *model.TransferItem
		for i := range items {
			if items[i].InvSkuID == sku.ID {
				line = &items[i]
				break
			}
		}
		if line == nil {
			return fmt.Errorf("%w: 商品不在这张调拨单上", ErrBadRequest)
		}
		whID := order.FromWarehouseID
		if inbound {
			whID = order.ToWarehouseID
		}
		loc, e := s.mustUsableLocation(tx, whID, locationCode)
		if e != nil {
			return e
		}
		if !inbound {
			if order.FromLocationID > 0 && loc.ID != order.FromLocationID {
				return fmt.Errorf("%w: 请扫调出库位", ErrBadRequest)
			}
			if line.SourceLocationID > 0 && loc.ID != line.SourceLocationID {
				return fmt.Errorf("%w: 请扫原来的调出库位", ErrBadRequest)
			}
			if line.PickedQty+qty > line.Qty+0.0001 {
				return fmt.Errorf("%w: 超过调出数量", ErrBadRequest)
			}
			line.SourceLocationID = loc.ID
			line.PickedQty += qty
		} else {
			if order.ToLocationID > 0 && loc.ID != order.ToLocationID {
				return fmt.Errorf("%w: 请扫调入库位", ErrBadRequest)
			}
			if line.ToLocationID > 0 && loc.ID != line.ToLocationID {
				return fmt.Errorf("%w: 请扫原来的调入库位", ErrBadRequest)
			}
			if line.ReceivedQty+qty > line.Qty+0.0001 {
				return fmt.Errorf("%w: 超过调入数量", ErrBadRequest)
			}
			line.ToLocationID = loc.ID
			line.ReceivedQty += qty
		}
		return tx.Save(line).Error
	})
	if err != nil {
		return nil, err
	}
	view, err := s.viewTransfer(order.ID)
	return &PdaHit{Kind: "transfer", Transfer: view}, err
}

func (s *FlowService) ConfirmTransfer(transferID uint64, inbound bool) (*PdaHit, error) {
	order, err := s.mustTransfer(transferID)
	if err != nil {
		return nil, err
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var items []model.TransferItem
		if e := tx.Where("tenant_id = ? AND order_id = ?", s.tenantID, order.ID).Find(&items).Error; e != nil {
			return e
		}
		engine := NewStockEngine(s.repos.DB, s.tenantID)
		var lines []MoveLine
		if !inbound {
			if order.Status != model.DocStatusDraft {
				return fmt.Errorf("%w: 调拨单已出库", ErrInvalidStatus)
			}
			for _, it := range items {
				if it.PickedQty+0.0001 < it.Qty || it.SourceLocationID == 0 {
					return fmt.Errorf("%w: 还有商品没扫调出库位", ErrBadRequest)
				}
				lines = append(lines, MoveLine{
					WarehouseID: order.FromWarehouseID, LocationID: it.SourceLocationID, InvSkuID: it.InvSkuID, Qty: -it.Qty,
					MoveType: model.MoveTransferOut, DocType: "transfer", DocNo: order.DocNo, DocID: order.ID,
				})
			}
			if e := engine.ApplyMoves(tx, lines); e != nil {
				return e
			}
			return tx.Model(order).Update("status", model.DocStatusInTransit).Error
		}
		if order.Status != model.DocStatusInTransit {
			return fmt.Errorf("%w: 调拨单还没在途", ErrInvalidStatus)
		}
		for _, it := range items {
			if it.ReceivedQty+0.0001 < it.Qty || it.ToLocationID == 0 {
				return fmt.Errorf("%w: 还有商品没扫调入库位", ErrBadRequest)
			}
			lines = append(lines, MoveLine{
				WarehouseID: order.ToWarehouseID, LocationID: it.ToLocationID, InvSkuID: it.InvSkuID, Qty: it.Qty,
				MoveType: model.MoveTransferIn, DocType: "transfer", DocNo: order.DocNo, DocID: order.ID,
			})
		}
		if e := engine.ApplyMoves(tx, lines); e != nil {
			return e
		}
		return tx.Model(order).Update("status", model.DocStatusReceived).Error
	})
	if err != nil {
		return nil, err
	}
	view, err := s.viewTransfer(order.ID)
	msg := "调拨已出库，库存在途"
	if inbound {
		msg = "调拨已入到库位"
	}
	return &PdaHit{Kind: "transfer", Transfer: view, Message: msg}, err
}

func (s *FlowService) bestBin(tx *gorm.DB, warehouseID, skuID uint64) (string, float64, error) {
	type row struct {
		Code   string
		OnHand float64
	}
	var rows []row
	err := tx.Table("inv_balances AS b").
		Select("l.code, b.on_hand").
		Joins("JOIN warehouse_locations l ON l.id = b.location_id").
		Where("b.tenant_id = ? AND b.warehouse_id = ? AND b.inv_sku_id = ? AND l.status = ?", s.tenantID, warehouseID, skuID, model.LocStatusOn).
		Where("l.code NOT IN ?", []string{model.StageLocationCode, model.IsolateLocationCode, model.DefaultLocationCode}).
		Order("b.on_hand DESC, l.zone, l.shelf, l.level, l.bin").
		Scan(&rows).Error
	if err != nil {
		return "", 0, err
	}
	total := 0.0
	code := ""
	for _, row := range rows {
		total += row.OnHand
		if code == "" && row.OnHand > 0 {
			code = row.Code
		}
	}
	if code == "" {
		var loc model.WarehouseLocation
		e := tx.Where("tenant_id = ? AND warehouse_id = ? AND status = ? AND code NOT IN ?", s.tenantID, warehouseID, model.LocStatusOn, []string{model.StageLocationCode, model.IsolateLocationCode, model.DefaultLocationCode}).
			Order("zone, shelf, level, bin, code").First(&loc).Error
		if e == nil {
			code = loc.Code
		}
	}
	return code, total, nil
}

func (s *FlowService) mustUsableLocation(tx *gorm.DB, warehouseID uint64, code string) (*model.WarehouseLocation, error) {
	var loc model.WarehouseLocation
	err := tx.Where("tenant_id = ? AND warehouse_id = ? AND code = ?", s.tenantID, warehouseID, strings.TrimSpace(code)).First(&loc).Error
	if err != nil {
		return nil, fmt.Errorf("%w: 库位不存在或不在当前仓库", ErrBadRequest)
	}
	switch loc.Status {
	case model.LocStatusOff:
		return nil, fmt.Errorf("%w: 库位已禁用", ErrBadRequest)
	case model.LocStatusLocked:
		return nil, fmt.Errorf("%w: 库位已锁定", ErrBadRequest)
	case model.LocStatusFull:
		return nil, fmt.Errorf("%w: 库位已满", ErrBadRequest)
	case model.LocStatusOn:
		return &loc, nil
	default:
		return nil, fmt.Errorf("%w: 库位不可用", ErrBadRequest)
	}
}

func (s *FlowService) ensureIsolate(tx *gorm.DB, warehouseID uint64) (uint64, error) {
	var loc model.WarehouseLocation
	err := tx.Where("tenant_id = ? AND warehouse_id = ? AND code = ?", s.tenantID, warehouseID, model.IsolateLocationCode).First(&loc).Error
	if err == nil {
		return loc.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	loc = model.WarehouseLocation{
		TenantID: s.tenantID, WarehouseID: warehouseID, Code: model.IsolateLocationCode, Zone: "隔离", Status: model.LocStatusOn,
	}
	if err := tx.Create(&loc).Error; err != nil {
		return 0, err
	}
	return loc.ID, nil
}

func (s *FlowService) mustPick(id uint64) (*model.PickOrder, error) {
	var order model.PickOrder
	if err := s.repos.DB.Where("tenant_id = ? AND id = ?", s.tenantID, id).First(&order).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &order, nil
}

func (s *FlowService) mustReturn(id uint64) (*model.ReturnReceipt, error) {
	var order model.ReturnReceipt
	if err := s.repos.DB.Where("tenant_id = ? AND id = ?", s.tenantID, id).First(&order).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &order, nil
}

func (s *FlowService) mustCount(id uint64) (*model.StocktakeOrder, error) {
	var order model.StocktakeOrder
	if err := s.repos.DB.Where("tenant_id = ? AND id = ?", s.tenantID, id).First(&order).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &order, nil
}

func (s *FlowService) mustTransfer(id uint64) (*model.TransferOrder, error) {
	var order model.TransferOrder
	if err := s.repos.DB.Where("tenant_id = ? AND id = ?", s.tenantID, id).First(&order).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &order, nil
}

func (s *FlowService) viewPick(id uint64) (*PdaPick, error) {
	order, err := s.mustPick(id)
	if err != nil {
		return nil, err
	}
	var lines []model.PickLine
	if err := s.repos.DB.Where("tenant_id = ? AND pick_id = ?", s.tenantID, id).Order("id").Find(&lines).Error; err != nil {
		return nil, err
	}
	out := &PdaPick{
		ID: order.ID, DocNo: order.DocNo, OrderNo: order.OrderNo, Source: order.Source, Buyer: order.Buyer,
		Address: order.Address, MailNo: order.MailNo, PhotoURL: order.PhotoURL, Status: order.Status,
		StatusLabel: pickLabel(order.Status), BlockReason: order.BlockReason, Note: order.Note, Lines: []PdaPickLine{},
	}
	for _, line := range lines {
		out.Lines = append(out.Lines, PdaPickLine{
			SkuCode: line.SkuCode, ProductName: line.ProductName, SuggestCode: line.SuggestCode,
			Qty: line.Qty, PickedQty: line.PickedQty, CheckedQty: line.CheckedQty,
		})
	}
	return out, nil
}

func (s *FlowService) viewReturn(id uint64) (*PdaReturn, error) {
	order, err := s.mustReturn(id)
	if err != nil {
		return nil, err
	}
	var lines []model.ReturnLine
	if err := s.repos.DB.Where("tenant_id = ? AND return_id = ?", s.tenantID, id).Order("id").Find(&lines).Error; err != nil {
		return nil, err
	}
	out := &PdaReturn{
		ID: order.ID, DocNo: order.DocNo, LogisticsNo: order.LogisticsNo, AftersaleNo: order.AftersaleNo,
		OrderNo: order.OrderNo, Status: order.Status, StatusLabel: returnLabel(order.Status), Lines: []PdaReturnLine{},
	}
	for _, line := range lines {
		out.Lines = append(out.Lines, PdaReturnLine{
			SkuCode: line.SkuCode, ProductName: line.ProductName, ExpectedQty: line.ExpectedQty,
			GoodQty: line.GoodQty, DefectQty: line.DefectQty, PlacedGood: line.PlacedGood, PlacedDefect: line.PlacedDefect,
			SuggestCode: line.SuggestCode,
		})
	}
	return out, nil
}

func (s *FlowService) viewCount(id uint64) (*PdaCount, error) {
	order, err := s.mustCount(id)
	if err != nil {
		return nil, err
	}
	var loc model.WarehouseLocation
	_ = s.repos.DB.First(&loc, order.LocationID).Error
	var items []model.StocktakeItem
	if err := s.repos.DB.Where("tenant_id = ? AND order_id = ?", s.tenantID, id).Order("id").Find(&items).Error; err != nil {
		return nil, err
	}
	blind := order.Status == model.DocStatusCounting
	out := &PdaCount{
		ID: order.ID, DocNo: order.DocNo, LocationCode: loc.Code, Status: order.Status,
		StatusLabel: countLabel(order.Status), Blind: blind, Lines: []PdaCountLine{},
	}
	for _, item := range items {
		var sku model.InvSku
		_ = s.repos.DB.First(&sku, item.InvSkuID).Error
		line := PdaCountLine{SkuCode: sku.SkuCode, Name: sku.PickName, CountQty: item.CountQty}
		if !blind {
			line.BookQty = item.BookQty
			line.DiffQty = item.DiffQty
		}
		out.Lines = append(out.Lines, line)
	}
	return out, nil
}

func (s *FlowService) viewTransfer(id uint64) (*PdaTransfer, error) {
	order, err := s.mustTransfer(id)
	if err != nil {
		return nil, err
	}
	var items []model.TransferItem
	if err := s.repos.DB.Where("tenant_id = ? AND order_id = ?", s.tenantID, id).Order("id").Find(&items).Error; err != nil {
		return nil, err
	}
	fromName, toName := s.warehouseName(order.FromWarehouseID), s.warehouseName(order.ToWarehouseID)
	out := &PdaTransfer{
		ID: order.ID, DocNo: order.DocNo, Status: order.Status, StatusLabel: transferLabel(order.Status),
		FromName: fromName, ToName: toName, Lines: []PdaTransferLine{},
	}
	for _, item := range items {
		var sku model.InvSku
		_ = s.repos.DB.First(&sku, item.InvSkuID).Error
		out.Lines = append(out.Lines, PdaTransferLine{
			ID: item.ID, SkuCode: sku.SkuCode, Qty: item.Qty, PickedQty: item.PickedQty, ReceivedQty: item.ReceivedQty,
			FromCode: s.locationCode(item.SourceLocationID), ToCode: s.locationCode(item.ToLocationID),
		})
	}
	return out, nil
}

func (s *FlowService) warehouseName(id uint64) string {
	var wh model.Warehouse
	if s.repos.DB.First(&wh, id).Error != nil {
		return ""
	}
	return wh.Name
}

func (s *FlowService) locationCode(id uint64) string {
	if id == 0 {
		return ""
	}
	var loc model.WarehouseLocation
	if s.repos.DB.First(&loc, id).Error != nil {
		return ""
	}
	return loc.Code
}

func pickLabel(status string) string {
	switch status {
	case "blocked":
		return "审单拦截"
	case "picking":
		return "拣货中"
	case "reviewing":
		return "待复核"
	case "packed":
		return "待交接"
	case "done":
		return "已出库"
	default:
		return status
	}
}

func returnLabel(status string) string {
	if status == "done" {
		return "退货已入库"
	}
	return "退货验收"
}

func countLabel(status string) string {
	switch status {
	case model.DocStatusCounting:
		return "盲盘中"
	case model.DocStatusReview:
		return "差异待审"
	default:
		return status
	}
}

type TaskCard struct {
	DocNo     string    `json:"docNo"`
	Subtitle  string    `json:"subtitle"`
	Status    string    `json:"status"`
	QtyText   string    `json:"qtyText"`
	Extra     string    `json:"extra"`
	Remark    string    `json:"remark"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"createdAt"`
}

type TaskBoard struct {
	Receive int        `json:"receive"`
	Audit   int        `json:"audit"`
	Pick    int        `json:"pick"`
	Ship    int        `json:"ship"`
	List    []TaskCard `json:"list"`
}

func (s *FlowService) Tasks(bucket, keyword string) (*TaskBoard, error) {
	board := &TaskBoard{List: []TaskCard{}}
	var err error
	if board.Receive, err = s.countQuery(&model.ReceiptOrder{}, "status = ? AND po_no <> ''", "open"); err != nil {
		return nil, err
	}
	blocked, err := s.countQuery(&model.PickOrder{}, "status = ?", "blocked")
	if err != nil {
		return nil, err
	}
	review, err := s.countQuery(&model.StocktakeOrder{}, "status = ?", model.DocStatusReview)
	if err != nil {
		return nil, err
	}
	board.Audit = blocked + review
	if board.Pick, err = s.countQuery(&model.PickOrder{}, "status = ?", "picking"); err != nil {
		return nil, err
	}
	if board.Ship, err = s.countQuery(&model.PickOrder{}, "status = ?", "packed"); err != nil {
		return nil, err
	}
	switch bucket {
	case "receive":
		board.List, err = s.listReceiveTasks(keyword)
	case "audit":
		board.List, err = s.listAuditTasks(keyword)
	case "pick":
		board.List, err = s.listPickTasks(keyword, "picking")
	case "ship":
		board.List, err = s.listPickTasks(keyword, "packed")
	}
	return board, err
}

func (s *FlowService) countQuery(model interface{}, query string, args ...interface{}) (int, error) {
	var n int64
	err := s.repos.DB.Model(model).Where("tenant_id = ?", s.tenantID).Where(query, args...).Count(&n).Error
	return int(n), err
}

func (s *FlowService) listReceiveTasks(keyword string) ([]TaskCard, error) {
	var rows []model.ReceiptOrder
	q := s.repos.DB.Where("tenant_id = ? AND status = ? AND po_no <> ''", s.tenantID, "open")
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("doc_no ILIKE ? OR logistics_no ILIKE ? OR po_no ILIKE ? OR supplier_name ILIKE ?", like, like, like, like)
	}
	if err := q.Order("id DESC").Limit(30).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]TaskCard, 0, len(rows))
	for _, row := range rows {
		out = append(out, TaskCard{
			DocNo: row.DocNo, Subtitle: row.SupplierName, Status: "open",
			QtyText: "采购 " + row.PoNo, Extra: row.LogisticsNo, Remark: "采购到货",
			Code: row.LogisticsNo, CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func (s *FlowService) listPickTasks(keyword, status string) ([]TaskCard, error) {
	var rows []model.PickOrder
	q := s.repos.DB.Where("tenant_id = ? AND status = ?", s.tenantID, status)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("doc_no ILIKE ? OR order_no ILIKE ? OR buyer ILIKE ?", like, like, like)
	}
	if err := q.Order("id DESC").Limit(30).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]TaskCard, 0, len(rows))
	for _, row := range rows {
		out = append(out, TaskCard{
			DocNo: row.DocNo, Subtitle: row.Buyer, Status: row.Status,
			QtyText: "订单 " + row.OrderNo, Extra: row.Address, Remark: row.BlockReason,
			Code: row.OrderNo, CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func (s *FlowService) listAuditTasks(keyword string) ([]TaskCard, error) {
	picks, err := s.listPickTasks(keyword, "blocked")
	if err != nil {
		return nil, err
	}
	var rows []model.StocktakeOrder
	q := s.repos.DB.Where("tenant_id = ? AND status = ?", s.tenantID, model.DocStatusReview)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		q = q.Where("doc_no ILIKE ?", "%"+keyword+"%")
	}
	if err := q.Order("id DESC").Limit(30).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		code := ""
		var loc model.WarehouseLocation
		if s.repos.DB.First(&loc, row.LocationID).Error == nil {
			code = loc.Code
		}
		picks = append(picks, TaskCard{
			DocNo: row.DocNo, Subtitle: code, Status: row.Status,
			QtyText: "盘点差异", Extra: row.Remark, Remark: "待审核后才调库存",
			Code: row.DocNo, CreatedAt: row.CreatedAt,
		})
	}
	return picks, nil
}

func transferLabel(status string) string {
	switch status {
	case model.DocStatusDraft:
		return "待调出"
	case model.DocStatusInTransit:
		return "在途"
	case model.DocStatusReceived:
		return "已调入"
	default:
		return status
	}
}
