const JUNCTION_ID = 'A';
const API_BASE = `/api/junctions/${JUNCTION_ID}`;

// DOM Elements
const valMode = document.getElementById('val-mode');
const valPhase = document.getElementById('val-phase');
const valHardware = document.getElementById('val-hardware');
const phaseTimer = document.getElementById('phase-timer');
const diagEmergency = document.getElementById('diag-emergency');
const diagTransition = document.getElementById('diag-transition');
const diagPendingCmd = document.getElementById('diag-pending-cmd');
const diagQueueTotal = document.getElementById('diag-queue-total');
const diagSignalsState = document.getElementById('diag-signals-state');
const flowIndicator = document.getElementById('flow-indicator');
const badgeManualStatus = document.getElementById('badge-manual-status');
const journalFeed = document.getElementById('journal-feed');

// Alerts Banner (Sections 14.2 & 14.6)
const alertsBanner = document.getElementById('alerts-banner');
const alertsText = document.getElementById('alerts-text');
const alertsIcon = document.getElementById('alerts-icon');
const alertsSub = document.getElementById('alerts-sub');

// Form & Controls
const dispatchIdInput = document.getElementById('dispatch-id');
const dispatchTypeInput = document.getElementById('dispatch-type');
const dispatchDirInput = document.getElementById('dispatch-dir');
const btnDispatch = document.getElementById('btn-dispatch');
const btnQuickEmergency = document.getElementById('btn-quick-emergency');
const btnClearFirst = document.getElementById('btn-clear-first');

const manualPhaseSelect = document.getElementById('manual-phase-select');
const manualDurationRange = document.getElementById('manual-duration-range');
const lblDuration = document.getElementById('lbl-duration');
const btnApplyManual = document.getElementById('btn-apply-manual');
const btnReleaseManual = document.getElementById('btn-release-manual');

const btnFaultNack = document.getElementById('btn-fault-nack');
const btnFaultTimeout = document.getElementById('btn-fault-timeout');
const btnFaultRestore = document.getElementById('btn-fault-restore');

// Live State
let lastState = null;
let eventSeq = 1500;
let vehicleCounter = 501;

// Slider listener
if (manualDurationRange) {
  manualDurationRange.addEventListener('input', (e) => {
    lblDuration.textContent = `${e.target.value}s`;
  });
}

// Update Light Heads
function updateSignals(signals) {
  const directions = ['north', 'south', 'east', 'west'];
  directions.forEach(dir => {
    const dirUpper = dir.toUpperCase();
    const sig = signals ? (signals[dirUpper] || 'RED') : 'RED';
    const head = document.getElementById(`signal-${dir}`);
    if (!head) return;

    const lampRed = head.querySelector('.lamp-red');
    const lampYellow = head.querySelector('.lamp-yellow');
    const lampGreen = head.querySelector('.lamp-green');

    if (lampRed) lampRed.className = `lamp lamp-red ${sig === 'RED' ? 'active' : ''}`;
    if (lampYellow) lampYellow.className = `lamp lamp-yellow ${sig === 'YELLOW' ? 'active' : ''}`;
    if (lampGreen) lampGreen.className = `lamp lamp-green ${sig === 'GREEN' ? 'active' : ''}`;
  });
}

