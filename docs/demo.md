# Demonstration Scripts: 9 Factory Traffic Operational Scenarios (Section 15)

This guide provides executable `curl` commands verifying the 9 functional scenarios defined in Section 15 of the assessment specification.

Ensure the server is running locally:
```powershell
go run .\cmd\server
# or run the compiled binary:
.\server.exe
```

---

### Scenario 1: Normal Traffic
**Objective**: Several vehicles arrive from different directions and the system determines which phase to serve.

```bash
# 1. Query baseline state of Junction A
curl -s http://localhost:8080/api/junctions/A/status

# 2. Vehicle arrives on EAST approach
curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-1001",
    "junction_id": "A",
    "direction": "EAST",
    "event_type": "VEHICLE_ARRIVED",
    "vehicle_id": "VH-501",
    "vehicle_type": "FORKLIFT",
    "sequence_no": 1,
    "timestamp": "2026-10-08T02:00:00Z"
  }'

# 3. Observe safe transition: North/South YELLOW (5s) -> ALL_RED (2s) -> East/West GREEN
sleep 8
curl -s http://localhost:8080/api/junctions/A/status
```

---

### Scenario 2: Priority Traffic
**Objective**: A TRUCK or FORKLIFT affects scheduling compared with ordinary employee traffic.

```bash
# 1. Employee transport vehicle arrives on SOUTH (Priority weight 1)
curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-1002",
    "junction_id": "A",
    "direction": "SOUTH",
    "event_type": "VEHICLE_ARRIVED",
    "vehicle_id": "EMP-CAR-01",
    "vehicle_type": "EMPLOYEE_VEHICLE",
    "sequence_no": 2
  }'

# 2. Heavy delivery TRUCK arrives on EAST (Priority weight 5)
curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-1003",
    "junction_id": "A",
    "direction": "EAST",
    "event_type": "VEHICLE_ARRIVED",
    "vehicle_id": "TRUCK-99",
    "vehicle_type": "TRUCK",
    "sequence_no": 3
  }'

# TRUCK with higher priority score influences phase transition preference to East/West
curl -s http://localhost:8080/api/junctions/A/status
```

---

### Scenario 3: Emergency Preemption
**Objective**: An emergency vehicle arrives while a conflicting phase is GREEN. The system executes immediate safe preemption.

```bash
# Emergency vehicle arrives on WEST corridor
curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-1004",
    "junction_id": "A",
    "direction": "WEST",
    "event_type": "VEHICLE_ARRIVED",
    "vehicle_id": "FIRE-TRUCK-01",
    "vehicle_type": "EMERGENCY",
    "sequence_no": 4
  }'

# Backend immediately initiates safe yellow clearance (5s) -> all-red clearance (2s) -> emergency phase GREEN
curl -s http://localhost:8080/api/junctions/A/status

# Clear the emergency vehicle once through
curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-1005",
    "junction_id": "A",
    "direction": "WEST",
    "event_type": "VEHICLE_CLEARED",
    "vehicle_id": "FIRE-TRUCK-01",
    "sequence_no": 5
  }'
```

---

### Scenario 4: Manual Override
**Objective**: An administrator takes control of Junction A and later returns it to automatic mode.

```bash
# 1. Administrator requests manual GREEN for WEST
curl -X POST http://localhost:8080/api/junctions/A/commands \
  -H "Content-Type: application/json" \
  -d '{
    "command": "MANUAL_GREEN_REQUEST",
    "direction": "WEST"
  }'

curl -s http://localhost:8080/api/junctions/A/status

# 2. Return to automatic mode
curl -X POST http://localhost:8080/api/junctions/A/commands \
  -H "Content-Type: application/json" \
  -d '{
    "command": "RETURN_TO_AUTOMATIC"
  }'

curl -s http://localhost:8080/api/junctions/A/status
```

---

### Scenario 5: Duplicate Event (Idempotency)
**Objective**: The exact same sensor event is submitted twice. Queue state must remain correct.

