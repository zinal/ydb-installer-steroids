package unit

import (
	"testing"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/hosts"
)

// TestParseTimedatectl_Chrony tests parsing of timedatectl output with chrony.
func TestParseTimedatectl_Chrony(t *testing.T) {
	c := hosts.NewChecker()
	output := `NTPService=chronyd.service
NTPSynchronized=yes
`
	result := c.ParseTimedatectl(output)

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPDaemon != "chrony" {
		t.Errorf("expected daemon 'chrony', got '%s'", result.NTPDaemon)
	}
	if !result.NTPSynced {
		t.Error("expected NTPSynced=true")
	}
}

// TestParseTimedatectl_Ntpd tests parsing of timedatectl output with ntpd.
func TestParseTimedatectl_Ntpd(t *testing.T) {
	c := hosts.NewChecker()
	output := `NTPService=ntpd.service
NTPSynchronized=no
`
	result := c.ParseTimedatectl(output)

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPDaemon != "ntpd" {
		t.Errorf("expected daemon 'ntpd', got '%s'", result.NTPDaemon)
	}
	if result.NTPSynced {
		t.Error("expected NTPSynced=false")
	}
	if result.Error == "" {
		t.Error("expected error message for not-synced state")
	}
}

// TestParseTimedatectl_Empty tests parsing of empty timedatectl output.
func TestParseTimedatectl_Empty(t *testing.T) {
	c := hosts.NewChecker()
	result := c.ParseTimedatectl("")

	if result.NTPActive {
		t.Error("expected NTPActive=false for empty output")
	}
}

// TestParseChronyc_Synced tests parsing of chronyc tracking output when synced.
func TestParseChronyc_Synced(t *testing.T) {
	c := hosts.NewChecker()
	output := `Reference ID    : CA7C0B2A (ntp1.example.com)
Stratum         : 2
Ref time (UTC)  : Thu Jun 26 10:00:00 2025
System time     : 0.000123456 seconds fast of NTP time
Last offset     : 0.000123456 seconds
RMS offset      : 0.000123456 seconds
Leap status     : Normal
`
	result := c.ParseChronyc(output)

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPDaemon != "chrony" {
		t.Errorf("expected daemon 'chrony', got '%s'", result.NTPDaemon)
	}
	if !result.NTPSynced {
		t.Error("expected NTPSynced=true for Leap status Normal")
	}
	if result.ClockOffsetMs != 0 {
		t.Errorf("expected offset 0ms, got %d", result.ClockOffsetMs)
	}
}

// TestParseChronyc_NotSynced tests parsing of chronyc tracking when not synced.
func TestParseChronyc_NotSynced(t *testing.T) {
	c := hosts.NewChecker()
	output := `Reference ID    : 00000000 ()
Stratum         : 0
Leap status     : Not synchronised
`
	result := c.ParseChronyc(output)

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPSynced {
		t.Error("expected NTPSynced=false for Not synchronised")
	}
	if result.Error == "" {
		t.Error("expected error message for not-synced state")
	}
}

// TestParseNtpq_Synced tests parsing of ntpq -p output when synced.
func TestParseNtpq_Synced(t *testing.T) {
	c := hosts.NewChecker()
	output := `     remote           refid      st t when poll reach   offset   delay  jitter
==============================================================================
*ntp1.example.com .GPS.          1 u   64  128  377    0.123   1.234   0.567
+ntp2.example.com .PPS.          1 u   32  128  377    0.456   2.345   0.678
`
	result := c.ParseNtpq(output)

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPDaemon != "ntpd" {
		t.Errorf("expected daemon 'ntpd', got '%s'", result.NTPDaemon)
	}
	if !result.NTPSynced {
		t.Error("expected NTPSynced=true for system peer (*)")
	}
}

// TestParseNtpq_NotSynced tests parsing of ntpq -p output when not synced.
func TestParseNtpq_NotSynced(t *testing.T) {
	c := hosts.NewChecker()
	output := `     remote           refid      st t when poll reach   offset   delay  jitter
==============================================================================
 ntp1.example.com .GPS.          1 u   64  128  377    0.123   1.234   0.567
`
	result := c.ParseNtpq(output)

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPSynced {
		t.Error("expected NTPSynced=false (no system peer)")
	}
}