// Render Waiting Vehicles in Queues (Handles vehicles array or count map)
function updateQueues(vehicles, queueCounts) {
  const directions = ['north', 'south', 'east', 'west'];
  let total = 0;

  directions.forEach(dir => {
    const dirUpper = dir.toUpperCase();
    const track = document.getElementById(`queue-${dir}`);
    if (!track) return;

    track.innerHTML = '';
    const vList = (vehicles && vehicles[dirUpper]) || [];
    const count = (queueCounts && queueCounts[dirUpper]) || (Array.isArray(vList) ? vList.length : 0);
    total += count;

    if (Array.isArray(vList) && vList.length > 0) {
      vList.forEach(v => {
        const pill = document.createElement('div');
        let typeClass = 'vehicle-normal';
        if (v.type === 'TRUCK' || v.type === 'AGV') typeClass = 'vehicle-agv';
        if (v.type === 'FORKLIFT') typeClass = 'vehicle-forklift';
        if (v.type === 'EMERGENCY') typeClass = 'vehicle-emergency';

        pill.className = `vehicle-pill ${typeClass}`;
        pill.textContent = `${v.id}`;
        pill.title = `${v.type} (${v.weight || 1} pts)`;
        track.appendChild(pill);
      });
    } else if (count > 0) {
      // Fallback integer queue count representation
      for (let i = 0; i < count; i++) {
        const pill = document.createElement('div');
        pill.className = 'vehicle-pill vehicle-normal';
        pill.textContent = `VH-${dirUpper[0]}${i+1}`;
        track.appendChild(pill);
      }
    }
  });

  if (diagQueueTotal) diagQueueTotal.textContent = `${total} VEHICLES`;
}

// Update Failure & Alerts Display (Sections 14.2 & 14.6)
function updateAlerts(state, ctrlStatus, mismatch) {
  if (!alertsBanner) return;

  const alerts = state.active_alerts || [];
  const mode = state.mode || 'AUTO';

  if (mode === 'DEGRADED') {
    alertsBanner.className = 'alerts-banner alerts-danger';
    if (alertsIcon) alertsIcon.textContent = '🚨';
    if (alertsText) alertsText.textContent = `FAILSAFE DEGRADED MODE ACTIVE // ALL SIGNALS FLASHING YELLOW`;
    if (alertsSub) alertsSub.textContent = state.fail_safe_reason || 'HARDWARE NACK / COIL FAILURE DETECTED';
  } else if (ctrlStatus === 'OFFLINE') {
    alertsBanner.className = 'alerts-banner alerts-danger';
    if (alertsIcon) alertsIcon.textContent = '⚠️';
    if (alertsText) alertsText.textContent = `CONTROLLER HARDWARE OFFLINE // COMMUNICATION FAILURE`;
    if (alertsSub) alertsSub.textContent = 'AWAITING CONTROLLER RECONNECTION';
  } else if (mismatch) {
    alertsBanner.className = 'alerts-banner alerts-warning';
    if (alertsIcon) alertsIcon.textContent = '⚠️';
    if (alertsText) alertsText.textContent = `DESIRED / ACTUAL SIGNAL MISMATCH // ACTUATION IN FLIGHT`;
    if (alertsSub) alertsSub.textContent = 'PENDING CONTROLLER ACKNOWLEDGMENT';
  } else if (mode === 'EMERGENCY' || alerts.some(a => a.includes('EMERGENCY'))) {
    alertsBanner.className = 'alerts-banner alerts-danger';
    if (alertsIcon) alertsIcon.textContent = '🚨';
    if (alertsText) alertsText.textContent = `PRIORITY PREEMPTION ACTIVE // EMERGENCY CORRIDOR CLEARANCE IN PROGRESS`;
    if (alertsSub) alertsSub.textContent = state.emergency_queue && state.emergency_queue.length > 0 
      ? `UNIT: ${state.emergency_queue[0].id}` 
      : 'EMERGENCY BEACON DETECTED';
  } else if (mode === 'MANUAL') {
    alertsBanner.className = 'alerts-banner alerts-warning';
    if (alertsIcon) alertsIcon.textContent = '🕹️';
    if (alertsText) alertsText.textContent = `MANUAL OVERRIDE ACTIVE // SUPERVISOR PHASE HOLD`;
    if (alertsSub) alertsSub.textContent = 'AUTO-SCHEDULER TEMPORARILY SUSPENDED';
  } else {
    alertsBanner.className = 'alerts-banner alerts-nominal';
    if (alertsIcon) alertsIcon.textContent = '●';
    if (alertsText) alertsText.textContent = 'ALL SYSTEMS NOMINAL // CONTROLLER SYNCED // ZERO SAFETY VIOLATIONS';
    if (alertsSub) alertsSub.textContent = 'MUTUAL EXCLUSION ENFORCED';
  }
}