```bash
# Submit event_id evt-dup-101
curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-dup-101",
    "junction_id": "A",
    "direction": "NORTH",
    "event_type": "VEHICLE_ARRIVED",
    "vehicle_id": "FORK-42",
    "vehicle_type": "FORKLIFT",
    "sequence_no": 10
  }'

# Repeat same event_id: idempotent rejection ensures queue does not increment twice
curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-dup-101",
    "junction_id": "A",
    "direction": "NORTH",
    "event_type": "VEHICLE_ARRIVED",
    "vehicle_id": "FORK-42",
    "vehicle_type": "FORKLIFT",
    "sequence_no": 10
  }'

curl -s http://localhost:8080/api/junctions/A/status
```

---

### Scenario 6: Vehicle Clearance
**Objective**: A vehicle arrives and later sends a `VEHICLE_CLEARED` event. Queue state changes correctly.

```bash
# 1. Arrival
curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-clear-1",
    "junction_id": "A",
    "direction": "NORTH",
    "event_type": "VEHICLE_ARRIVED",
    "vehicle_id": "CLEAR-TEST-VH",
    "vehicle_type": "TRUCK",
    "sequence_no": 15
  }'

# Check queue increased
curl -s http://localhost:8080/api/junctions/A/status

# 2. Clearance
curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-clear-2",
    "junction_id": "A",
    "direction": "NORTH",
    "event_type": "VEHICLE_CLEARED",
    "vehicle_id": "CLEAR-TEST-VH",
    "sequence_no": 16
  }'

# Check queue decreased
curl -s http://localhost:8080/api/junctions/A/status
```

---

### Scenario 7: Controller Failure & Failsafe
**Objective**: Controller reports NACK rejection or fails to acknowledge. Backend responds safely by entering DEGRADED yellow flashing mode.

```bash
# 1. Inject simulated controller NACK failure
curl -X POST http://localhost:8080/api/sim/fault \
  -H "Content-Type: application/json" \
  -d '{"simulate_nack": true}'

# 2. Trigger arrival requiring signal change
curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-fault-trigger",
    "junction_id": "A",
    "direction": "EAST",
    "event_type": "VEHICLE_ARRIVED",
    "vehicle_id": "FAULT-TEST",
    "vehicle_type": "TRUCK",
    "sequence_no": 20
  }'

sleep 1
# Verify junction entered DEGRADED mode
curl -s http://localhost:8080/api/junctions/A/status

# Restore controller
curl -X POST http://localhost:8080/api/sim/fault \
  -H "Content-Type: application/json" \
  -d '{"simulate_nack": false, "healthy": true}'
```

---

### Scenario 8: Restart & Boot Recovery
**Objective**: Backend restarts while persisted data exists. Important state and history remain available.

```bash
# 1. State snapshot is saved after every event
curl -s http://localhost:8080/api/junctions/A/status

# 2. Restart server process:
# The server reloads SQLite state and recovers cleanly without assuming stale physical states.

# 3. Check history
curl -s "http://localhost:8080/api/junctions/A/history?limit=10"
```

---

### Scenario 9: Concurrent Events
**Objective**: Events affecting the same junction occur almost simultaneously (T=0, T=4ms, T=8ms, T=12ms, T=17ms).

```bash
# In concurrent event processing, the per-junction actor mailbox serializes decisions
# and enforces that conflicting phases NEVER receive simultaneous GREEN.
curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{"event_id": "seq-1", "junction_id": "A", "direction": "NORTH", "event_type": "VEHICLE_ARRIVED", "vehicle_id": "C-TRUCK", "vehicle_type": "TRUCK", "sequence_no": 1}' &

curl -X POST http://localhost:8080/api/sensor-events \
  -H "Content-Type: application/json" \
  -d '{"event_id": "seq-2", "junction_id": "A", "direction": "EAST", "event_type": "VEHICLE_ARRIVED", "vehicle_id": "C-EMERG", "vehicle_type": "EMERGENCY", "sequence_no": 2}' &

curl -X POST http://localhost:8080/api/junctions/A/commands \
  -H "Content-Type: application/json" \
  -d '{"command": "MANUAL_GREEN_REQUEST", "direction": "WEST"}' &

wait
curl -s http://localhost:8080/api/junctions/A/status
```
