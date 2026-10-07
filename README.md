# Factory Traffic Management System (Nexus Traffic OS)
**CSI Smart Tech - Backend Developer Intern Assessment V2**

An event-driven, safety-critical traffic-management control system for internal roads within an industrial garment manufacturing facility.

Built with **Go 1.24+**, adhering strictly to **Hexagonal Architecture (Ports & Adapters)**, **Actor-Model Concurrency**, **Pure Functional Decision Core**, and **Fail-Safe Industrial Safety Standards**.

---

## 1. Quick Start & Setup Instructions

### Prerequisites
- **Go 1.24+** installed.
- No external CGO compiler or databases required (uses pure-Go SQLite).

### Running the Application
From the repository root (`factory-traffic`):

```powershell
# Run directly with Go:
go run .\cmd\server

# Or compile and execute:
go build -o server.exe .\cmd\server
.\server.exe
```

The server starts on port `8080`, initializing **Junction A** as primary.

### Accessing the Frontend Dashboard
Open your web browser at:
```
http://localhost:8080
```
Or launch automatically in PowerShell:
```powershell
Start-Process "http://localhost:8080"
```

---

## 2. Major Architectural Decisions

The solution is architected as an **event-driven control system**, not a basic CRUD database wrapper:

```
                  [ REST Client / Sensor / Web Dashboard ]
                                    │
                         [ HTTP Transport Layer ]
                                    │
                           [ Application Layer ]
                ┌───────────────────┴───────────────────┐
                ▼                                       ▼
    [ Junction Actor Loop ]                 [ Traffic Service Orchestrator ]
         (Serialized Inbox)                             │
                │                                       │
                ▼                                       ▼
    [ Pure Domain Decision Core ]             [ Ports & Adapters Interfaces ]
        • Invariant Safety Guard                        │
        • Scoring Scheduler                             ├──► [ SQLite Repository ]
        • Phase State Machine                           └──► [ Simulated PLC Controller ]
```

1. **Pure Domain Decision Core (`internal/domain/`)**:
   - **Zero I/O dependency**: `Decide(state, input, now) -> (newState, []Effect, error)`.
   - The domain engine does not import HTTP, SQL, or MQTT packages, allowing exhaustive, instant unit testing without mocks.
2. **Actor-Model Concurrency (`internal/app/actor.go`)**:
   - Each junction runs inside an isolated goroutine with a buffered mailbox (`chan actorEnvelope`).
   - High-frequency sensor triggers, emergency alerts, and administrator commands are serialized sequentially.
   - Eliminates lock contention, data races, and hazardous intermediate states without blocking request handlers with `sleep()`.
3. **Hexagonal Architecture (Ports & Adapters)**:
   - Hardware traffic light PLCs are abstracted behind `ports.ControllerPort`.
   - Persistence is decoupled behind `ports.Repository`.
   - Simulated REST controller (`internal/adapters/simcontroller`) allows testing physical actuation delays, timeouts, and fault injection without hardware.
4. **Separation of Desired vs. Controller-Confirmed Physical State**:
   - The system tracks `desired_signals` (state machine intent) separately from `actual_signals` (confirmed by controller ACK).
   - If physical confirmation fails or times out after 3 retries, the junction degrades safely to Yellow Flashing.
5. **Boot-Time Recovery (`internal/app/recovery.go`)**:
   - Snapshots and audit journals are written to SQLite.
   - Upon reboot, any dangling mid-transition state (e.g. crash during yellow or all-red) is recovered safely to steady clearance before resuming automatic scheduling.

---

## 3. Traffic-Control Algorithm

The automatic scheduling engine balances corridor throughput, vehicle priorities, and starvation prevention using dynamic scoring:

$$\text{PhaseScore} = \sum (\text{VehicleWeights}) + \text{StarvationBonus} + \text{WaitTimeBonus}$$

1. **Vehicle Priorities (Section 4)**:
   - `EMERGENCY`: Weight **100** (instant preemption trigger)
   - `TRUCK`: Weight **5**
   - `FORKLIFT`: Weight **3**
   - `EMPLOYEE_VEHICLE`: Weight **1**
2. **Hysteresis & Anti-Flapping**:
   - An opposing corridor must exceed the active corridor's score by a minimum margin ($\Delta \ge 3$) to prevent rapid flip-flopping.
3. **Starvation Protection**:
   - Any vehicle waiting longer than `StarvationThreshold` (15s) gains a $+10$ starvation bonus per vehicle, preventing high-priority corridors from starving isolated vehicles.
4. **Timing Constraints**:
   - **Minimum Green (`MinGreenDuration = 5s`)**: Cannot switch before min green elapses unless preempted by an emergency vehicle.
   - **Maximum Green (`MaxGreenDuration = 30s`)**: Active phase yields when opposing corridor has pending demand.

---

## 4. Traffic-State Transitions

Signal transitions are governed by an explicit deterministic state machine:

```
[ NORTH/SOUTH GREEN ]
          │ (Opposing demand or Emergency trigger)
          ▼
[ NORTH/SOUTH YELLOW ]   (Clearance: 5 seconds)
          │
          ▼
    [ ALL RED ]          (Clearance: 2 seconds)
          │
          ▼
 [ EAST/WEST GREEN ]     (Active: 5s to 30s)
```

- **Invariant Rule 1 (Mutual Exclusion)**: Conflicting directions (`NORTH/SOUTH` vs `EAST/WEST`) can **never** receive `GREEN` or `YELLOW` concurrently.
- **Invariant Rule 2**: Conflicting green phases can never transition directly without completing the full `YELLOW -> ALL_RED` clearance sequence.
- **Invariant Rule 3 (Failsafe Degraded Mode)**: When hardware reports `NACK` or fails to ACK after 3 retries, all lamps transition to flashing yellow.