// Render Telemetry & Flow
function renderState(state) {
  lastState = state;
  const mode = state.mode || 'AUTO';
  const rawPhase = state.phase || state.current_phase || 'NORTH_SOUTH';
  const cleanPhase = rawPhase.replace('PHASE_', '');

  if (valMode) valMode.textContent = mode;
  if (valPhase) valPhase.textContent = cleanPhase;

  // Flow and Mode Colors
  if (mode === 'DEGRADED') {
    if (valMode) valMode.style.color = '#dc2626';
    if (flowIndicator) {
      flowIndicator.textContent = 'FAILSAFE / YELLOW FLASH';
      flowIndicator.style.color = '#f59e0b';
    }
  } else if (mode === 'EMERGENCY') {
    if (valMode) valMode.style.color = '#e11d48';
    if (flowIndicator) {
      flowIndicator.textContent = 'EMERGENCY PRIORITY CORRIDOR';
      flowIndicator.style.color = '#e11d48';
    }
  } else if (mode === 'MANUAL') {
    if (valMode) valMode.style.color = '#09090b';
    if (flowIndicator) {
      const holdPhase = (state.manual_hold && state.manual_hold.phase) ? state.manual_hold.phase.replace('PHASE_', '') : cleanPhase;
      flowIndicator.textContent = `MANUAL HOLD: ${holdPhase}`;
      flowIndicator.style.color = '#ffffff';
    }
  } else {
    if (valMode) valMode.style.color = '#059669';
    if (flowIndicator) {
      flowIndicator.textContent = `${cleanPhase} FLOW`;
      flowIndicator.style.color = '#ffffff';
    }
  }

  // Timer
  if (state.phase_started_at && phaseTimer) {
    const elapsedMs = Date.now() - new Date(state.phase_started_at).getTime();
    const sec = Math.max(0, (elapsedMs / 1000).toFixed(1));
    phaseTimer.textContent = `${sec}s`;
  }

  // Diagnostics
  if (diagEmergency) {
    diagEmergency.textContent = state.emergency_queue && state.emergency_queue.length > 0
      ? `${state.emergency_queue[0].id} (${state.emergency_queue[0].direction})`
      : 'NONE';
  }

  if (diagTransition) {
    diagTransition.textContent = state.in_transition ? (state.transition_step || 'IN_PROGRESS') : 'STEADY';
  }

  // Section 14.2: Pending Command
  if (diagPendingCmd) {
    if (state.pending_command) {
      diagPendingCmd.textContent = `${state.pending_command.id} (${state.pending_command.status || 'PENDING'})`;
      diagPendingCmd.style.color = '#f59e0b';
    } else {
      diagPendingCmd.textContent = 'NONE';
      diagPendingCmd.style.color = '#09090b';
    }
  }

  if (badgeManualStatus) {
    if (state.manual_hold && state.manual_hold.active) {
      badgeManualStatus.textContent = `OVERRIDE ACTIVE`;
      badgeManualStatus.style.background = '#09090b';
      badgeManualStatus.style.color = '#ffffff';
      badgeManualStatus.style.border = '1px solid #09090b';
    } else {
      badgeManualStatus.textContent = 'AUTO SCHEDULING';
      badgeManualStatus.style.background = '#f4f4f5';
      badgeManualStatus.style.color = '#09090b';
      badgeManualStatus.style.border = '1px solid #e4e4e7';
    }
  }

  const signals = state.desired_signals || state.signals || {};
  updateSignals(signals);
  updateQueues(state.vehicles, state.queues);
}

