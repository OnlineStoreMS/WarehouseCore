package admin

import (
	"net/http"
	"strconv"
	"time"

	"warehousecore/internal/dto"
	"warehousecore/internal/pkg/authcontext"
	"warehousecore/internal/pkg/httputil"
	"warehousecore/internal/pkg/response"
	"warehousecore/internal/service"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Master *service.MasterService
	Doc    *service.DocumentService
	Query  *service.QueryService
	Integ  *service.IntegrationService
}

func NewHandlers(master *service.MasterService, doc *service.DocumentService, query *service.QueryService, integ *service.IntegrationService) *Handlers {
	return &Handlers{Master: master, Doc: doc, Query: query, Integ: integ}
}

func (h *Handlers) master(c *gin.Context) *service.MasterService {
	return h.Master.ForTenant(authcontext.TenantID(c))
}
func (h *Handlers) doc(c *gin.Context) *service.DocumentService {
	return h.Doc.ForTenant(authcontext.TenantID(c))
}
func (h *Handlers) query(c *gin.Context) *service.QueryService {
	return h.Query.ForTenant(authcontext.TenantID(c))
}
func (h *Handlers) integ(c *gin.Context) *service.IntegrationService {
	return h.Integ.ForTenant(authcontext.TenantID(c))
}

// ── Categories ──

