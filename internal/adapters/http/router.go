package http

import (
	"net/http"
	"strings"
)

// NewRouter constructs the HTTP handler hierarchy, routing both REST APIs and the Web Dashboard.
func NewRouter(h *Handler, staticDir string) http.Handler {
	mux := http.NewServeMux()

	// Static Dashboard Assets
	if staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", fs)
	}

	// API Routing Dispatcher
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS for testing & modern web integrations
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		path := strings.TrimSuffix(r.URL.Path, "/")

		// Section 10.2: Sensor Events
		if path == "/api/sensor-events" && r.Method == http.MethodPost {
			h.PostSensorEvent(w, r)
			return
		}

		// Section 10.5: Controller Events / Acknowledgements
		if path == "/api/controller-events" && r.Method == http.MethodPost {
			h.PostControllerEvent(w, r)
			return
		}

		// Section 10.1: List Junctions
		if path == "/api/junctions" && r.Method == http.MethodGet {
			h.GetJunctions(w, r)
			return
		}

		// Simulation endpoints
		if path == "/api/sim/fault" && r.Method == http.MethodPost {
			h.PostSimFault(w, r)
			return
		}
		if path == "/api/sim/signals" && r.Method == http.MethodGet {
			h.GetSimSignals(w, r)
			return
		}

		// Junction endpoints: /api/junctions/{id}/*
		if strings.HasPrefix(path, "/api/junctions/") {
			parts := strings.Split(strings.TrimPrefix(path, "/api/junctions/"), "/")
			if len(parts) == 1 {
				// GET /api/junctions/{id}
				if r.Method == http.MethodGet {
					h.GetStatus(w, r)
					return
				}
			} else if len(parts) >= 2 {
				action := parts[1]
				switch action {
				case "status":
					// Section 10.3: GET /api/junctions/:id/status
					if r.Method == http.MethodGet {
						h.GetStatusSection10(w, r)
						return
					}
				case "commands":
					// Section 10.4: POST /api/junctions/:id/commands
					if r.Method == http.MethodPost {
						h.PostJunctionCommands(w, r)
						return
					}
				case "history":
					// Section 10.6: GET /api/junctions/:id/history
					if r.Method == http.MethodGet {
						h.GetHistory(w, r)
						return
					}
				case "arrivals":
					if r.Method == http.MethodPost {
						h.PostArrival(w, r)
						return
					}
				case "cleared":
					if r.Method == http.MethodPost {
						h.PostCleared(w, r)
						return
					}
				case "emergency":
					if len(parts) == 3 && parts[2] == "clear" && r.Method == http.MethodPost {
						h.PostClearEmergency(w, r)
						return
					}
					if r.Method == http.MethodPost {
						h.PostEmergency(w, r)
						return
					}
				case "manual":
					if r.Method == http.MethodPost {
						h.PostManualOverride(w, r)
						return
					}
					if r.Method == http.MethodDelete {
						h.DeleteManualOverride(w, r)
						return
					}
				case "audit":
					if r.Method == http.MethodGet {
						h.GetAuditLogs(w, r)
						return
					}
				}
			}
		}

		http.NotFound(w, r)
	})

	return mux
}