// Section 14.9: Periodic Telemetry Polling (Every 500ms)
async function pollStatus() {
  let isMismatch = false;
  let ctrlStatus = 'ONLINE';

  try {
    const res = await fetch(`${API_BASE}`);
    if (res.ok) {
      const data = await res.json();
      if (data.success && data.data) {
        renderState(data.data);
        ctrlStatus = data.data.controller_status || 'ONLINE';
      }
    } else {
      ctrlStatus = 'WARNING';
    }

    // Check confirmed actual hardware signals vs desired (Section 14.2 / 14.6)
    const simRes = await fetch('/api/sim/signals');
    if (simRes.ok) {
      const simData = await simRes.json();
      if (simData.success && diagSignalsState && lastState) {
        const desired = lastState.desired_signals || lastState.signals || {};
        let match = true;
        for (const d of ['NORTH', 'SOUTH', 'EAST', 'WEST']) {
          if (desired[d] && simData.data[d] && desired[d] !== simData.data[d]) {
            match = false;
            break;
          }
        }
        if (match) {
          diagSignalsState.textContent = 'CONFIRMED';
          diagSignalsState.style.color = '#059669';
          if (valHardware) {
            valHardware.textContent = 'ONLINE';
            valHardware.style.color = '#059669';
          }
        } else {
          isMismatch = true;
          diagSignalsState.textContent = 'UPDATING / MISMATCH';
          diagSignalsState.style.color = '#d97706';
        }
      }
    }

    if (lastState) {
      updateAlerts(lastState, ctrlStatus, isMismatch);
    }
  } catch (err) {
    // Section 14.10: Frontend Error Handling (Backend unreachable)
    if (valHardware) {
      valHardware.textContent = 'OFFLINE';
      valHardware.style.color = '#dc2626';
    }
    if (alertsBanner) {
      alertsBanner.className = 'alerts-banner alerts-danger';
      if (alertsIcon) alertsIcon.textContent = '⚠️';
      if (alertsText) alertsText.textContent = 'BACKEND CONNECTION OFFLINE // ATTEMPTING RECONNECT...';
      if (alertsSub) alertsSub.textContent = 'HTTP SERVER PORT 8080 UNREACHABLE';
    }
  }
}

// Section 14.7: Recent Activity Journal
async function fetchAuditLogs() {
  try {
    const res = await fetch(`${API_BASE}/history?limit=25`);
    if (res.ok) {
      const logs = await res.json();
      if (Array.isArray(logs)) {
        renderLogs(logs);
      }
    }
  } catch (err) {
    // Graceful error logging
    console.warn('Audit ledger fetch deferred:', err.message);
  }
}

function renderLogs(logs) {
  if (!journalFeed) return;
  journalFeed.innerHTML = '';
  logs.forEach(log => {
    const entry = document.createElement('div');
    entry.className = 'log-entry';

    let tagClass = 'tag-event';
    const evt = log.event_type || '';
    if (evt.includes('EMERGENCY') || evt.includes('FAILSAFE')) tagClass = 'tag-danger';
    if (evt.includes('MANUAL') || evt.includes('YELLOW')) tagClass = 'tag-warn';
    if (evt.includes('BOOT') || evt.includes('INIT')) tagClass = 'tag-sys';

    const timeStr = log.timestamp ? new Date(log.timestamp).toLocaleTimeString() : 'LIVE';
    entry.innerHTML = `
      <span class="log-time">${timeStr}</span>
      <span class="log-tag ${tagClass}">${evt}</span>
      <span class="log-text">${log.phase || ''} [${log.mode || ''}] ${JSON.stringify(log.details || {})}</span>
    `;
    journalFeed.appendChild(entry);
  });
}

// Section 14.8: Traffic Simulation & Dispatches
btnDispatch.addEventListener('click', async () => {
  try {
    const id = dispatchIdInput.value.trim() || `VH-${vehicleCounter++}`;
    const type = dispatchTypeInput.value;
    const direction = dispatchDirInput.value;
    eventSeq++;

    await fetch('/api/sensor-events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        event_id: `evt-${eventSeq}`,
        junction_id: JUNCTION_ID,
        direction: direction,
        event_type: 'VEHICLE_ARRIVED',
        vehicle_id: id,
        vehicle_type: type,
        sequence_no: eventSeq,
        timestamp: new Date().toISOString()
      })
    });

    dispatchIdInput.value = `VH-${vehicleCounter++}`;
    pollStatus();
    fetchAuditLogs();
  } catch (err) {
    console.error('Dispatch error', err);
  }
});