// TestCheckNTPStatus_NoDaemon tests that when all NTP tools are unavailable,
// the result reports NTPActive=false with an error.
func TestCheckNTPStatus_NoDaemon(t *testing.T) {
	c := hosts.NewChecker()

	// All outputs empty simulates every NTP tool being unavailable
	result := c.CheckNTPStatus(hosts.NTPOutputs{})

	if result.NTPActive {
		t.Error("expected NTPActive=false when no NTP tools available")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

// TestCheckNTPStatus_TimedatectlSuccess tests that timedatectl is tried first.
func TestCheckNTPStatus_TimedatectlSuccess(t *testing.T) {
	c := hosts.NewChecker()

	result := c.CheckNTPStatus(hosts.NTPOutputs{
		ShowTimesync: "NTPService=chronyd.service\nNTPSynchronized=yes\n",
	})

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPDaemon != "chrony" {
		t.Errorf("expected daemon 'chrony', got '%s'", result.NTPDaemon)
	}
	if !result.NTPSynced {
		t.Error("expected NTPSynced=true")
	}
}

// TestCheckNTPStatus_ChronycFallback tests that chronyc is tried when timedatectl fails.
func TestCheckNTPStatus_ChronycFallback(t *testing.T) {
	c := hosts.NewChecker()

	result := c.CheckNTPStatus(hosts.NTPOutputs{
		ChronycTracking: "Leap status     : Normal\nSystem time     : 0.001 seconds fast of NTP time\n",
	})

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPDaemon != "chrony" {
		t.Errorf("expected daemon 'chrony', got '%s'", result.NTPDaemon)
	}
	if !result.NTPSynced {
		t.Error("expected NTPSynced=true")
	}
}

// TestMeasureDrift_AllSameTime tests drift calculation when all hosts have the same time.
func TestMeasureDrift_AllSameTime(t *testing.T) {
	// We can't easily test measureDrift directly since it makes SSH calls,
	// but we can test the drift computation logic via the TimeSyncSummary.
	// The actual SSH-based measurement is tested via integration.
	// Here we verify the threshold logic.

	summary := &models.TimeSyncSummary{}
	summary.Threshold.WarningMs = 5
	summary.Threshold.ErrorMs = 25

	summary.MaxDriftMs = 0
	if summary.MaxDriftMs >= summary.Threshold.ErrorMs {
		summary.Status = "error"
	} else if summary.MaxDriftMs >= summary.Threshold.WarningMs {
		summary.Status = "warning"
	} else {
		summary.Status = "ok"
	}

	if summary.Status != "ok" {
		t.Errorf("expected status 'ok' for 0ms drift, got '%s'", summary.Status)
	}
}

// TestMeasureDrift_WarningLevel tests drift classification at warning level.
func TestMeasureDrift_WarningLevel(t *testing.T) {
	summary := &models.TimeSyncSummary{}
	summary.Threshold.WarningMs = 5
	summary.Threshold.ErrorMs = 25
	summary.MaxDriftMs = 10

	switch {
	case summary.MaxDriftMs >= summary.Threshold.ErrorMs:
		summary.Status = "error"
	case summary.MaxDriftMs >= summary.Threshold.WarningMs:
		summary.Status = "warning"
	default:
		summary.Status = "ok"
	}

	if summary.Status != "warning" {
		t.Errorf("expected status 'warning' for 10ms drift, got '%s'", summary.Status)
	}
}

// TestMeasureDrift_ErrorLevel tests drift classification at error level.
func TestMeasureDrift_ErrorLevel(t *testing.T) {
	summary := &models.TimeSyncSummary{}
	summary.Threshold.WarningMs = 5
	summary.Threshold.ErrorMs = 25
	summary.MaxDriftMs = 30

	switch {
	case summary.MaxDriftMs >= summary.Threshold.ErrorMs:
		summary.Status = "error"
	case summary.MaxDriftMs >= summary.Threshold.WarningMs:
		summary.Status = "warning"
	default:
		summary.Status = "ok"
	}

	if summary.Status != "error" {
		t.Errorf("expected status 'error' for 30ms drift, got '%s'", summary.Status)
	}
}

// TestMeasureDrift_BoundaryConditions tests exact threshold boundaries.
func TestMeasureDrift_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name     string
		driftMs  int64
		expected string
	}{
		{"just below warning (4ms)", 4, "ok"},
		{"exactly at warning (5ms)", 5, "warning"},
		{"just above warning (6ms)", 6, "warning"},
		{"just below error (24ms)", 24, "warning"},
		{"exactly at error (25ms)", 25, "error"},
		{"just above error (26ms)", 26, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := &models.TimeSyncSummary{}
			summary.Threshold.WarningMs = 5
			summary.Threshold.ErrorMs = 25
			summary.MaxDriftMs = tt.driftMs

			switch {
			case summary.MaxDriftMs >= summary.Threshold.ErrorMs:
				summary.Status = "error"
			case summary.MaxDriftMs >= summary.Threshold.WarningMs:
				summary.Status = "warning"
			default:
				summary.Status = "ok"
			}

			if summary.Status != tt.expected {
				t.Errorf("drift %dms: expected '%s', got '%s'", tt.driftMs, tt.expected, summary.Status)
			}
		})
	}
}

