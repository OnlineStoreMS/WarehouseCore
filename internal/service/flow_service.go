package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"warehousecore/internal/integrations/peers"
	"warehousecore/internal/integrations/supplycore"
	"warehousecore/internal/model"
	"warehousecore/internal/repo"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FlowService struct {
	repos    *repo.Repos
	sc       *supplycore.Client
	peers    *peers.Client
	tenantID uint64
}

func NewFlowService(repos *repo.Repos, sc *supplycore.Client, peersClient *peers.Client) *FlowService {
	return &FlowService{repos: repos, sc: sc, peers: peersClient}
}

func (s *FlowService) ForTenant(tenantID uint64) *FlowService {
	return &FlowService{repos: s.repos, sc: s.sc, peers: s.peers, tenantID: repo.NormalizeTenantID(tenantID)}
}

type PdaLine struct {
	SkuCode     string  `json:"skuCode"`
	ProductName string  `json:"productName"`
	BatchNo     string  `json:"batchNo"`
	Expiry      string  `json:"expiry"`
	ExpectedQty float64 `json:"expectedQty"`
	ReceivedQty float64 `json:"receivedQty"`
}

type PdaVariance struct {
	SkuCode string  `json:"skuCode"`
	Reason  string  `json:"reason"`
	Qty     float64 `json:"qty"`
	Note    string  `json:"note"`
}

type PdaTask struct {
	ID          uint64  `json:"id"`
	SkuCode     string  `json:"skuCode"`
	BatchNo     string  `json:"batchNo"`
	Expiry      string  `json:"expiry"`
	SuggestCode string  `json:"suggestCode"`
	Qty         float64 `json:"qty"`
	DoneQty     float64 `json:"doneQty"`
	Status      string  `json:"status"`
}

type PdaReceipt struct {
	ID           uint64         `json:"id"`
	DocNo        string         `json:"docNo"`
	LogisticsNo  string         `json:"logisticsNo"`
	PoNo         string         `json:"poNo"`
	SupplierName string         `json:"supplierName"`
	Status       string         `json:"status"`
	StatusLabel  string         `json:"statusLabel"`
	Lines        []PdaLine      `json:"lines"`
	Variances    []PdaVariance  `json:"variances"`
	Tasks        []PdaTask      `json:"tasks"`
}

type PdaStock struct {
	SkuCode      string  `json:"skuCode"`
	ProductName  string  `json:"productName"`
	LocationCode string  `json:"locationCode"`
	OnHand       float64 `json:"onHand"`
	PendingQty   float64 `json:"pendingQty"`
	BatchNo      string  `json:"batchNo"`
	Expiry       string  `json:"expiry"`
}

type PdaHit struct {
	Kind     string       `json:"kind"`
	Code     string       `json:"code"`
	Message  string       `json:"message,omitempty"`
	Receipt  *PdaReceipt  `json:"receipt,omitempty"`
	Stocks   []PdaStock   `json:"stocks,omitempty"`
	Pick     *PdaPick     `json:"pick,omitempty"`
	Return   *PdaReturn   `json:"returnDoc,omitempty"`
	Count    *PdaCount    `json:"count,omitempty"`
	Transfer *PdaTransfer `json:"transfer,omitempty"`
}