btnQuickEmergency.addEventListener('click', async () => {
  try {
    eventSeq++;
    const id = `AMB-${Math.floor(Math.random() * 900 + 100)}`;
    const dir = dispatchDirInput.value;

    await fetch('/api/sensor-events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        event_id: `evt-${eventSeq}`,
        junction_id: JUNCTION_ID,
        direction: dir,
        event_type: 'VEHICLE_ARRIVED',
        vehicle_id: id,
        vehicle_type: 'EMERGENCY',
        sequence_no: eventSeq,
        timestamp: new Date().toISOString()
      })
    });

    pollStatus();
    fetchAuditLogs();
  } catch (err) {
    console.error('Emergency dispatch error', err);
  }
});

btnClearFirst.addEventListener('click', async () => {
  try {
    if (!lastState) return;
    eventSeq++;

    const rawPhase = lastState.phase || lastState.current_phase || 'NORTH_SOUTH';
    let activeDirs = ['NORTH', 'SOUTH'];
    if (rawPhase.includes('EAST_WEST')) {
      activeDirs = ['EAST', 'WEST'];
    }

    const queues = lastState.vehicles || lastState.queues || {};
    for (const d of activeDirs) {
      const q = queues[d];
      let vehID = `VH-CLEAR-${d}`;
      if (Array.isArray(q) && q.length > 0) {
        vehID = q[0].id;
      }
      if ((Array.isArray(q) && q.length > 0) || (typeof q === 'number' && q > 0)) {
        await fetch('/api/sensor-events', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            event_id: `evt-${eventSeq}`,
            junction_id: JUNCTION_ID,
            direction: d,
            event_type: 'VEHICLE_CLEARED',
            vehicle_id: vehID,
            sequence_no: eventSeq,
            timestamp: new Date().toISOString()
          })
        });
        break;
      }
    }

    pollStatus();
    fetchAuditLogs();
  } catch (err) {
    console.error('Clear crossing error', err);
  }
});

// Section 14.4: Manual Traffic Control
btnApplyManual.addEventListener('click', async () => {
  try {
    const phase = manualPhaseSelect.value;
    const dir = (phase === 'PHASE_EAST_WEST') ? 'WEST' : 'NORTH';

    await fetch(`${API_BASE}/commands`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        command: 'MANUAL_GREEN_REQUEST',
        direction: dir
      })
    });

    pollStatus();
    fetchAuditLogs();
  } catch (err) {
    console.error('Manual override request error', err);
  }
});

btnReleaseManual.addEventListener('click', async () => {
  try {
    await fetch(`${API_BASE}/commands`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        command: 'RETURN_TO_AUTOMATIC'
      })
    });

    pollStatus();
    fetchAuditLogs();
  } catch (err) {
    console.error('Release manual error', err);
  }
});

// Section 14.6 & 14.8: Fault Injection & Hardware Simulator
btnFaultNack.addEventListener('click', async () => {
  try {
    await fetch('/api/sim/fault', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ simulate_nack: true, healthy: true })
    });
    pollStatus();
  } catch (err) {
    console.error('Fault injection error', err);
  }
});

btnFaultTimeout.addEventListener('click', async () => {
  try {
    await fetch('/api/sim/fault', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ drop_ack: true, healthy: true })
    });
    pollStatus();
  } catch (err) {
    console.error('Fault injection error', err);
  }
});

btnFaultRestore.addEventListener('click', async () => {
  try {
    await fetch('/api/sim/fault', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ simulate_nack: false, drop_ack: false, healthy: true })
    });
    pollStatus();
  } catch (err) {
    console.error('Fault restore error', err);
  }
});

// Polling intervals (Section 14.9: Polling chosen for resilient reconnect and low overhead)
setInterval(pollStatus, 500);
setInterval(fetchAuditLogs, 2000);
pollStatus();
fetchAuditLogs();
