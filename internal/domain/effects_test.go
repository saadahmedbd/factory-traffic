package domain

import (
	"testing"
	"time"
)

func TestEffectConstructors(t *testing.T) {
	now := time.Now()
	cmd := ControllerCommand{ID: "c1", Type: CommandAllRed}
	cmdEff := NewSendCommandEffect(cmd)

	if cmdEff.Type != EffectTypeSendCommand || cmdEff.Command.ID != "c1" {
		t.Errorf("invalid send command effect: %+v", cmdEff)
	}

	auditEff := NewAuditEffect("j1", "TEST_EVENT", PhaseNorthSouth, ModeAuto, map[string]interface{}{"key": "val"}, now)
	if auditEff.Type != EffectTypeAudit || auditEff.Audit.EventType != "TEST_EVENT" {
		t.Errorf("invalid audit effect: %+v", auditEff)
	}

	persistEff := NewPersistEffect()
	if persistEff.Type != EffectTypePersist {
		t.Errorf("invalid persist effect: %+v", persistEff)
	}

	timerEff := NewScheduleTimerEffect(5*time.Second, "TEST_TIMER")
	if timerEff.Type != EffectTypeScheduleTimer || timerEff.TimerDuration != 5*time.Second {
		t.Errorf("invalid timer effect: %+v", timerEff)
	}
}