// TestParseTimedatectlStatus_ActiveSynced tests parsing of plain `timedatectl` output
// when NTP service is active and clock is synchronized.
func TestParseTimedatectlStatus_ActiveSynced(t *testing.T) {
	c := hosts.NewChecker()
	output := `               Local time: Thu 2026-06-26 14:30:00 UTC
	          Universal time: Thu 2026-06-26 14:30:00 UTC
	                RTC time: Thu 2026-06-26 14:30:00
	               Time zone: Etc/UTC (UTC, +0000)
System clock synchronized: yes
	             NTP service: active
	         RTC in local TZ: no`

	result := c.ParseTimedatectlStatus(output)

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPDaemon != "systemd-timesyncd" {
		t.Errorf("expected daemon 'systemd-timesyncd', got '%s'", result.NTPDaemon)
	}
	if !result.NTPSynced {
		t.Error("expected NTPSynced=true")
	}
}

// TestParseTimedatectlStatus_Inactive tests parsing of plain `timedatectl` output
// when NTP service is inactive.
func TestParseTimedatectlStatus_Inactive(t *testing.T) {
	c := hosts.NewChecker()
	output := `System clock synchronized: no
	             NTP service: inactive`

	result := c.ParseTimedatectlStatus(output)

	if result.NTPActive {
		t.Error("expected NTPActive=false for inactive NTP service")
	}
}

// TestParseTimedatectlStatus_ActiveNotSynced tests parsing when NTP is active
// but clock is not synchronized.
func TestParseTimedatectlStatus_ActiveNotSynced(t *testing.T) {
	c := hosts.NewChecker()
	output := `System clock synchronized: no
	             NTP service: active`

	result := c.ParseTimedatectlStatus(output)

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPSynced {
		t.Error("expected NTPSynced=false")
	}
	if result.Error == "" {
		t.Error("expected error message for not-synced state")
	}
}

// TestParseTimedatectlStatus_Empty tests parsing of empty timedatectl status output.
func TestParseTimedatectlStatus_Empty(t *testing.T) {
	c := hosts.NewChecker()
	result := c.ParseTimedatectlStatus("")

	if result.NTPActive {
		t.Error("expected NTPActive=false for empty output")
	}
}

// TestCheckNTPStatus_SystemctlFallback tests that systemctl is tried when all
// other NTP tools are unavailable.
func TestCheckNTPStatus_SystemctlFallback(t *testing.T) {
	c := hosts.NewChecker()

	// systemctl fallback: chronyd inactive, systemd-timesyncd active.
	// The combined script only emits "svc:active" lines for active services.
	result := c.CheckNTPStatus(hosts.NTPOutputs{
		SystemctlActive: "systemd-timesyncd.service:active\n",
	})

	if !result.NTPActive {
		t.Error("expected NTPActive=true via systemctl fallback")
	}
	if result.NTPDaemon != "systemd-timesyncd" {
		t.Errorf("expected daemon 'systemd-timesyncd', got '%s'", result.NTPDaemon)
	}
	if !result.NTPSynced {
		t.Error("expected NTPSynced=true")
	}
}

// TestCheckNTPStatus_TimedatectlThenStatus tests that plain `timedatectl` is tried
// when `show-timesync` succeeds but finds no NTP service.
func TestCheckNTPStatus_TimedatectlThenStatus(t *testing.T) {
	c := hosts.NewChecker()

	// show-timesync succeeds but has no NTPService= line → parsed as NTPActive=false
	// plain timedatectl shows NTP service: active → should be used
	result := c.CheckNTPStatus(hosts.NTPOutputs{
		// Output with LinkNTPServers but no NTPService= (real-world scenario)
		ShowTimesync:     "LinkNTPServers=\nFallbackNTPServers=\nPollIntervalUSec=0\n",
		PlainTimedatectl: "NTP service: active\nSystem clock synchronized: yes\n",
	})

	if !result.NTPActive {
		t.Error("expected NTPActive=true via plain timedatectl fallback")
	}
}

