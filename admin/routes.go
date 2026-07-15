package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, h *Handlers) {
	// Categories
	g.GET("/categories", h.ListCategories)
	g.POST("/categories", h.CreateCategory)
	g.PUT("/categories/:id", h.UpdateCategory)
	g.DELETE("/categories/:id", h.DeleteCategory)

	// Pack specs
	g.GET("/pack-specs", h.ListPackSpecs)
	g.POST("/pack-specs", h.CreatePackSpec)
	g.PUT("/pack-specs/:id", h.UpdatePackSpec)
	g.DELETE("/pack-specs/:id", h.DeletePackSpec)
	g.GET("/pack-specs/:id/skus", h.ListPackSpecSkus)
	g.POST("/pack-specs/:id/skus", h.BindPackSpecSku)
	g.PUT("/pack-spec-skus/:id", h.UpdatePackSpecSku)
	g.DELETE("/pack-spec-skus/:id", h.UnbindPackSpecSku)

	// Products (parent SKU)
	g.GET("/products", h.ListProducts)
	g.POST("/products", h.CreateProduct)
	g.POST("/products/with-skus", h.CreateProductWithSkus)
	g.GET("/products/:id", h.GetProduct)
	g.PUT("/products/:id", h.UpdateProduct)
	g.PUT("/products/:id/with-skus", h.UpdateProductWithSkus)
	g.DELETE("/products/:id", h.DeleteProduct)

	// Inventory SKUs
	g.GET("/skus", h.ListSkus)
	g.GET("/skus/by-code", h.GetSkuByCode)
	g.POST("/skus/update-weight", h.UpdateSkuWeight)
	g.POST("/skus", h.CreateSku)
	g.GET("/skus/:id", h.GetSku)
	g.PUT("/skus/:id", h.UpdateSku)
	g.DELETE("/skus/:id", h.DeleteSku)

	// BOM
	g.GET("/boms", h.ListBoms)
	g.POST("/boms", h.SaveBom)
	g.GET("/boms/:id", h.GetBom)
	g.DELETE("/boms/:id", h.DeleteBom)

	// Warehouses / locations
	g.GET("/warehouses", h.ListWarehouses)
	g.POST("/warehouses", h.CreateWarehouse)
	g.PUT("/warehouses/:id", h.UpdateWarehouse)
	g.DELETE("/warehouses/:id", h.DeleteWarehouse)

	g.GET("/locations", h.ListLocations)
	g.POST("/locations", h.CreateLocation)
	g.PUT("/locations/:id", h.UpdateLocation)
	g.DELETE("/locations/:id", h.DeleteLocation)
	g.GET("/locations/:id/skus", h.ListLocationSkus)
	g.POST("/locations/:id/skus", h.BindLocationSku)
	g.DELETE("/location-skus/:id", h.UnbindLocationSku)

	// Stock queries
	g.GET("/stock/balances", h.QueryBalances)
	g.GET("/stock/summary", h.QuerySummary)
	g.GET("/stock/movements", h.QueryMovements)
	g.GET("/stock/slow-moving", h.QuerySlowMoving)

	// Other inbound
	g.GET("/other-inbounds", h.ListOtherIn)
	g.POST("/other-inbounds", h.CreateOtherIn)
	g.GET("/other-inbounds/:id", h.GetOtherIn)
	g.POST("/other-inbounds/:id/post", h.PostOtherIn)
	g.POST("/other-inbounds/:id/cancel", h.CancelOtherIn)

	// Other outbound
	g.GET("/other-outbounds", h.ListOtherOut)
	g.POST("/other-outbounds", h.CreateOtherOut)
	g.GET("/other-outbounds/:id", h.GetOtherOut)
	g.POST("/other-outbounds/:id/post", h.PostOtherOut)
	g.POST("/other-outbounds/:id/cancel", h.CancelOtherOut)

	// Stocktake
	g.GET("/stocktakes", h.ListStocktakes)
	g.POST("/stocktakes", h.CreateStocktake)
	g.GET("/stocktakes/:id", h.GetStocktake)
	g.POST("/stocktakes/:id/start", h.StartStocktake)
	g.POST("/stocktakes/:id/count", h.SubmitStocktakeCount)
	g.POST("/stocktakes/:id/post", h.PostStocktake)
	g.POST("/stocktakes/:id/cancel", h.CancelStocktake)
	g.GET("/stocktake-details", h.ListStocktakeDetails)

	// Transfers
	g.GET("/transfers", h.ListTransfers)
	g.POST("/transfers", h.CreateTransfer)
	g.GET("/transfers/:id", h.GetTransfer)
	g.POST("/transfers/:id/ship", h.ShipTransfer)
	g.POST("/transfers/:id/receive", h.ReceiveTransfer)
	g.POST("/transfers/:id/cancel", h.CancelTransfer)

	// M4 integrations
	g.GET("/pim-mappings", h.ListPimMappings)
	g.POST("/pim-mappings", h.UpsertPimMapping)
	g.DELETE("/pim-mappings/:id", h.DeletePimMapping)
	g.POST("/integrations/purchase-inbound", h.PurchaseInbound)
	g.POST("/integrations/transfer-to-store", h.TransferToStore)

	// VMS suppliers (proxy SupplyCore)
	g.GET("/suppliers", h.ListSuppliers)

	// 商品其它工具（对齐普源）
	g.GET("/goods-fee-settings", h.GetGoodsFeeSettings)
	g.PUT("/goods-fee-settings", h.SaveGoodsFeeSettings)
	g.GET("/profit-trials", h.ListProfitTrials)
	g.POST("/profit-trials", h.UpsertProfitTrial)
	g.PUT("/profit-trials/:id", h.UpdateProfitTrial)
	g.POST("/profit-trials/delete", h.DeleteProfitTrials)
	g.POST("/profit-trials/calc", h.CalcProfitTrials)
}
