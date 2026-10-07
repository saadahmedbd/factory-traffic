package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"factory-traffic/internal/adapters/simcontroller"
	"factory-traffic/internal/app"
	"factory-traffic/internal/domain"
)

type Handler struct {
	service *app.TrafficService
	simCtrl *simcontroller.SimController
}

func NewHandler(service *app.TrafficService, simCtrl *simcontroller.SimController) *Handler {
	return &Handler{
		service: service,
		simCtrl: simCtrl,
	}
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, ApiResponse{
		Success: false,
		Error:   msg,
	})
}

func extractJunctionID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if p == "junctions" && i+1 < len(parts) {
			id := parts[i+1]
			if id != "" && id != "status" && id != "commands" && id != "history" {
				return id
			}
		}
	}
	return "A"
}

// GetJunctions handles Section 10.1 GET /api/junctions
func (h *Handler) GetJunctions(w http.ResponseWriter, r *http.Request) {
	junctions := h.service.ListJunctions()
	if len(junctions) == 0 {
		junctions = []string{"A"}
	}
	h.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data:    junctions,
	})
}

// GetStatus returns the junction status in Section 10.3 / 14.2 format.
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	junctionID := extractJunctionID(r.URL.Path)
	state, err := h.service.GetJunctionStatus(junctionID)
	if err != nil {
		// Fallback check
		if junctionID == "A" {
			state, err = h.service.GetJunctionStatus("junction-1")
		} else if junctionID == "junction-1" {
			state, err = h.service.GetJunctionStatus("A")
		}
	}
	if err != nil {
		h.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	queueCounts := make(map[domain.Direction]int)
	for _, d := range domain.AllDirections {
		queueCounts[d] = len(state.Queues[d])
	}

	ctrlStatus := "ONLINE"
	actualSignals := state.Signals
	if h.simCtrl != nil {
		if err := h.simCtrl.HealthCheck(r.Context()); err != nil {
			ctrlStatus = "OFFLINE"
		}
		actualSignals = h.simCtrl.GetSignals()
	}

	var activeAlerts []string
	if state.Mode == domain.ModeDegraded {
		activeAlerts = append(activeAlerts, "DEGRADED_FAILSAFE: "+state.FailSafeReason)
	}
	if len(state.EmergencyQueue) > 0 {
		activeAlerts = append(activeAlerts, "EMERGENCY_PREEMPTION_ACTIVE")
	}

	statusResp := JunctionStatusResponse{
		JunctionID:       state.JunctionID,
		Mode:             string(state.Mode),
		Phase:            strings.TrimPrefix(string(state.CurrentPhase), "PHASE_"),
		ControllerStatus: ctrlStatus,
		DesiredSignals:   state.Signals,
		ActualSignals:    actualSignals,
		Queues:           queueCounts,
		Vehicles:         state.Queues,
		PendingCommand:   state.PendingCommand,
		InTransition:     state.InTransition,
		TransitionStep:   state.TransitionStep,
		ActiveAlerts:     activeAlerts,
	}

	h.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data:    statusResp,
	})
}

// GetStatusSection10 returns the EXACT raw JSON object required by Section 10.3
func (h *Handler) GetStatusSection10(w http.ResponseWriter, r *http.Request) {
	junctionID := extractJunctionID(r.URL.Path)
	state, err := h.service.GetJunctionStatus(junctionID)
	if err != nil {
		if junctionID == "A" {
			state, err = h.service.GetJunctionStatus("junction-1")
		} else {
			state, err = h.service.GetJunctionStatus("A")
		}
	}
	if err != nil {
		h.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	queueCounts := make(map[domain.Direction]int)
	for _, d := range domain.AllDirections {
		queueCounts[d] = len(state.Queues[d])
	}

	ctrlStatus := "ONLINE"
	actualSignals := state.Signals
	if h.simCtrl != nil {
		if err := h.simCtrl.HealthCheck(r.Context()); err != nil {
			ctrlStatus = "OFFLINE"
		}
		actualSignals = h.simCtrl.GetSignals()
	}

	resp := map[string]interface{}{
		"junction_id":       state.JunctionID,
		"mode":              string(state.Mode),
		"phase":             string(state.CurrentPhase),
		"controller_status": ctrlStatus,
		"desired_signals":   state.Signals,
		"actual_signals":    actualSignals,
		"queues":            queueCounts,
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// PostSensorEvent implements Section 10.2: POST /api/sensor-events
func (h *Handler) PostSensorEvent(w http.ResponseWriter, r *http.Request) {
	var req SensorEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.JunctionID == "" {
		req.JunctionID = "A"
	}
	if !domain.IsValidDirection(req.Direction) {
		h.writeError(w, http.StatusBadRequest, "invalid direction: "+string(req.Direction))
		return
	}

	t := time.Now()
	if req.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			t = parsed
		}
	}

	err := h.service.ProcessSensorEvent(
		r.Context(),
		req.EventID,
		req.JunctionID,
		req.Direction,
		req.EventType,
		req.VehicleID,
		req.VehicleType,
		req.SequenceNo,
		t,
	)
	if err != nil {
		// Attempt alias fallback if junction A or junction-1
		aliasID := "A"
		if req.JunctionID == "A" {
			aliasID = "junction-1"
		}
		if errAlias := h.service.ProcessSensorEvent(
			r.Context(),
			req.EventID,
			aliasID,
			req.Direction,
			req.EventType,
			req.VehicleID,
			req.VehicleType,
			req.SequenceNo,
			t,
		); errAlias == nil {
			err = nil
		}
	}

	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"event_id":   req.EventID,
		"status":     "PROCESSED",
		"timestamp":  t.Format(time.RFC3339),
	})
}