func (s *FlowService) Lookup(ctx context.Context, bearer, code string, warehouseID uint64) (*PdaHit, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("%w: 请扫码", ErrBadRequest)
	}
	if receipt, err := s.findOpenReceipt(code); err != nil {
		return nil, err
	} else if receipt != nil {
		view, err := s.view(receipt.ID)
		if err != nil {
			return nil, err
		}
		return &PdaHit{Kind: "receipt", Code: code, Receipt: view}, nil
	}
	if hit, err := s.lookupLocalCycle(code); err != nil || hit != nil {
		return hit, err
	}
	if stocks, ok, err := s.stocksByLocation(code); err != nil {
		return nil, err
	} else if ok {
		return &PdaHit{Kind: "stock", Code: code, Message: "库位 " + code, Stocks: stocks}, nil
	}
	if stocks, ok, err := s.stocksBySku(code); err != nil {
		return nil, err
	} else if ok {
		return &PdaHit{Kind: "stock", Code: code, Message: "商品 " + code, Stocks: stocks}, nil
	}
	if s.sc != nil {
		notice, err := s.sc.LookupArrival(ctx, bearer, code)
		if err == nil && notice != nil && notice.PoNo != "" {
			id, err := s.createFromArrival(notice, warehouseID)
			if err != nil {
				return nil, err
			}
			view, err := s.view(id)
			if err != nil {
				return nil, err
			}
			return &PdaHit{Kind: "receipt", Code: code, Message: "已按采购单生成收货单", Receipt: view}, nil
		}
	}
	if s.peers != nil {
		if order, err := s.peers.LookupFulfillment(ctx, bearer, code); err == nil && order != nil {
			return s.createPick(order, warehouseID)
		}
		if notice, err := s.peers.LookupReturn(ctx, bearer, code); err == nil && notice != nil {
			return s.createReturn(notice, warehouseID)
		}
	}
	return &PdaHit{Kind: "miss", Code: code, Message: "没有对应单据。采购到货可以现场补收货单。"}, nil
}

func (s *FlowService) CreateWalkIn(logisticsNo string, warehouseID uint64) (*PdaHit, error) {
	logisticsNo = strings.TrimSpace(logisticsNo)
	if logisticsNo == "" {
		return nil, fmt.Errorf("%w: 请填写物流单号", ErrBadRequest)
	}
	if existing, err := s.findOpenReceipt(logisticsNo); err != nil {
		return nil, err
	} else if existing != nil {
		view, err := s.view(existing.ID)
		if err != nil {
			return nil, err
		}
		return &PdaHit{Kind: "receipt", Code: logisticsNo, Receipt: view}, nil
	}
	wh, err := s.pickWarehouse(warehouseID)
	if err != nil {
		return nil, err
	}
	order := model.ReceiptOrder{
		TenantID: s.tenantID, DocNo: newDocNo("RCV"), WarehouseID: wh,
		LogisticsNo: logisticsNo, Status: "open", Remark: "现场补单",
	}
	if err := s.repos.DB.Create(&order).Error; err != nil {
		return nil, err
	}
	view, err := s.view(order.ID)
	if err != nil {
		return nil, err
	}
	return &PdaHit{Kind: "receipt", Code: logisticsNo, Message: "已补收货单", Receipt: view}, nil
}

