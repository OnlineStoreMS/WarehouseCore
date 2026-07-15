package database

import (
	"fmt"
	"os"
	"path/filepath"

	"warehousecore/internal/config"
	"warehousecore/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.PostgresDSN)
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(cfg.SQLitePath)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.InvCategory{},
		&model.InvProduct{},
		&model.InvSku{},
		&model.InvProductSupplier{},
		&model.InvProductDescription{},
		&model.InvPackSpec{},
		&model.InvPackSpecSku{},
		&model.InvBomHeader{},
		&model.InvBomItem{},
		&model.Warehouse{},
		&model.WarehouseLocation{},
		&model.InvBalance{},
		&model.StockMovement{},
		&model.OtherInboundOrder{},
		&model.OtherInboundItem{},
		&model.OtherOutboundOrder{},
		&model.OtherOutboundItem{},
		&model.StocktakeOrder{},
		&model.StocktakeItem{},
		&model.TransferOrder{},
		&model.TransferItem{},
		&model.PimSkuMapping{},
		&model.InvGoodsFeeBase{},
		&model.InvScoreWeightRule{},
		&model.InvOrderQtyCoeff{},
		&model.InvProfitTrial{},
	); err != nil {
		return err
	}
	return ensureIndexes(db)
}

func ensureIndexes(db *gorm.DB) error {
	switch db.Dialector.Name() {
	case "postgres":
		return db.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_cat_tenant_code ON inv_categories (tenant_id, code);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_products_tenant_parent ON inv_products (tenant_id, parent_sku);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_skus_tenant_code ON inv_skus (tenant_id, sku_code);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_product_supplier ON inv_product_suppliers (tenant_id, product_id, supplier_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_product_desc_lang ON inv_product_descriptions (tenant_id, product_id, language_code);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_pack_tenant_name ON inv_pack_specs (tenant_id, name);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_pack_sku_unique ON inv_pack_spec_skus (tenant_id, pack_spec_id, inv_sku_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_bom_tenant_parent ON inv_bom_headers (tenant_id, parent_sku_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_wh_tenant_code ON warehouses (tenant_id, code);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_loc_wh_code ON warehouse_locations (tenant_id, warehouse_id, code);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_bal_unique ON inv_balances (tenant_id, warehouse_id, location_id, inv_sku_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_oin_tenant_no ON other_inbound_orders (tenant_id, doc_no);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_oout_tenant_no ON other_outbound_orders (tenant_id, doc_no);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_stk_tenant_no ON stocktake_orders (tenant_id, doc_no);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_xfer_tenant_no ON transfer_orders (tenant_id, doc_no);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_pim_map_inv ON pim_sku_mappings (tenant_id, inv_sku_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_pim_map_pim ON pim_sku_mappings (tenant_id, pim_sku_id);
		`).Error
	default:
		return nil
	}
}
