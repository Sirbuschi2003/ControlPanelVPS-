package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
)

// DNSHandler handles HTTP requests for DNS zone and record management.
type DNSHandler struct {
	svc *services.DNSService
}

// NewDNSHandler creates a new DNSHandler.
func NewDNSHandler(svc *services.DNSService) *DNSHandler {
	return &DNSHandler{svc: svc}
}

// ListZones handles GET /api/dns/zones?server_id=... (server_id optional, returns all when omitted)
func (h *DNSHandler) ListZones(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	zones, err := h.svc.ListZones(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list DNS zones: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, zones)
}

// GetZone handles GET /api/dns/zones/{id}
func (h *DNSHandler) GetZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	zone, err := h.svc.GetZone(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "DNS zone not found: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, zone)
}

type createDNSZoneRequest struct {
	ServerID   string `json:"server_id"`
	Name       string `json:"name"`
	Nameserver string `json:"nameserver"`
	AdminEmail string `json:"admin_email"`
}

// CreateZone handles POST /api/dns/zones
func (h *DNSHandler) CreateZone(w http.ResponseWriter, r *http.Request) {
	var req createDNSZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Nameserver == "" {
		req.Nameserver = "ns1." + req.Name + "."
	}
	if req.AdminEmail == "" {
		req.AdminEmail = "admin@" + req.Name
	}

	zone, err := h.svc.CreateZone(r.Context(), req.ServerID, req.Name, req.Nameserver, req.AdminEmail)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create DNS zone: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, zone)
}

// DeleteZone handles DELETE /api/dns/zones/{id}
func (h *DNSHandler) DeleteZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.DeleteZone(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete DNS zone: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "DNS zone deleted"})
}

type addDNSRecordRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}

// AddRecord handles POST /api/dns/zones/{id}/records
func (h *DNSHandler) AddRecord(w http.ResponseWriter, r *http.Request) {
	zoneID := chi.URLParam(r, "id")
	if zoneID == "" {
		writeError(w, http.StatusBadRequest, "zone id is required")
		return
	}

	var req addDNSRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if req.TTL == 0 {
		req.TTL = 3600
	}

	record, err := h.svc.AddRecord(r.Context(), zoneID, req.Name, req.Type, req.Content, req.TTL, req.Priority)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add DNS record: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, record)
}

// DeleteRecord handles DELETE /api/dns/records/{id}
func (h *DNSHandler) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.svc.DeleteRecord(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete DNS record: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "DNS record deleted"})
}