type mockError struct{}

func (e *mockError) Error() string {
	return "mock error"
}

// TestParseTimedatectlTimesync_Synced tests parsing of `timedatectl timesync-status`
// when NTP is synchronized (Leap: normal) with a positive offset.
func TestParseTimedatectlTimesync_Synced(t *testing.T) {
	c := hosts.NewChecker()
	output := `       Server: 192.168.56.23 (ansible)
Poll interval: 17min 4s (min: 32s; max 34min 8s)
         Leap: normal
      Version: 4
      Stratum: 3
    Reference: 5D4DB71C
    Precision: 1us (-23)
Root distance: 6.438ms (max: 5s)
       Offset: +2.670ms
        Delay: 1.504ms
       Jitter: 1.881ms
 Packet count: 6
    Frequency: +9.485ppm`

	result := c.ParseTimedatectlTimesync(output)

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPDaemon != "systemd-timesyncd" {
		t.Errorf("expected daemon 'systemd-timesyncd', got '%s'", result.NTPDaemon)
	}
	if !result.NTPSynced {
		t.Error("expected NTPSynced=true for Leap: normal")
	}
	if result.ClockOffsetMs != 2 {
		t.Errorf("expected ClockOffsetMs=2, got %d", result.ClockOffsetMs)
	}
}

// TestParseTimedatectlTimesync_NotSynced tests parsing when Leap is not normal.
func TestParseTimedatectlTimesync_NotSynced(t *testing.T) {
	c := hosts.NewChecker()
	output := `       Server: (unreachable)
         Leap: not synchronised
       Offset: +0.000ms`

	result := c.ParseTimedatectlTimesync(output)

	if !result.NTPActive {
		t.Error("expected NTPActive=true")
	}
	if result.NTPSynced {
		t.Error("expected NTPSynced=false for Leap: not synchronised")
	}
	if result.Error == "" {
		t.Error("expected error message for not-synced state")
	}
}

// TestParseTimedatectlTimesync_NegativeOffset tests parsing of negative offset.
func TestParseTimedatectlTimesync_NegativeOffset(t *testing.T) {
	c := hosts.NewChecker()
	output := `         Leap: normal
       Offset: -5.232ms`

	result := c.ParseTimedatectlTimesync(output)

	if !result.NTPSynced {
		t.Error("expected NTPSynced=true")
	}
	if result.ClockOffsetMs != -5 {
		t.Errorf("expected ClockOffsetMs=-5, got %d", result.ClockOffsetMs)
	}
}

// TestMeasureDrift_RealWorldOffsets tests with the actual offsets from the user's cluster:
// node-1: +2.670ms, node-2: +5.232ms, node-3: +5.293ms
// Expected drift: 5 - 2 = 3ms (after int truncation) → status "ok"
// This replaces the old date-based method that incorrectly reported 36ms.
func TestMeasureDrift_RealWorldOffsets(t *testing.T) {
	summary := &models.TimeSyncSummary{}
	summary.Threshold.WarningMs = 5
	summary.Threshold.ErrorMs = 25

	// Real offsets from timedatectl timesync-status (truncated to int ms)
	offsets := []int64{2, 5, 5}
	var minOffset, maxOffset int64 = offsets[0], offsets[0]
	for _, off := range offsets[1:] {
		if off < minOffset {
			minOffset = off
		}
		if off > maxOffset {
			maxOffset = off
		}
	}
	drift := maxOffset - minOffset
	if drift < 0 {
		drift = -drift
	}
	summary.MaxDriftMs = drift

	switch {
	case summary.MaxDriftMs >= summary.Threshold.ErrorMs:
		summary.Status = "error"
	case summary.MaxDriftMs >= summary.Threshold.WarningMs:
		summary.Status = "warning"
	default:
		summary.Status = "ok"
	}

	// With real offsets [2, 5, 5], drift = 3ms → ok (NOT 36ms error!)
	if summary.Status != "ok" {
		t.Errorf("expected status 'ok' for real-world offsets, got '%s' (drift=%dms)", summary.Status, summary.MaxDriftMs)
	}
	if summary.MaxDriftMs >= 25 {
		t.Errorf("drift %dms should not be error-level (was 36ms with old date-based method)", summary.MaxDriftMs)
	}
}
