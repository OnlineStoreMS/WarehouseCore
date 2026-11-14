package service

import (
	"strings"

	"warehousecore/internal/model"

	"gorm.io/gorm"
)

type SelfPickLineIn struct {
	InvSkuID    uint64  `json:"invSkuId"`
	SkuCode     string  `json:"skuCode"`
	ProductName string  `json:"productName"`
	Qty         float64 `json:"qty"`
}

type SelfPickIn struct {
	WarehouseID uint64           `json:"warehouseId"`
	SoNo        string           `json:"soNo"`
	Buyer       string           `json:"buyer"`
	Address     string           `json:"address"`
	Unbound     int              `json:"unbound"`
	Lines       []SelfPickLineIn `json:"lines"`
}

type DeskReceipt struct {
	ID            uint64                  `json:"id"`
	DocNo         string                  `json:"docNo"`
	WarehouseID   uint64                  `json:"warehouseId"`
	WarehouseName string                  `json:"warehouseName"`
	LogisticsNo   string                  `json:"logisticsNo"`
	PoNo          string                  `json:"poNo"`
	SupplierName  string                  `json:"supplierName"`
	Status        string                  `json:"status"`
	StatusLabel   string                  `json:"statusLabel"`
	ExpectedQty   float64                 `json:"expectedQty"`
	ReceivedQty   float64                 `json:"receivedQty"`
	Remark        string                  `json:"remark"`
	CreatedAt     string                  `json:"createdAt"`
	Lines         []DeskReceiptLine       `json:"lines,omitempty"`
	Variances     []model.ReceiptVariance `json:"variances,omitempty"`
}

type DeskReceiptLine struct {
	SkuCode     string  `json:"skuCode"`
	ProductName string  `json:"productName"`
	BatchNo     string  `json:"batchNo"`
	Expiry      string  `json:"expiry"`
	ExpectedQty float64 `json:"expectedQty"`
	ReceivedQty float64 `json:"receivedQty"`
}

type DeskPutaway struct {
	ID            uint64  `json:"id"`
	ReceiptID     uint64  `json:"receiptId"`
	DocNo         string  `json:"docNo"`
	WarehouseName string  `json:"warehouseName"`
	SkuCode       string  `json:"skuCode"`
	SuggestCode   string  `json:"suggestCode"`
	BatchNo       string  `json:"batchNo"`
	Expiry        string  `json:"expiry"`
	Qty           float64 `json:"qty"`
	DoneQty       float64 `json:"doneQty"`
	Status        string  `json:"status"`
	StatusLabel   string  `json:"statusLabel"`
}

type DeskOffshelf struct {
	PickID        uint64  `json:"pickId"`
	DocNo         string  `json:"docNo"`
	OrderNo       string  `json:"orderNo"`
	Buyer         string  `json:"buyer"`
	WarehouseName string  `json:"warehouseName"`
	SkuCode       string  `json:"skuCode"`
	ProductName   string  `json:"productName"`
	SuggestCode   string  `json:"suggestCode"`
	Qty           float64 `json:"qty"`
	PickedQty     float64 `json:"pickedQty"`
}

type DeskPick struct {
	ID            uint64        `json:"id"`
	DocNo         string        `json:"docNo"`
	OrderNo       string        `json:"orderNo"`
	Source        string        `json:"source"`
	Buyer         string        `json:"buyer"`
	Address       string        `json:"address"`
	MailNo        string        `json:"mailNo"`
	WarehouseName string        `json:"warehouseName"`
	Status        string        `json:"status"`
	StatusLabel   string        `json:"statusLabel"`
	BlockReason   string        `json:"blockReason"`
	Qty           float64       `json:"qty"`
	PickedQty     float64       `json:"pickedQty"`
	CreatedAt     string        `json:"createdAt"`
	Lines         []PdaPickLine `json:"lines,omitempty"`
}

