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

// ListZones handles GET /api/dns/zones?server_id=...
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
	zone, err := h.svc.GetZone(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "DNS zone not found: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, zone)
}

// GetRecords handles GET /api/dns/zones/{id}/records
func (h *DNSHandler) GetRecords(w http.ResponseWriter, r *http.Request) {
	zoneID := chi.URLParam(r, "id")
	records, err := h.svc.GetRecords(r.Context(), zoneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get DNS records: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, records)
}

type createDNSZoneRequest struct {
	ServerID   string `json:"server_id"`
	Name       string `json:"name"`
	ZoneType   string `json:"zone_type"`
	MasterIP   string `json:"master_ip"`
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
	if req.ZoneType == "" {
		req.ZoneType = "master"
	}
	if req.ZoneType != "master" && req.ZoneType != "slave" {
		writeError(w, http.StatusBadRequest, "zone_type must be 'master' or 'slave'")
		return
	}
	if req.ZoneType == "slave" && req.MasterIP == "" {
		writeError(w, http.StatusBadRequest, "master_ip is required for slave zones")
		return
	}
	if req.Nameserver == "" {
		req.Nameserver = "ns1." + req.Name + "."
	}
	if req.AdminEmail == "" {
		req.AdminEmail = "admin@" + req.Name
	}

	zone, err := h.svc.CreateZone(r.Context(), req.ServerID, req.Name, req.Nameserver, req.AdminEmail, req.ZoneType, req.MasterIP)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create DNS zone: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, zone)
}

// DeleteZone handles DELETE /api/dns/zones/{id}
func (h *DNSHandler) DeleteZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteZone(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete DNS zone: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	var req addDNSRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Type == "" || req.Content == "" {
		writeError(w, http.StatusBadRequest, "name, type and content are required")
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
	if err := h.svc.DeleteRecord(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete DNS record: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UpdateRecord handles PUT /api/dns/records/{id}
func (h *DNSHandler) UpdateRecord(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Content  string `json:"content"`
		TTL      int    `json:"ttl"`
		Priority int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Type == "" || req.Content == "" {
		writeError(w, http.StatusBadRequest, "name, type and content are required")
		return
	}
	record, err := h.svc.UpdateRecord(r.Context(), id, req.Name, req.Type, req.Content, req.TTL, req.Priority)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update DNS record: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

// ApplyTemplate handles POST /api/dns/zones/{id}/apply-template
// Applies the standard Plesk-style DNS template to an existing zone (idempotent).
func (h *DNSHandler) ApplyTemplate(w http.ResponseWriter, r *http.Request) {
	zoneID := chi.URLParam(r, "id")
	records, err := h.svc.ApplyTemplate(r.Context(), zoneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to apply DNS template: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, records)
}
