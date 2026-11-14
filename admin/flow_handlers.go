package admin

import (
	"net/http"
	"strconv"

	"warehousecore/internal/pkg/httputil"
	"warehousecore/internal/pkg/response"
	"warehousecore/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) ListReceipts(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	whID, _ := strconv.ParseUint(c.Query("warehouseId"), 10, 64)
	list, total, err := h.flow(c).ListReceipts(c.Query("keyword"), c.Query("status"), whID, page, pageSize)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) GetReceipt(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.flow(c).GetReceipt(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) ListPutaways(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	whID, _ := strconv.ParseUint(c.Query("warehouseId"), 10, 64)
	list, total, err := h.flow(c).ListPutaways(c.Query("keyword"), c.Query("status"), whID, page, pageSize)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) ListOffshelves(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	whID, _ := strconv.ParseUint(c.Query("warehouseId"), 10, 64)
	list, total, err := h.flow(c).ListOffshelves(c.Query("keyword"), whID, page, pageSize)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) ListPicks(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	whID, _ := strconv.ParseUint(c.Query("warehouseId"), 10, 64)
	list, total, err := h.flow(c).ListPicks(c.Query("keyword"), c.Query("status"), c.Query("stage"), whID, page, pageSize)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *Handlers) GetPick(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.flow(c).GetDeskPick(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) EnsureSelfPick(c *gin.Context) {
	var in service.SelfPickIn
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.flow(c).EnsureSelfPick(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) PdaTasks(c *gin.Context) {
	board, err := h.flow(c).Tasks(c.Query("bucket"), c.Query("keyword"))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, board)
}

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

func (h *Handlers) ScanPick(c *gin.Context) {
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
	hit, err := h.flow(c).ScanPick(id, in.LocationCode, in.Barcode, in.Qty)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) ReviewPick(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		Barcode string `json:"barcode"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).ReviewPick(id, in.Barcode)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) HandoverPick(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		MailNo   string `json:"mailNo"`
		PhotoURL string `json:"photoUrl"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).HandoverPick(id, in.MailNo, in.PhotoURL)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) RecheckPick(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	hit, err := h.flow(c).RecheckPick(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) GradeReturn(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		Barcode string `json:"barcode"`
		Grade   string `json:"grade"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).GradeReturn(id, in.Barcode, in.Grade)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) PlaceReturn(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		LocationCode string  `json:"locationCode"`
		Barcode      string  `json:"barcode"`
		Grade        string  `json:"grade"`
		Qty          float64 `json:"qty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).PlaceReturn(id, in.LocationCode, in.Barcode, in.Grade, in.Qty)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) StartCount(c *gin.Context) {
	var in struct {
		LocationCode string `json:"locationCode"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).StartCount(in.LocationCode)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) ScanCount(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		Barcode string  `json:"barcode"`
		Qty     float64 `json:"qty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).ScanCount(id, in.Barcode, in.Qty)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) SubmitCount(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	hit, err := h.flow(c).SubmitCount(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) ScanPdaTransfer(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		LocationCode string  `json:"locationCode"`
		Barcode      string  `json:"barcode"`
		Inbound      bool    `json:"inbound"`
		Qty          float64 `json:"qty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	hit, err := h.flow(c).ScanTransfer(id, in.LocationCode, in.Barcode, in.Inbound, in.Qty)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, hit)
}

func (h *Handlers) ConfirmPdaTransfer(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		Inbound bool `json:"inbound"`
	}
	_ = c.ShouldBindJSON(&in)
	hit, err := h.flow(c).ConfirmTransfer(id, in.Inbound)
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