func (s *FlowService) EnsureSelfPick(in *SelfPickIn) (*PdaPick, error) {
	if in == nil || strings.TrimSpace(in.SoNo) == "" {
		return nil, ErrBadRequest
	}
	if len(in.Lines) == 0 {
		return nil, nil
	}
	wh, err := s.pickWarehouse(in.WarehouseID)
	if err != nil {
		return nil, err
	}
	var existing model.PickOrder
	err = s.repos.DB.Where("tenant_id = ? AND source = ? AND source_no = ? AND status <> ?", s.tenantID, "self", in.SoNo, "done").
		Order("id DESC").First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	found := err == nil
	if found && (existing.Status == "reviewing" || existing.Status == "packed") {
		return s.viewPick(existing.ID)
	}
	var picked float64
	if found {
		s.repos.DB.Model(&model.PickLine{}).Where("tenant_id = ? AND pick_id = ?", s.tenantID, existing.ID).
			Select("COALESCE(SUM(picked_qty),0)").Scan(&picked)
	}
	err = s.repos.DB.Transaction(func(tx *gorm.DB) error {
		row := existing
		if !found {
			row = model.PickOrder{
				TenantID: s.tenantID, WarehouseID: wh, DocNo: newDocNo("PICK"),
				Source: "self", SourceNo: in.SoNo, OrderNo: in.SoNo, Status: "picking",
			}
			if e := tx.Create(&row).Error; e != nil {
				return e
			}
			existing = row
		}
		reasons := s.writeSelfPickLines(tx, &existing, wh, in, picked > 0)
		status := "picking"
		block := ""
		if len(reasons) > 0 {
			status = "blocked"
			block = strings.Join(reasons, "；")
		}
		return tx.Model(&existing).Updates(map[string]interface{}{
			"warehouse_id": wh,
			"buyer":        in.Buyer,
			"address":      in.Address,
			"order_no":     in.SoNo,
			"status":       status,
			"block_reason": block,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.viewPick(existing.ID)
}

func (s *FlowService) writeSelfPickLines(tx *gorm.DB, order *model.PickOrder, warehouseID uint64, in *SelfPickIn, keepPicked bool) []string {
	var reasons []string
	if in.Unbound > 0 {
		reasons = append(reasons, "还有商品未绑定库存SKU")
	}
	if strings.TrimSpace(in.Address) == "" {
		reasons = append(reasons, "收货地址不完整")
	}
	var current []model.PickLine
	if keepPicked {
		_ = tx.Where("tenant_id = ? AND pick_id = ?", s.tenantID, order.ID).Find(&current).Error
	} else {
		_ = tx.Where("tenant_id = ? AND pick_id = ?", s.tenantID, order.ID).Delete(&model.PickLine{}).Error
	}
	for _, line := range in.Lines {
		if line.Qty <= 0 {
			continue
		}
		sku, err := s.skuByIDOrCode(tx, line.InvSkuID, line.SkuCode)
		if err != nil {
			reasons = append(reasons, line.SkuCode+" 读取失败")
			continue
		}
		item := model.PickLine{
			TenantID: s.tenantID, PickID: order.ID, InvSkuID: line.InvSkuID,
			SkuCode: firstText(line.SkuCode, ""), ProductName: line.ProductName, Qty: line.Qty,
		}
		if sku != nil {
			item.InvSkuID = sku.ID
			item.SkuCode = sku.SkuCode
			if item.ProductName == "" {
				item.ProductName = sku.PickName
			}
			code, qty, e := s.bestBin(tx, warehouseID, sku.ID)
			if e != nil {
				reasons = append(reasons, sku.SkuCode+" 库位查询失败")
			} else {
				item.SuggestCode = code
				if qty+0.0001 < line.Qty {
					reasons = append(reasons, sku.SkuCode+" 可用库存不足")
				}
			}
		} else {
			reasons = append(reasons, firstText(line.SkuCode, "商品")+" 不在商品资料里")
		}
		if keepPicked {
			var hit *model.PickLine
			for i := range current {
				if current[i].InvSkuID == item.InvSkuID && item.InvSkuID > 0 {
					hit = &current[i]
					break
				}
			}
			if hit != nil {
				if item.Qty < hit.PickedQty {
					item.Qty = hit.PickedQty
				}
				_ = tx.Model(hit).Updates(map[string]interface{}{
					"qty": item.Qty, "suggest_code": item.SuggestCode, "sku_code": item.SkuCode, "product_name": item.ProductName,
				}).Error
				continue
			}
		}
		if e := tx.Create(&item).Error; e != nil {
			reasons = append(reasons, item.SkuCode+" 写入失败")
		}
	}
	return reasons
}

func (s *FlowService) skuByIDOrCode(tx *gorm.DB, id uint64, code string) (*model.InvSku, error) {
	var sku model.InvSku
	if id > 0 {
		err := tx.Where("tenant_id = ? AND id = ?", s.tenantID, id).First(&sku).Error
		if err == nil {
			return &sku, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}
	err := tx.Where("tenant_id = ? AND (sku_code = ? OR upc = ?)", s.tenantID, code, code).First(&sku).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sku, nil
}

func (s *FlowService) ListReceipts(keyword, status string, warehouseID uint64, page, pageSize int) ([]DeskReceipt, int64, error) {
	page, pageSize = clampPage(page, pageSize)
	q := s.repos.DB.Model(&model.ReceiptOrder{}).Where("tenant_id = ?", s.tenantID)
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("doc_no ILIKE ? OR logistics_no ILIKE ? OR po_no ILIKE ? OR supplier_name ILIKE ?", like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.ReceiptOrder
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	sums := s.receiptSums(rows)
	names := s.warehouseNames(receiptWarehouses(rows))
	out := make([]DeskReceipt, 0, len(rows))
	for _, row := range rows {
		sum := sums[row.ID]
		out = append(out, DeskReceipt{
			ID: row.ID, DocNo: row.DocNo, WarehouseID: row.WarehouseID, WarehouseName: names[row.WarehouseID],
			LogisticsNo: row.LogisticsNo, PoNo: row.PoNo, SupplierName: row.SupplierName,
			Status: row.Status, StatusLabel: receiptLabel(row.Status),
			ExpectedQty: sum[0], ReceivedQty: sum[1], Remark: row.Remark,
			CreatedAt: row.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return out, total, nil
}

func (s *FlowService) GetReceipt(id uint64) (*DeskReceipt, error) {
	var row model.ReceiptOrder
	if err := s.repos.DB.Where("tenant_id = ? AND id = ?", s.tenantID, id).First(&row).Error; err != nil {
		return nil, mapNotFound(err)
	}
	var lines []model.ReceiptLine
	if err := s.repos.DB.Where("tenant_id = ? AND receipt_id = ?", s.tenantID, id).Order("id").Find(&lines).Error; err != nil {
		return nil, err
	}
	var variances []model.ReceiptVariance
	if err := s.repos.DB.Where("tenant_id = ? AND receipt_id = ?", s.tenantID, id).Order("id").Find(&variances).Error; err != nil {
		return nil, err
	}
	view := &DeskReceipt{
		ID: row.ID, DocNo: row.DocNo, WarehouseID: row.WarehouseID, WarehouseName: s.warehouseName(row.WarehouseID),
		LogisticsNo: row.LogisticsNo, PoNo: row.PoNo, SupplierName: row.SupplierName,
		Status: row.Status, StatusLabel: receiptLabel(row.Status), Remark: row.Remark,
		CreatedAt: row.CreatedAt.Format("2006-01-02 15:04:05"),
		Lines:     []DeskReceiptLine{}, Variances: variances,
	}
	for _, line := range lines {
		view.ExpectedQty += line.ExpectedQty
		view.ReceivedQty += line.ReceivedQty
		view.Lines = append(view.Lines, DeskReceiptLine{
			SkuCode: line.SkuCode, ProductName: line.ProductName, BatchNo: line.BatchNo, Expiry: line.Expiry,
			ExpectedQty: line.ExpectedQty, ReceivedQty: line.ReceivedQty,
		})
	}
	return view, nil
}

func (s *FlowService) ListPutaways(keyword, status string, warehouseID uint64, page, pageSize int) ([]DeskPutaway, int64, error) {
	page, pageSize = clampPage(page, pageSize)
	q := s.repos.DB.Model(&model.PutawayTask{}).Where("tenant_id = ?", s.tenantID)
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	switch status {
	case "", "open":
		q = q.Where("status = ? AND done_qty + 0.0001 < qty", "open")
	case "done":
		q = q.Where("status = ?", "done")
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("sku_code ILIKE ? OR suggest_code ILIKE ? OR receipt_id IN (SELECT id FROM receipt_orders WHERE tenant_id = ? AND (doc_no ILIKE ? OR po_no ILIKE ?))",
			like, like, s.tenantID, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.PutawayTask
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	docNos := map[uint64]string{}
	names := map[uint64]string{}
	out := make([]DeskPutaway, 0, len(rows))
	for _, row := range rows {
		if _, ok := docNos[row.ReceiptID]; !ok {
			var receipt model.ReceiptOrder
			if s.repos.DB.Select("doc_no").First(&receipt, row.ReceiptID).Error == nil {
				docNos[row.ReceiptID] = receipt.DocNo
			}
		}
		if _, ok := names[row.WarehouseID]; !ok {
			names[row.WarehouseID] = s.warehouseName(row.WarehouseID)
		}
		label := "待上架"
		if row.Status == "done" || row.DoneQty+0.0001 >= row.Qty {
			label = "已上架"
		}
		out = append(out, DeskPutaway{
			ID: row.ID, ReceiptID: row.ReceiptID, DocNo: docNos[row.ReceiptID], WarehouseName: names[row.WarehouseID],
			SkuCode: row.SkuCode, SuggestCode: row.SuggestCode, BatchNo: row.BatchNo, Expiry: row.Expiry,
			Qty: row.Qty, DoneQty: row.DoneQty, Status: row.Status, StatusLabel: label,
		})
	}
	return out, total, nil
}

func (s *FlowService) ListOffshelves(keyword string, warehouseID uint64, page, pageSize int) ([]DeskOffshelf, int64, error) {
	page, pageSize = clampPage(page, pageSize)
	q := s.repos.DB.Table("pick_lines AS l").
		Joins("JOIN pick_orders p ON p.id = l.pick_id AND p.tenant_id = l.tenant_id").
		Where("l.tenant_id = ? AND p.status = ? AND l.picked_qty + 0.0001 < l.qty", s.tenantID, "picking")
	if warehouseID > 0 {
		q = q.Where("p.warehouse_id = ?", warehouseID)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("p.doc_no ILIKE ? OR p.order_no ILIKE ? OR l.sku_code ILIKE ? OR l.suggest_code ILIKE ?", like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type offRow struct {
		PickID      uint64
		DocNo       string
		OrderNo     string
		Buyer       string
		WarehouseID uint64
		SkuCode     string
		ProductName string
		SuggestCode string
		Qty         float64
		PickedQty   float64
	}
	var raw []offRow
	err := q.Select("p.id AS pick_id, p.doc_no, p.order_no, p.buyer, p.warehouse_id, l.sku_code, l.product_name, l.suggest_code, l.qty, l.picked_qty").
		Order("p.id DESC, l.id").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&raw).Error
	if err != nil {
		return nil, 0, err
	}
	names := map[uint64]string{}
	rows := make([]DeskOffshelf, 0, len(raw))
	for _, row := range raw {
		if _, ok := names[row.WarehouseID]; !ok {
			names[row.WarehouseID] = s.warehouseName(row.WarehouseID)
		}
		rows = append(rows, DeskOffshelf{
			PickID: row.PickID, DocNo: row.DocNo, OrderNo: row.OrderNo, Buyer: row.Buyer,
			WarehouseName: names[row.WarehouseID], SkuCode: row.SkuCode, ProductName: row.ProductName,
			SuggestCode: row.SuggestCode, Qty: row.Qty, PickedQty: row.PickedQty,
		})
	}
	return rows, total, nil
}

func (s *FlowService) ListPicks(keyword, status, stage string, warehouseID uint64, page, pageSize int) ([]DeskPick, int64, error) {
	page, pageSize = clampPage(page, pageSize)
	q := s.repos.DB.Model(&model.PickOrder{}).Where("tenant_id = ?", s.tenantID)
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	if stage == "outbound" && (status == "" || status == "all") {
		q = q.Where("status IN ?", []string{"packed", "done"})
	} else if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("doc_no ILIKE ? OR order_no ILIKE ? OR source_no ILIKE ? OR buyer ILIKE ? OR mail_no ILIKE ?", like, like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.PickOrder
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	sums := s.pickSums(rows)
	out := make([]DeskPick, 0, len(rows))
	for _, row := range rows {
		sum := sums[row.ID]
		out = append(out, DeskPick{
			ID: row.ID, DocNo: row.DocNo, OrderNo: row.OrderNo, Source: row.Source, Buyer: row.Buyer,
			Address: row.Address, MailNo: row.MailNo, WarehouseName: s.warehouseName(row.WarehouseID),
			Status: row.Status, StatusLabel: pickLabel(row.Status), BlockReason: row.BlockReason,
			Qty: sum[0], PickedQty: sum[1], CreatedAt: row.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return out, total, nil
}

func (s *FlowService) GetDeskPick(id uint64) (*DeskPick, error) {
	view, err := s.viewPick(id)
	if err != nil {
		return nil, err
	}
	order, err := s.mustPick(id)
	if err != nil {
		return nil, err
	}
	qty, picked := 0.0, 0.0
	for _, line := range view.Lines {
		qty += line.Qty
		picked += line.PickedQty
	}
	return &DeskPick{
		ID: view.ID, DocNo: view.DocNo, OrderNo: view.OrderNo, Source: view.Source, Buyer: view.Buyer,
		Address: view.Address, MailNo: view.MailNo, WarehouseName: s.warehouseName(order.WarehouseID),
		Status: view.Status, StatusLabel: view.StatusLabel, BlockReason: view.BlockReason,
		Qty: qty, PickedQty: picked, CreatedAt: order.CreatedAt.Format("2006-01-02 15:04:05"), Lines: view.Lines,
	}, nil
}

func (s *FlowService) receiptSums(rows []model.ReceiptOrder) map[uint64][2]float64 {
	out := map[uint64][2]float64{}
	if len(rows) == 0 {
		return out
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	type sum struct {
		ReceiptID uint64
		Expected  float64
		Received  float64
	}
	var sums []sum
	_ = s.repos.DB.Model(&model.ReceiptLine{}).
		Select("receipt_id, COALESCE(SUM(expected_qty),0) AS expected, COALESCE(SUM(received_qty),0) AS received").
		Where("tenant_id = ? AND receipt_id IN ?", s.tenantID, ids).
		Group("receipt_id").Scan(&sums).Error
	for _, row := range sums {
		out[row.ReceiptID] = [2]float64{row.Expected, row.Received}
	}
	return out
}

func (s *FlowService) pickSums(rows []model.PickOrder) map[uint64][2]float64 {
	out := map[uint64][2]float64{}
	if len(rows) == 0 {
		return out
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	type sum struct {
		PickID uint64
		Qty    float64
		Picked float64
	}
	var sums []sum
	_ = s.repos.DB.Model(&model.PickLine{}).
		Select("pick_id, COALESCE(SUM(qty),0) AS qty, COALESCE(SUM(picked_qty),0) AS picked").
		Where("tenant_id = ? AND pick_id IN ?", s.tenantID, ids).
		Group("pick_id").Scan(&sums).Error
	for _, row := range sums {
		out[row.PickID] = [2]float64{row.Qty, row.Picked}
	}
	return out
}

func receiptWarehouses(rows []model.ReceiptOrder) []uint64 {
	ids := make([]uint64, 0, len(rows))
	seen := map[uint64]bool{}
	for _, row := range rows {
		if row.WarehouseID > 0 && !seen[row.WarehouseID] {
			seen[row.WarehouseID] = true
			ids = append(ids, row.WarehouseID)
		}
	}
	return ids
}

func (s *FlowService) warehouseNames(ids []uint64) map[uint64]string {
	out := map[uint64]string{}
	if len(ids) == 0 {
		return out
	}
	var rows []model.Warehouse
	_ = s.repos.DB.Where("id IN ?", ids).Find(&rows).Error
	for _, row := range rows {
		out[row.ID] = row.Name
	}
	return out
}

func clampPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	return page, pageSize
}

func firstText(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}