func (h *Handlers) ListCategories(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.master(c).ListCategories(c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) CreateCategory(c *gin.Context) {
	var in dto.InvCategoryDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.master(c).CreateCategory(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) UpdateCategory(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.InvCategoryDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.master(c).UpdateCategory(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) DeleteCategory(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.master(c).DeleteCategory(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ── Products ──

func (h *Handlers) ListProducts(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	catID, _ := strconv.ParseUint(c.Query("categoryId"), 10, 64)
	list, total, err := h.master(c).ListProducts(c.Query("keyword"), catID, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) GetProduct(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.master(c).GetProduct(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CreateProduct(c *gin.Context) {
	var in dto.InvProductDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.master(c).CreateProduct(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) UpdateProduct(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.InvProductDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.master(c).UpdateProduct(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) DeleteProduct(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.master(c).DeleteProduct(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ── SKUs ──

func (h *Handlers) ListSkus(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.master(c).ListSkus(c.Query("keyword"), c.Query("productType"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) GetSku(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.master(c).GetSku(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CreateSku(c *gin.Context) {
	var in dto.InvSkuDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.master(c).CreateSku(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) UpdateSku(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.InvSkuDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.master(c).UpdateSku(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) DeleteSku(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.master(c).DeleteSku(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ── BOM ──

func (h *Handlers) ListBoms(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.master(c).ListBoms(page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) GetBom(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.master(c).GetBom(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) SaveBom(c *gin.Context) {
	var in dto.BomDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.master(c).SaveBom(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) DeleteBom(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.master(c).DeleteBom(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ── Warehouses ──

func (h *Handlers) ListWarehouses(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.master(c).ListWarehouses(c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) CreateWarehouse(c *gin.Context) {
	var in dto.WarehouseDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.master(c).CreateWarehouse(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) UpdateWarehouse(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.WarehouseDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.master(c).UpdateWarehouse(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) DeleteWarehouse(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.master(c).DeleteWarehouse(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ── Locations ──

func (h *Handlers) ListLocations(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	whID, _ := strconv.ParseUint(c.Query("warehouseId"), 10, 64)
	list, total, err := h.master(c).ListLocations(whID, c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) CreateLocation(c *gin.Context) {
	var in dto.LocationDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.master(c).CreateLocation(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) UpdateLocation(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.LocationDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.master(c).UpdateLocation(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) DeleteLocation(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.master(c).DeleteLocation(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ── Stock queries ──

func (h *Handlers) QueryBalances(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	whID, _ := strconv.ParseUint(c.Query("warehouseId"), 10, 64)
	locID, _ := strconv.ParseUint(c.Query("locationId"), 10, 64)
	skuID, _ := strconv.ParseUint(c.Query("invSkuId"), 10, 64)
	catID, _ := strconv.ParseUint(c.Query("categoryId"), 10, 64)
	list, total, err := h.query(c).QueryBalances(dto.StockQuery{
		WarehouseID: whID, LocationID: locID, InvSkuID: skuID, CategoryID: catID,
		SkuCode: c.Query("skuCode"), Keyword: c.Query("keyword"), Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) QuerySummary(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	whID, _ := strconv.ParseUint(c.Query("warehouseId"), 10, 64)
	var from, to *time.Time
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end := t.Add(24*time.Hour - time.Nanosecond)
			to = &end
		}
	}
	list, total, err := h.query(c).QuerySummary(whID, from, to, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) QueryMovements(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	whID, _ := strconv.ParseUint(c.Query("warehouseId"), 10, 64)
	skuID, _ := strconv.ParseUint(c.Query("invSkuId"), 10, 64)
	var from, to *time.Time
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end := t.Add(24*time.Hour - time.Nanosecond)
			to = &end
		}
	}
	list, total, err := h.query(c).QueryMovements(whID, skuID, c.Query("moveType"), c.Query("docNo"), from, to, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) QuerySlowMoving(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	whID, _ := strconv.ParseUint(c.Query("warehouseId"), 10, 64)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	minOnHand, _ := strconv.ParseFloat(c.DefaultQuery("minOnHand", "0"), 64)
	list, total, err := h.query(c).QuerySlowMoving(dto.SlowMovingQuery{
		WarehouseID: whID, Days: days, MinOnHand: minOnHand, Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

// ── Other inbound/outbound ──

func (h *Handlers) ListOtherIn(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.doc(c).ListOtherIn(c.Query("keyword"), c.Query("status"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) GetOtherIn(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.doc(c).GetOtherIn(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CreateOtherIn(c *gin.Context) {
	var in dto.OtherInboundDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.doc(c).CreateOtherIn(&in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) PostOtherIn(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.doc(c).PostOtherIn(id, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CancelOtherIn(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.doc(c).CancelOtherIn(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handlers) ListOtherOut(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.doc(c).ListOtherOut(c.Query("keyword"), c.Query("status"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) GetOtherOut(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.doc(c).GetOtherOut(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CreateOtherOut(c *gin.Context) {
	var in dto.OtherOutboundDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.doc(c).CreateOtherOut(&in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) PostOtherOut(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.doc(c).PostOtherOut(id, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CancelOtherOut(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.doc(c).CancelOtherOut(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ── Stocktake ──

func (h *Handlers) ListStocktakes(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.doc(c).ListStocktakes(c.Query("keyword"), c.Query("status"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) GetStocktake(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.doc(c).GetStocktake(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CreateStocktake(c *gin.Context) {
	var in dto.StocktakeCreateDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.doc(c).CreateStocktake(&in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) StartStocktake(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.doc(c).StartCounting(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) SubmitStocktakeCount(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.StocktakeCountDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.doc(c).SubmitCount(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) PostStocktake(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.doc(c).PostStocktake(id, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CancelStocktake(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.doc(c).CancelStocktake(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handlers) ListStocktakeDetails(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.doc(c).ListStocktakeDetails(c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

// ── Transfers ──

func (h *Handlers) ListTransfers(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.doc(c).ListTransfers(c.Query("keyword"), c.Query("status"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) GetTransfer(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.doc(c).GetTransfer(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CreateTransfer(c *gin.Context) {
	var in dto.TransferDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.doc(c).CreateTransfer(&in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) ShipTransfer(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.doc(c).ShipTransfer(id, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) ReceiveTransfer(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.doc(c).ReceiveTransfer(id, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CancelTransfer(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.doc(c).CancelTransfer(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ── Integrations (M4) ──

func (h *Handlers) ListPimMappings(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.integ(c).ListPimMappings(page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) UpsertPimMapping(c *gin.Context) {
	var in dto.PimMappingDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.integ(c).UpsertPimMapping(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) DeletePimMapping(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.integ(c).DeletePimMapping(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handlers) PurchaseInbound(c *gin.Context) {
	var in dto.PurchaseInboundDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.integ(c).PurchaseInbound(&in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) TransferToStore(c *gin.Context) {
	var in dto.StoreTransferDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.integ(c).TransferToStore(&in, authcontext.UserID(c))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}