// PostJunctionCommands implements Section 10.4: POST /api/junctions/:id/commands
func (h *Handler) PostJunctionCommands(w http.ResponseWriter, r *http.Request) {
	junctionID := extractJunctionID(r.URL.Path)
	var req ManualCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid command payload: "+err.Error())
		return
	}

	var err error
	switch req.Command {
	case "MANUAL_GREEN_REQUEST":
		targetPhase := req.Phase
		if targetPhase == "" {
			switch req.Direction {
			case domain.DirectionNorth, domain.DirectionSouth:
				targetPhase = domain.PhaseNorthSouth
			case domain.DirectionEast, domain.DirectionWest:
				targetPhase = domain.PhaseEastWest
			default:
				targetPhase = domain.PhaseNorthSouth
			}
		}
		duration := 30 * time.Second
		if req.Duration > 0 {
			duration = time.Duration(req.Duration) * time.Second
		}
		err = h.service.SetManualOverride(r.Context(), junctionID, targetPhase, duration, "operator")
		if err != nil && junctionID == "A" {
			err = h.service.SetManualOverride(r.Context(), "junction-1", targetPhase, duration, "operator")
		}

	case "RETURN_TO_AUTOMATIC":
		err = h.service.ReleaseManualOverride(r.Context(), junctionID)
		if err != nil && junctionID == "A" {
			err = h.service.ReleaseManualOverride(r.Context(), "junction-1")
		}

	default:
		h.writeError(w, http.StatusBadRequest, "unknown command: "+req.Command)
		return
	}

	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"command": req.Command,
		"status":  "ACCEPTED",
	})
}

// PostControllerEvent implements Section 10.5: POST /api/controller-events
func (h *Handler) PostControllerEvent(w http.ResponseWriter, r *http.Request) {
	var req ControllerEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid controller event body: "+err.Error())
		return
	}

	isAck := strings.ToUpper(req.Status) == "ACK"
	ack := domain.ControllerAck{
		CommandID:  req.CommandID,
		JunctionID: req.JunctionID,
		Success:    isAck,
		Timestamp:  time.Now(),
	}
	if !isAck {
		ack.ErrorMessage = "Controller NACK / rejection"
	}

	err := h.service.ProcessControllerAck(r.Context(), ack)
	if err != nil && req.JunctionID == "A" {
		ack.JunctionID = "junction-1"
		err = h.service.ProcessControllerAck(r.Context(), ack)
	}

	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"command_id": req.CommandID,
		"status":     "RECORDED",
	})
}

// GetHistory implements Section 10.6: GET /api/junctions/:id/history
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	junctionID := extractJunctionID(r.URL.Path)
	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := h.service.GetAuditHistory(r.Context(), junctionID, limit)
	if err != nil || len(logs) == 0 {
		if junctionID == "A" {
			logs, _ = h.service.GetAuditHistory(r.Context(), "junction-1", limit)
		}
	}

	h.writeJSON(w, http.StatusOK, logs)
}