func (s *FlowService) ScanReceive(receiptID uint64, barcode, batch, expiry string, qty float64) (*PdaHit, error) {
	if qty <= 0 {
		qty = 1
	}
	barcode = strings.TrimSpace(barcode)
	order, err := s.mustReceipt(receiptID)
	if err != nil {
		return nil, err
	}
	if order.Status != "open" {
		return nil, fmt.Errorf("%w: 这张收货单已经确认", ErrInvalidStatus)
	}
	sku, skuErr := s.findSku(barcode)
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var lines []model.ReceiptLine
		if e := tx.Where("tenant_id = ? AND receipt_id = ?", s.tenantID, order.ID).Find(&lines).Error; e != nil {
			return e
		}
		var line *model.ReceiptLine
		for i := range lines {
			if sameCode(lines[i].SkuCode, barcode) || (sku != nil && lines[i].InvSkuID == sku.ID) {
				line = &lines[i]
				break
			}
		}
		if line == nil {
			if order.PoNo != "" {
				return s.addVariance(tx, order.ID, sku, barcode, "wrong", qty, "不在这张收货单上")
			}
			if skuErr != nil || sku == nil {
				return fmt.Errorf("%w: 商品不在资料里", ErrBadRequest)
			}
			row := model.ReceiptLine{
				TenantID: s.tenantID, ReceiptID: order.ID, InvSkuID: sku.ID,
				SkuCode: sku.SkuCode, ProductName: sku.PickName,
				BatchNo: batch, Expiry: expiry, ExpectedQty: qty, ReceivedQty: qty,
			}
			return tx.Create(&row).Error
		}
		if sku != nil && line.InvSkuID == 0 {
			line.InvSkuID = sku.ID
		}
		if batch != "" {
			line.BatchNo = batch
		}
		if expiry != "" {
			line.Expiry = expiry
		}
		if order.PoNo == "" {
			line.ExpectedQty += qty
			line.ReceivedQty += qty
			return tx.Save(line).Error
		}
		room := line.ExpectedQty - line.ReceivedQty
		if room < 0 {
			room = 0
		}
		take := qty
		if take > room {
			take = room
		}
		line.ReceivedQty += take
		if e := tx.Save(line).Error; e != nil {
			return e
		}
		if extra := qty - take; extra > 0.0001 {
			return s.addVariance(tx, order.ID, sku, line.SkuCode, "over", extra, "超过应收")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	view, err := s.view(order.ID)
	if err != nil {
		return nil, err
	}
	return &PdaHit{Kind: "receipt", Code: order.LogisticsNo, Receipt: view}, nil
}

func (s *FlowService) MarkVariance(receiptID uint64, barcode, reason string, qty float64) (*PdaHit, error) {
	if qty <= 0 {
		qty = 1
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "damage"
	}
	order, err := s.mustReceipt(receiptID)
	if err != nil {
		return nil, err
	}
	if order.Status != "open" {
		return nil, fmt.Errorf("%w: 这张收货单已经确认", ErrInvalidStatus)
	}
	sku, _ := s.findSku(barcode)
	if err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		return s.addVariance(tx, order.ID, sku, barcode, reason, qty, "")
	}); err != nil {
		return nil, err
	}
	view, err := s.view(order.ID)
	if err != nil {
		return nil, err
	}
	return &PdaHit{Kind: "receipt", Code: order.LogisticsNo, Receipt: view}, nil
}

func (s *FlowService) ConfirmReceive(receiptID uint64) (*PdaHit, error) {
	order, err := s.mustReceipt(receiptID)
	if err != nil {
		return nil, err
	}
	if order.Status != "open" {
		return nil, fmt.Errorf("%w: 这张收货单已经确认", ErrInvalidStatus)
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var lines []model.ReceiptLine
		if e := tx.Where("tenant_id = ? AND receipt_id = ?", s.tenantID, order.ID).Find(&lines).Error; e != nil {
			return e
		}
		received := 0.0
		for _, line := range lines {
			received += line.ReceivedQty
			if order.PoNo != "" && line.ExpectedQty-line.ReceivedQty > 0.0001 {
				if e := s.addVariance(tx, order.ID, nil, line.SkuCode, "short", line.ExpectedQty-line.ReceivedQty, "少于应收"); e != nil {
					return e
				}
			}
		}
		if received <= 0.0001 {
			return fmt.Errorf("%w: 还没有实收数量", ErrBadRequest)
		}
		stageID, e := s.ensureStage(tx, order.WarehouseID)
		if e != nil {
			return e
		}
		for _, line := range lines {
			if line.ReceivedQty <= 0 || line.InvSkuID == 0 {
				if line.ReceivedQty > 0 && line.InvSkuID == 0 {
					return fmt.Errorf("%w: %s 还没有对应商品，不能上架", ErrBadRequest, line.SkuCode)
				}
				continue
			}
			suggest, e := s.suggestLocation(tx, order.WarehouseID, line.InvSkuID)
			if e != nil {
				return e
			}
			if e := s.addPending(tx, order.WarehouseID, stageID, line.InvSkuID, line.ReceivedQty, line.BatchNo, line.Expiry); e != nil {
				return e
			}
			task := model.PutawayTask{
				TenantID: s.tenantID, ReceiptID: order.ID, WarehouseID: order.WarehouseID,
				InvSkuID: line.InvSkuID, SkuCode: line.SkuCode, BatchNo: line.BatchNo, Expiry: line.Expiry,
				SuggestCode: suggest, Qty: line.ReceivedQty, Status: "open",
			}
			if e := tx.Create(&task).Error; e != nil {
				return e
			}
		}
		return tx.Model(&model.ReceiptOrder{}).Where("id = ?", order.ID).Update("status", "putaway").Error
	})
	if err != nil {
		return nil, err
	}
	view, err := s.view(order.ID)
	if err != nil {
		return nil, err
	}
	return &PdaHit{Kind: "receipt", Code: order.LogisticsNo, Message: "已进入待上架", Receipt: view}, nil
}