---

## 5. Assumptions / Questions / Requirement Issues

As instructed in Sections 17 & 19 of the assessment specification, here are the identified ambiguities, contradictions, and safety hazards, along with the engineering decisions made:

### A. Unclear Requirements
1. **Authoritative Timestamp (Sensor vs. Server)**:
   - *Issue*: Sensors provide timestamps in payloads, but physical sensor clocks drift or deliver delayed packets.
   - *Decision*: The backend treats sensor timestamp as metadata for audit logs, but calculates queue duration and signal state transitions using monotonically increasing server clock (`time.Now()`) to prevent clock skew exploits.
2. **Maximum Waiting Time Calculation**:
   - *Issue*: Unclear if wait time is tracked per vehicle or per lane.
   - *Decision*: Tracked from arrival timestamp of the **oldest waiting vehicle** at the head of each lane queue.

### B. Contradictory Requirements
1. **Emergency Preemption vs. Safety Clearances**:
   - *Contradiction*: The specification states emergency vehicles have absolute priority, but also mandates that safety transitions must never be bypassed.
   - *Decision*: Emergency vehicles trigger an immediate phase change, but the active conflicting phase **must still execute yellow clearance and all-red clearance** to allow vehicles inside the junction box to clear before granting emergency green. Bypassing yellow would cause catastrophic broadside collisions.

### C. Unsafe Requirements
1. **Direct Signal Overrides from API / Frontend**:
   - *Hazard*: Allowing clients to post arbitrary signal lamp colors directly could illuminate conflicting greens.
   - *Decision*: Replaced arbitrary signal writing with a **Command-Oriented API** (`MANUAL_GREEN_REQUEST` or `RETURN_TO_AUTOMATIC`). The domain engine safely plans and enforces the transition sequence.
2. **Stale / Phantom Emergency Signals**:
   - *Hazard*: If an emergency vehicle breaks down or diverts before reaching the intersection, the junction could stay green indefinitely.
   - *Decision*: Implemented `EmergencyTTL = 45s`. If not cleared within the TTL window, emergency mode automatically lapses back to automatic mode.

### D. Missing Requirements
1. **Competing Emergency Vehicles from Conflicting Directions**:
   - *Decision*: Priority follows First-Come-First-Served (FCFS) based on arrival timestamp. Once the first emergency vehicle clears, the conflicting emergency vehicle is served next.
2. **Manual Override Expiration**:
   - *Decision*: Manual hold defaults to 30 seconds (configurable up to 60s max) and auto-reverts to `AUTOMATIC` if the operator disconnects or fails to release it.

### E. Technically Problematic Areas
1. **Sensor Bouncing & Duplicate Submissions**:
   - *Issue*: Inductive loops and RFID scanners frequently report duplicate reads or repeated `event_id`s.
   - *Decision*: Implemented two-tiered deduplication:
     - Exact `event_id` idempotency token cache (Section 4).
     - Temporal vehicle deduplication window (3s) to discard RFID tag contact bounce.
2. **Out-of-Order Clearances**:
   - *Issue*: `VEHICLE_CLEARED` event arriving without a matching prior `VEHICLE_ARRIVED`.
   - *Decision*: If the vehicle ID is in queue, it is dequeued; if not found, the event is logged as an audit anomaly without decrementing queue below 0.

### F. Dependent on Business Decision
1. **Priority Hierarchy Weights**:
   - Assigned `EMERGENCY=100`, `TRUCK=5`, `FORKLIFT=3`, `EMPLOYEE_VEHICLE=1` to ensure heavy transport flow while preventing employee transport starvation.

---

## 6. Instructions for Demonstrating the Main Scenarios

All 9 assessment scenarios (Section 15) can be demonstrated using the curl commands in [`docs/demo.md`](docs/demo.md):

1. **Normal Traffic**: Autonomous cycling between North/South and East/West upon vehicle detection.
2. **Priority Traffic**: Demonstrating heavy `TRUCK` priority advancement over `EMPLOYEE_VEHICLE`.
3. **Emergency Preemption**: Emergency beacon triggering safe yellow/all-red sequence to clear priority corridor.
4. **Manual Override**: Administrator forcing corridor hold and returning to automatic control.
5. **Duplicate Event (Idempotency)**: Re-submitting identical `event_id` without corrupting queue size.
6. **Vehicle Clearance**: Arrived vehicle clearing and decrementing queue safely.
7. **Controller Failure**: Injecting simulated NACK / timeout causing failsafe Degraded yellow flash.
8. **Restart & Boot Recovery**: Re-launching server to verify state persistence and transition recovery.
9. **Concurrent Events**: Firing simultaneous arrivals and overrides, confirming zero race conditions.

---

## 7. Running Automated Tests

Run the complete test suite across domain logic, actor concurrency, and SQLite persistence:

```powershell
go test -v ./...
```

To view coverage metrics:
```powershell
go test -cover ./...
```

---

## 8. AI / Tool Usage

In compliance with Section 18.12 of the assessment instructions:
- **Tools Used**: AI Coding Assistant (Antigravity).
- **Purpose**: Assisting with initial architectural boilerplate generation, typing Go domain models, formatting HTML/CSS for the dashboard, and generating exhaustive test fixtures.
- **Verification**: All architectural decisions, safety invariants, actor loops, deduplication mechanics, and API endpoint contracts were verified, debugged, and tested through unit tests and PowerShell terminal executions.