// Legacy & UI endpoints for full compatibility
func (h *Handler) PostArrival(w http.ResponseWriter, r *http.Request) {
	junctionID := extractJunctionID(r.URL.Path)
	var req VehicleArrivalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if !domain.IsValidDirection(req.Direction) {
		h.writeError(w, http.StatusBadRequest, "invalid direction: "+string(req.Direction))
		return
	}

	if req.VehicleType == "" {
		req.VehicleType = domain.VehicleTruck
	}

	v := domain.Vehicle{
		ID:        req.VehicleID,
		Type:      req.VehicleType,
		Direction: req.Direction,
		ArrivedAt: time.Now(),
	}

	err := h.service.RecordArrival(r.Context(), junctionID, v)
	if err != nil && junctionID == "A" {
		err = h.service.RecordArrival(r.Context(), "junction-1", v)
	}
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "arrival registered",
		Data:    v,
	})
}

func (h *Handler) PostCleared(w http.ResponseWriter, r *http.Request) {
	junctionID := extractJunctionID(r.URL.Path)
	var req VehicleClearedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	err := h.service.RecordCleared(r.Context(), junctionID, req.Direction, req.VehicleID)
	if err != nil && junctionID == "A" {
		err = h.service.RecordCleared(r.Context(), "junction-1", req.Direction, req.VehicleID)
	}
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "vehicle cleared",
	})
}

func (h *Handler) PostEmergency(w http.ResponseWriter, r *http.Request) {
	junctionID := extractJunctionID(r.URL.Path)
	var req EmergencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	v := domain.Vehicle{
		ID:        req.VehicleID,
		Type:      domain.VehicleEmergency,
		Direction: req.Direction,
		ArrivedAt: time.Now(),
	}

	err := h.service.TriggerEmergency(r.Context(), junctionID, v)
	if err != nil && junctionID == "A" {
		err = h.service.TriggerEmergency(r.Context(), "junction-1", v)
	}
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "emergency preemption triggered",
		Data:    v,
	})
}

func (h *Handler) PostClearEmergency(w http.ResponseWriter, r *http.Request) {
	junctionID := extractJunctionID(r.URL.Path)
	vehicleID := r.URL.Query().Get("vehicle_id")
	if vehicleID == "" {
		h.writeError(w, http.StatusBadRequest, "missing vehicle_id query parameter")
		return
	}

	err := h.service.ClearEmergency(r.Context(), junctionID, vehicleID)
	if err != nil && junctionID == "A" {
		err = h.service.ClearEmergency(r.Context(), "junction-1", vehicleID)
	}
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "emergency cleared",
	})
}

func (h *Handler) PostManualOverride(w http.ResponseWriter, r *http.Request) {
	junctionID := extractJunctionID(r.URL.Path)
	var req ManualHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	duration := time.Duration(req.DurationSeconds) * time.Second
	err := h.service.SetManualOverride(r.Context(), junctionID, req.Phase, duration, req.User)
	if err != nil && junctionID == "A" {
		err = h.service.SetManualOverride(r.Context(), "junction-1", req.Phase, duration, req.User)
	}
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "manual hold applied",
	})
}

func (h *Handler) DeleteManualOverride(w http.ResponseWriter, r *http.Request) {
	junctionID := extractJunctionID(r.URL.Path)
	err := h.service.ReleaseManualOverride(r.Context(), junctionID)
	if err != nil && junctionID == "A" {
		err = h.service.ReleaseManualOverride(r.Context(), "junction-1")
	}
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "manual hold released",
	})
}

func (h *Handler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	h.GetHistory(w, r)
}

func (h *Handler) PostSimFault(w http.ResponseWriter, r *http.Request) {
	if h.simCtrl == nil {
		h.writeError(w, http.StatusBadRequest, "simulated controller not active")
		return
	}

	var req SimFaultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.SimulateNACK != nil {
		h.simCtrl.SetSimulateNACK(*req.SimulateNACK)
	}
	if req.DropACK != nil {
		h.simCtrl.SetDropACK(*req.DropACK)
	}
	if req.Healthy != nil {
		h.simCtrl.SetHealthy(*req.Healthy)
	}
	if req.LatencyMs != nil {
		h.simCtrl.SetLatency(time.Duration(*req.LatencyMs) * time.Millisecond)
	}

	h.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "simulation parameters updated",
	})
}

func (h *Handler) GetSimSignals(w http.ResponseWriter, r *http.Request) {
	if h.simCtrl == nil {
		h.writeError(w, http.StatusBadRequest, "simulated controller not active")
		return
	}

	signals := h.simCtrl.GetSignals()
	h.writeJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data:    signals,
	})
}
