package service

import (
	"fmt"
	"time"

	"warehousecore/internal/model"
	"warehousecore/internal/repo"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StockEngine posts inventory movements atomically.
type StockEngine struct {
	db       *gorm.DB
	tenantID uint64
}

func NewStockEngine(db *gorm.DB, tenantID uint64) *StockEngine {
	return &StockEngine{db: db, tenantID: repo.NormalizeTenantID(tenantID)}
}

type MoveLine struct {
	WarehouseID uint64
	LocationID  uint64
	InvSkuID    uint64
	Qty         float64 // positive = in, negative = out
	MoveType    string
	DocType     string
	DocNo       string
	DocID       uint64
	RefDocType  string
	RefDocID    uint64
	Remark      string
	CreatedBy   uint64
}

func (e *StockEngine) EnsureDefaultLocation(tx *gorm.DB, warehouseID uint64) (uint64, error) {
	var loc model.WarehouseLocation
	err := tx.Where("tenant_id = ? AND warehouse_id = ? AND code = ?", e.tenantID, warehouseID, model.DefaultLocationCode).
		First(&loc).Error
	if err == nil {
		return loc.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	loc = model.WarehouseLocation{
		TenantID:    e.tenantID,
		WarehouseID: warehouseID,
		Code:        model.DefaultLocationCode,
		Status:      1,
	}
	if err := tx.Create(&loc).Error; err != nil {
		return 0, err
	}
	return loc.ID, nil
}

func (e *StockEngine) ApplyMoves(tx *gorm.DB, lines []MoveLine) error {
	for _, line := range lines {
		if line.Qty == 0 {
			continue
		}
		locID := line.LocationID
		if locID == 0 {
			id, err := e.EnsureDefaultLocation(tx, line.WarehouseID)
			if err != nil {
				return err
			}
			locID = id
		}

		var bal model.InvBalance
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND warehouse_id = ? AND location_id = ? AND inv_sku_id = ?",
				e.tenantID, line.WarehouseID, locID, line.InvSkuID).
			First(&bal).Error
		if err == gorm.ErrRecordNotFound {
			bal = model.InvBalance{
				TenantID:    e.tenantID,
				WarehouseID: line.WarehouseID,
				LocationID:  locID,
				InvSkuID:    line.InvSkuID,
				OnHand:      0,
			}
			if err := tx.Create(&bal).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ?", bal.ID).First(&bal).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		newQty := bal.OnHand + line.Qty
		if newQty < -0.0001 {
			return fmt.Errorf("%w: sku=%d warehouse=%d", ErrInsufficientStock, line.InvSkuID, line.WarehouseID)
		}
		if err := tx.Model(&bal).Updates(map[string]interface{}{
			"on_hand":    newQty,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}

		mv := model.StockMovement{
			TenantID:     e.tenantID,
			WarehouseID:  line.WarehouseID,
			LocationID:   locID,
			InvSkuID:     line.InvSkuID,
			MoveType:     line.MoveType,
			Qty:          line.Qty,
			BalanceAfter: newQty,
			DocType:      line.DocType,
			DocNo:        line.DocNo,
			DocID:        line.DocID,
			RefDocType:   line.RefDocType,
			RefDocID:     line.RefDocID,
			Remark:       line.Remark,
			CreatedBy:    line.CreatedBy,
		}
		if err := tx.Create(&mv).Error; err != nil {
			return err
		}
	}
	return nil
}

func GenDocNo(prefix string) string {
	return fmt.Sprintf("%s%s%04d", prefix, time.Now().Format("20060102150405"), time.Now().Nanosecond()%10000)
}