func (s *FlowService) Place(taskID uint64, locationCode, barcode string, qty float64) (*PdaHit, error) {
	locationCode = strings.TrimSpace(locationCode)
	barcode = strings.TrimSpace(barcode)
	var task model.PutawayTask
	if err := s.repos.DB.Where("tenant_id = ? AND id = ?", s.tenantID, taskID).First(&task).Error; err != nil {
		return nil, mapNotFound(err)
	}
	if task.Status != "open" {
		return nil, fmt.Errorf("%w: 这个上架任务已经完成", ErrInvalidStatus)
	}
	if !sameCode(task.SkuCode, barcode) {
		return nil, fmt.Errorf("%w: 商品不符，请扫 %s", ErrBadRequest, task.SkuCode)
	}
	remain := task.Qty - task.DoneQty
	if qty <= 0 || qty > remain {
		qty = remain
	}
	err := s.repos.DB.Transaction(func(tx *gorm.DB) error {
		var loc model.WarehouseLocation
		if e := tx.Where("tenant_id = ? AND warehouse_id = ? AND code = ?", s.tenantID, task.WarehouseID, locationCode).First(&loc).Error; e != nil {
			if e == gorm.ErrRecordNotFound {
				return fmt.Errorf("%w: 库位不存在", ErrBadRequest)
			}
			return e
		}
		if loc.WarehouseID != task.WarehouseID {
			return fmt.Errorf("%w: 库位不属于当前仓库", ErrBadRequest)
		}
		switch loc.Status {
		case model.LocStatusOff:
			return fmt.Errorf("%w: 库位已禁用", ErrBadRequest)
		case model.LocStatusLocked:
			return fmt.Errorf("%w: 库位已锁定", ErrBadRequest)
		case model.LocStatusFull:
			return fmt.Errorf("%w: 库位已满", ErrBadRequest)
		case model.LocStatusOn:
		default:
			return fmt.Errorf("%w: 库位不可用", ErrBadRequest)
		}
		if loc.Code == model.StageLocationCode || !sameCode(loc.Code, task.SuggestCode) {
			return fmt.Errorf("%w: 请扫推荐库位 %s", ErrBadRequest, task.SuggestCode)
		}
		stageID, e := s.ensureStage(tx, task.WarehouseID)
		if e != nil {
			return e
		}
		if e := s.addPending(tx, task.WarehouseID, stageID, task.InvSkuID, -qty, "", ""); e != nil {
			return e
		}
		engine := NewStockEngine(s.repos.DB, s.tenantID)
		if e := engine.ApplyMoves(tx, []MoveLine{{
			WarehouseID: task.WarehouseID, LocationID: loc.ID, InvSkuID: task.InvSkuID, Qty: qty,
			MoveType: model.MovePurchaseIn, DocType: "putaway", DocID: task.ReceiptID,
			Remark: task.SkuCode, CreatedBy: 0,
		}}); e != nil {
			return e
		}
		if e := tx.Model(&model.InvBalance{}).
			Where("tenant_id = ? AND warehouse_id = ? AND location_id = ? AND inv_sku_id = ?", s.tenantID, task.WarehouseID, loc.ID, task.InvSkuID).
			Updates(map[string]interface{}{"last_batch": task.BatchNo, "last_expiry": task.Expiry}).Error; e != nil {
			return e
		}
		var bound int64
		if e := tx.Model(&model.InvLocationSku{}).
			Where("tenant_id = ? AND location_id = ? AND inv_sku_id = ?", s.tenantID, loc.ID, task.InvSkuID).
			Count(&bound).Error; e != nil {
			return e
		}
		if bound == 0 {
			row := model.InvLocationSku{TenantID: s.tenantID, WarehouseID: task.WarehouseID, LocationID: loc.ID, InvSkuID: task.InvSkuID}
			if e := tx.Create(&row).Error; e != nil {
				return e
			}
		}
		done := task.DoneQty + qty
		status := "open"
		if done+0.0001 >= task.Qty {
			status = "done"
			done = task.Qty
		}
		if e := tx.Model(&task).Updates(map[string]interface{}{"done_qty": done, "status": status}).Error; e != nil {
			return e
		}
		var left int64
		if e := tx.Model(&model.PutawayTask{}).Where("tenant_id = ? AND receipt_id = ? AND status = ?", s.tenantID, task.ReceiptID, "open").Count(&left).Error; e != nil {
			return e
		}
		if left == 0 {
			return tx.Model(&model.ReceiptOrder{}).Where("id = ?", task.ReceiptID).Update("status", "done").Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	view, err := s.view(task.ReceiptID)
	if err != nil {
		return nil, err
	}
	return &PdaHit{Kind: "receipt", Receipt: view, Message: "已上架到 " + locationCode}, nil
}

func (s *FlowService) Move(fromCode, barcode, toCode string, qty float64) (*PdaHit, error) {
	fromCode = strings.TrimSpace(fromCode)
	toCode = strings.TrimSpace(toCode)
	barcode = strings.TrimSpace(barcode)
	if qty <= 0 {
		return nil, fmt.Errorf("%w: 请填写移出数量", ErrBadRequest)
	}
	if sameCode(fromCode, toCode) {
		return nil, fmt.Errorf("%w: 移出和移入库位相同", ErrBadRequest)
	}
	sku, err := s.findSku(barcode)
	if err != nil || sku == nil {
		return nil, fmt.Errorf("%w: 商品不在资料里", ErrBadRequest)
	}
	var from, to model.WarehouseLocation
	if err := s.repos.DB.Where("tenant_id = ? AND code = ?", s.tenantID, fromCode).First(&from).Error; err != nil {
		return nil, fmt.Errorf("%w: 移出库位不存在", ErrBadRequest)
	}
	if err := s.repos.DB.Where("tenant_id = ? AND warehouse_id = ? AND code = ?", s.tenantID, from.WarehouseID, toCode).First(&to).Error; err != nil {
		return nil, fmt.Errorf("%w: 移入库位不存在或不在同一仓库", ErrBadRequest)
	}
	if from.Status != model.LocStatusOn || to.Status != model.LocStatusOn {
		return nil, fmt.Errorf("%w: 库位不可用", ErrBadRequest)
	}
	var bal model.InvBalance
	if err := s.repos.DB.Where("tenant_id = ? AND location_id = ? AND inv_sku_id = ?", s.tenantID, from.ID, sku.ID).First(&bal).Error; err != nil || bal.OnHand+0.0001 < qty {
		return nil, fmt.Errorf("%w: 移出库位库存不足", ErrInsufficientStock)
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		engine := NewStockEngine(s.repos.DB, s.tenantID)
		return engine.ApplyMoves(tx, []MoveLine{
			{WarehouseID: from.WarehouseID, LocationID: from.ID, InvSkuID: sku.ID, Qty: -qty, MoveType: model.MoveTransferOut, DocType: "bin_move"},
			{WarehouseID: to.WarehouseID, LocationID: to.ID, InvSkuID: sku.ID, Qty: qty, MoveType: model.MoveTransferIn, DocType: "bin_move"},
		})
	})
	if err != nil {
		return nil, err
	}
	stocks, _, err := s.stocksBySku(sku.SkuCode)
	if err != nil {
		return nil, err
	}
	return &PdaHit{Kind: "stock", Code: sku.SkuCode, Message: "已从 " + fromCode + " 移到 " + toCode, Stocks: stocks}, nil
}

func (s *FlowService) createFromArrival(notice *supplycore.ArrivalNotice, warehouseID uint64) (uint64, error) {
	if existing, err := s.findOpenReceipt(notice.TrackingNo); err != nil {
		return 0, err
	} else if existing != nil {
		return existing.ID, nil
	}
	wh := notice.WarehouseID
	if warehouseID > 0 {
		wh = warehouseID
	}
	picked, err := s.pickWarehouse(wh)
	if err != nil {
		return 0, err
	}
	order := model.ReceiptOrder{
		TenantID: s.tenantID, DocNo: newDocNo("RCV"), WarehouseID: picked,
		LogisticsNo: notice.TrackingNo, PoNo: notice.PoNo, PoID: notice.PoID,
		SupplierName: notice.SupplierName, Status: "open",
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(&order).Error; e != nil {
			return e
		}
		for _, line := range notice.Lines {
			if strings.TrimSpace(line.SkuCode) == "" || line.Qty <= 0 {
				continue
			}
			row := model.ReceiptLine{
				TenantID: s.tenantID, ReceiptID: order.ID, SkuCode: line.SkuCode,
				ProductName: line.ProductName, ExpectedQty: float64(line.Qty),
			}
			if sku, e := s.findSku(line.SkuCode); e == nil && sku != nil {
				row.InvSkuID = sku.ID
				if row.ProductName == "" {
					row.ProductName = sku.PickName
				}
			}
			if e := tx.Create(&row).Error; e != nil {
				return e
			}
		}
		return nil
	})
	return order.ID, err
}

func (s *FlowService) findOpenReceipt(logisticsNo string) (*model.ReceiptOrder, error) {
	var order model.ReceiptOrder
	err := s.repos.DB.Where("tenant_id = ? AND logistics_no = ? AND status <> ?", s.tenantID, logisticsNo, "done").
		Order("id DESC").First(&order).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (s *FlowService) mustReceipt(id uint64) (*model.ReceiptOrder, error) {
	var order model.ReceiptOrder
	err := s.repos.DB.Where("tenant_id = ? AND id = ?", s.tenantID, id).First(&order).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &order, nil
}

func (s *FlowService) view(id uint64) (*PdaReceipt, error) {
	order, err := s.mustReceipt(id)
	if err != nil {
		return nil, err
	}
	var lines []model.ReceiptLine
	var variances []model.ReceiptVariance
	var tasks []model.PutawayTask
	if err := s.repos.DB.Where("tenant_id = ? AND receipt_id = ?", s.tenantID, id).Order("id").Find(&lines).Error; err != nil {
		return nil, err
	}
	if err := s.repos.DB.Where("tenant_id = ? AND receipt_id = ?", s.tenantID, id).Order("id").Find(&variances).Error; err != nil {
		return nil, err
	}
	if err := s.repos.DB.Where("tenant_id = ? AND receipt_id = ?", s.tenantID, id).Order("id").Find(&tasks).Error; err != nil {
		return nil, err
	}
	out := &PdaReceipt{
		ID: order.ID, DocNo: order.DocNo, LogisticsNo: order.LogisticsNo, PoNo: order.PoNo,
		SupplierName: order.SupplierName, Status: order.Status, StatusLabel: receiptLabel(order.Status),
		Lines: []PdaLine{}, Variances: []PdaVariance{}, Tasks: []PdaTask{},
	}
	for _, line := range lines {
		out.Lines = append(out.Lines, PdaLine{
			SkuCode: line.SkuCode, ProductName: line.ProductName, BatchNo: line.BatchNo, Expiry: line.Expiry,
			ExpectedQty: line.ExpectedQty, ReceivedQty: line.ReceivedQty,
		})
	}
	for _, row := range variances {
		out.Variances = append(out.Variances, PdaVariance{SkuCode: row.SkuCode, Reason: reasonLabel(row.Reason), Qty: row.Qty, Note: row.Note})
	}
	for _, task := range tasks {
		out.Tasks = append(out.Tasks, PdaTask{
			ID: task.ID, SkuCode: task.SkuCode, BatchNo: task.BatchNo, Expiry: task.Expiry,
			SuggestCode: task.SuggestCode, Qty: task.Qty, DoneQty: task.DoneQty, Status: task.Status,
		})
	}
	return out, nil
}

func (s *FlowService) stocksByLocation(code string) ([]PdaStock, bool, error) {
	var loc model.WarehouseLocation
	err := s.repos.DB.Where("tenant_id = ? AND code = ?", s.tenantID, code).First(&loc).Error
	if err == gorm.ErrRecordNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	rows, err := s.stockRows("b.location_id = ?", loc.ID)
	return rows, true, err
}

func (s *FlowService) stocksBySku(code string) ([]PdaStock, bool, error) {
	sku, err := s.findSku(code)
	if err != nil || sku == nil {
		return nil, false, nil
	}
	rows, err := s.stockRows("b.inv_sku_id = ?", sku.ID)
	if err != nil {
		return nil, false, err
	}
	if rows == nil {
		rows = []PdaStock{}
	}
	return rows, true, nil
}

func (s *FlowService) stockRows(where string, arg interface{}) ([]PdaStock, error) {
	var rows []PdaStock
	err := s.repos.DB.Table("inv_balances AS b").
		Select(`s.sku_code, COALESCE(s.pick_name,'') AS product_name, l.code AS location_code,
			b.on_hand, b.pending_qty, b.last_batch AS batch_no, b.last_expiry AS expiry`).
		Joins("JOIN inv_skus s ON s.id = b.inv_sku_id").
		Joins("JOIN warehouse_locations l ON l.id = b.location_id").
		Where("b.tenant_id = ?", s.tenantID).
		Where(where, arg).
		Where("b.on_hand <> 0 OR b.pending_qty <> 0").
		Scan(&rows).Error
	return rows, err
}

func (s *FlowService) findSku(code string) (*model.InvSku, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var sku model.InvSku
	err := s.repos.DB.Where("tenant_id = ? AND (sku_code = ? OR upc = ?)", s.tenantID, code, code).First(&sku).Error
	if err != nil {
		return nil, err
	}
	return &sku, nil
}

func (s *FlowService) pickWarehouse(id uint64) (uint64, error) {
	var wh model.Warehouse
	q := s.repos.DB.Where("tenant_id = ?", s.tenantID)
	if id > 0 {
		q = q.Where("id = ?", id)
	} else {
		q = q.Order("id")
	}
	if err := q.First(&wh).Error; err != nil {
		return 0, fmt.Errorf("%w: 还没有仓库", ErrBadRequest)
	}
	return wh.ID, nil
}

func (s *FlowService) ensureStage(tx *gorm.DB, warehouseID uint64) (uint64, error) {
	var loc model.WarehouseLocation
	err := tx.Where("tenant_id = ? AND warehouse_id = ? AND code = ?", s.tenantID, warehouseID, model.StageLocationCode).First(&loc).Error
	if err == nil {
		return loc.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	loc = model.WarehouseLocation{
		TenantID: s.tenantID, WarehouseID: warehouseID, Code: model.StageLocationCode,
		Zone: "收货暂存", Status: model.LocStatusOn,
	}
	if err := tx.Create(&loc).Error; err != nil {
		return 0, err
	}
	return loc.ID, nil
}

func (s *FlowService) suggestLocation(tx *gorm.DB, warehouseID, skuID uint64) (string, error) {
	var bound []uint64
	if err := tx.Model(&model.InvLocationSku{}).
		Where("tenant_id = ? AND warehouse_id = ? AND inv_sku_id = ?", s.tenantID, warehouseID, skuID).
		Pluck("location_id", &bound).Error; err != nil {
		return "", err
	}
	var locs []model.WarehouseLocation
	if err := tx.Where("tenant_id = ? AND warehouse_id = ? AND status = ? AND code <> ?", s.tenantID, warehouseID, model.LocStatusOn, model.StageLocationCode).
		Order("zone, shelf, level, bin, code").Find(&locs).Error; err != nil {
		return "", err
	}
	usable := make([]model.WarehouseLocation, 0, len(locs))
	for _, loc := range locs {
		if loc.Code == model.DefaultLocationCode {
			continue
		}
		usable = append(usable, loc)
	}
	if len(usable) == 0 {
		return "", fmt.Errorf("%w: 没有可上架库位，请先建立库位", ErrBadRequest)
	}
	for _, loc := range usable {
		for _, id := range bound {
			if loc.ID == id {
				return loc.Code, nil
			}
		}
	}
	return usable[0].Code, nil
}

func (s *FlowService) addPending(tx *gorm.DB, warehouseID, locationID, skuID uint64, qty float64, batch, expiry string) error {
	var bal model.InvBalance
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_id = ? AND warehouse_id = ? AND location_id = ? AND inv_sku_id = ?", s.tenantID, warehouseID, locationID, skuID).
		First(&bal).Error
	if err == gorm.ErrRecordNotFound {
		if qty < 0 {
			return fmt.Errorf("%w: 待上架数量不足", ErrBadRequest)
		}
		bal = model.InvBalance{
			TenantID: s.tenantID, WarehouseID: warehouseID, LocationID: locationID, InvSkuID: skuID,
			PendingQty: qty, LastBatch: batch, LastExpiry: expiry,
		}
		return tx.Create(&bal).Error
	}
	if err != nil {
		return err
	}
	next := bal.PendingQty + qty
	if next < -0.0001 {
		return fmt.Errorf("%w: 待上架数量不足", ErrBadRequest)
	}
	updates := map[string]interface{}{"pending_qty": next, "updated_at": time.Now()}
	if batch != "" {
		updates["last_batch"] = batch
	}
	if expiry != "" {
		updates["last_expiry"] = expiry
	}
	return tx.Model(&bal).Updates(updates).Error
}

func (s *FlowService) addVariance(tx *gorm.DB, receiptID uint64, sku *model.InvSku, code, reason string, qty float64, note string) error {
	row := model.ReceiptVariance{
		TenantID: s.tenantID, ReceiptID: receiptID, SkuCode: code, Reason: reason, Qty: qty, Note: note,
	}
	if sku != nil {
		row.InvSkuID = sku.ID
		if row.SkuCode == "" {
			row.SkuCode = sku.SkuCode
		}
	}
	return tx.Create(&row).Error
}

func sameCode(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func newDocNo(prefix string) string {
	return prefix + time.Now().Format("20060102150405")
}

func receiptLabel(status string) string {
	switch status {
	case "open":
		return "待收货"
	case "putaway":
		return "待上架"
	case "done":
		return "已上架"
	default:
		return status
	}
}

func reasonLabel(reason string) string {
	switch reason {
	case "over":
		return "多收"
	case "short":
		return "少收"
	case "wrong":
		return "错收"
	case "damage":
		return "破损"
	default:
		return reason
	}
}
