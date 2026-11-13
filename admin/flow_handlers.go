package admin

import (
	"net/http"
	"strconv"

	"warehousecore/internal/pkg/httputil"
	"warehousecore/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) PdaLookup(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.Query("warehouseId"), 10, 64)
	hit, err := h.flow(c).Lookup(c.Request.Context(), c.GetHeader("Authorization"), c.Query("code"), warehouseID)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) CreateReceipt(c *gin.Context) {
	var in struct {
		LogisticsNo string `json:"logisticsNo"`
		WarehouseID uint64 `json:"warehouseId"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).CreateWalkIn(in.LogisticsNo, in.WarehouseID)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) ScanReceipt(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		Barcode string  `json:"barcode"`
		BatchNo string  `json:"batchNo"`
		Expiry  string  `json:"expiry"`
		Qty     float64 `json:"qty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).ScanReceive(id, in.Barcode, in.BatchNo, in.Expiry, in.Qty)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) MarkReceiptVariance(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		Barcode string  `json:"barcode"`
		Reason  string  `json:"reason"`
		Qty     float64 `json:"qty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).MarkVariance(id, in.Barcode, in.Reason, in.Qty)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) ConfirmReceipt(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	hit, err := h.flow(c).ConfirmReceive(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) PlacePutaway(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		LocationCode string  `json:"locationCode"`
		Barcode      string  `json:"barcode"`
		Qty          float64 `json:"qty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).Place(id, in.LocationCode, in.Barcode, in.Qty)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) MoveStock(c *gin.Context) {
	var in struct {
		FromCode string  `json:"fromCode"`
		Barcode  string  `json:"barcode"`
		ToCode   string  `json:"toCode"`
		Qty      float64 `json:"qty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).Move(in.FromCode, in.Barcode, in.ToCode, in.Qty)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}
