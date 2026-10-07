# Factory Traffic Controller API Reference (Assessment V2 Specification)

The **Nexus Factory Traffic System** strictly implements the REST API specifications from Section 10 of the Backend Developer Assessment V2.

Base URL: `http://localhost:8080/api`

---

### 10.1 Junctions
- **List Junctions**: `GET /api/junctions`
  - Response:
    ```json
    {
      "success": true,
      "data": ["A", "junction-1"]
    }
    ```

- **Get Junction**: `GET /api/junctions/:id`
  - Returns current telemetry, signals, queues, and diagnostics.

---

### 10.2 Sensor Events
Sensors near the junction send vehicle detection events to the backend when vehicles arrive or clear.
- **Endpoint**: `POST /api/sensor-events`
- **Vehicle Arrived Example**:
  ```json
  {
    "event_id": "evt-10001",
    "junction_id": "A",
    "direction": "NORTH",
    "event_type": "VEHICLE_ARRIVED",
    "vehicle_id": "VH-501",
    "vehicle_type": "TRUCK",
    "sequence_no": 1501,
    "timestamp": "2026-10-05T10:15:20Z"
  }
  ```
- **Vehicle Cleared Example**:
  ```json
  {
    "event_id": "evt-10002",
    "junction_id": "A",
    "direction": "NORTH",
    "event_type": "VEHICLE_CLEARED",
    "vehicle_id": "VH-501",
    "sequence_no": 1502,
    "timestamp": "2026-10-05T10:16:10Z"
  }
  ```
- **Supported `vehicle_type`**: `EMERGENCY` (Priority 100) > `TRUCK` (Priority 5) > `FORKLIFT` (Priority 3) > `EMPLOYEE_VEHICLE` (Priority 1).
- **Idempotency**: Submitting the same `event_id` multiple times will not modify queue state multiple times.

---

### 10.3 Junction Status
Get the real-time operational state, active signals, approach queues, and transition diagnostics of a junction.
- **Endpoint**: `GET /api/junctions/:id/status`
- **Response**:
  ```json
  {
    "junction_id": "A",
    "mode": "AUTOMATIC",
    "phase": "NORTH_SOUTH",
    "controller_status": "ONLINE",
    "desired_signals": {
      "NORTH": "GREEN",
      "SOUTH": "GREEN",
      "EAST": "RED",
      "WEST": "RED"
    },
    "actual_signals": {
      "NORTH": "GREEN",
      "SOUTH": "GREEN",
      "EAST": "RED",
      "WEST": "RED"
    },
    "queues": {
      "NORTH": 3,
      "SOUTH": 1,
      "EAST": 7,
      "WEST": 2
    }
  }
  ```

---

### 10.4 Manual / Control Commands
Allows factory administrators to request manual traffic control or return to automatic mode.
- **Endpoint**: `POST /api/junctions/:id/commands`
- **Request Green for Direction**:
  ```json
  {
    "command": "MANUAL_GREEN_REQUEST",
    "direction": "WEST"
  }
  ```
- **Return to Automatic**:
  ```json
  {
    "command": "RETURN_TO_AUTOMATIC"
  }
  ```

---

### 10.5 Controller Events / Acknowledgements
Simulates physical traffic-controller hardware responses and acknowledgements.
- **Endpoint**: `POST /api/controller-events`
- **Payload**:
  ```json
  {
    "command_id": "cmd-8001",
    "junction_id": "A",
    "status": "ACK",
    "actual_state": "GREEN"
  }
  ```

---

### 10.6 History / Audit Log
Retrieves the history of important system events (transitions, arrivals, clearances, emergency preemption, and failures).
- **Endpoint**: `GET /api/junctions/:id/history`
- **Query Params**: `limit` (default: 50)
- **Response**:
  ```json
  [
    {
      "junction_id": "A",
      "event_type": "TRANSITION_GREEN_ACTIVATED",
      "phase": "PHASE_NORTH_SOUTH",
      "mode": "AUTOMATIC",
      "details": { "active_phase": "PHASE_NORTH_SOUTH" },
      "timestamp": "2026-10-05T10:20:00Z"
    }
  ]
  ```

---

### Additional Simulation Endpoints
- **Inject Controller Faults**: `POST /api/sim/fault` (`simulate_nack`, `drop_ack`, `latency_ms`)
- **Query Physical Lamp States**: `GET /api/sim/signals`
