package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatistics(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ShouldSend", testStatisticsShouldSend},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testStatisticsShouldSend(t *testing.T, ds *Datastore) {
	// Create new host for test
	_, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   "M",
	})
	require.NoError(t, err)

	// Create two new users for test
	u1, err := ds.NewUser(context.Background(), &fleet.User{
		Password:                 []byte("foobar"),
		AdminForcedPasswordReset: false,
		Email:                    "baz@example.com",
		SSOEnabled:               false,
		GlobalRole:               ptr.String(fleet.RoleObserver),
	})
	require.NoError(t, err)
	_, err = ds.NewUser(context.Background(), &fleet.User{
		Password:                 []byte("foobar"),
		AdminForcedPasswordReset: false,
		Email:                    "qux@example.com",
		SSOEnabled:               false,
		GlobalRole:               ptr.String(fleet.RoleObserver),
	})
	require.NoError(t, err)
	// Create a session for user baz, but not qux (so only 1 is active)
	_, err = ds.NewSession(context.Background(), u1.ID, "session_key")
	require.NoError(t, err)

	// Create new team for test
	_, err = ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "footeam",
		Description: "team of foo",
	})
	require.NoError(t, err)

	// Create new global policy for test
	_, err = ds.NewGlobalPolicy(context.Background(), ptr.Uint(1), fleet.PolicyPayload{
		Name:        "testpolicy",
		Query:       "select 1;",
		Description: "test policy desc",
		Resolution:  "test policy resolution",
	})
	require.NoError(t, err)

	// Create new label for test
	_, err = ds.NewLabel(context.Background(), &fleet.Label{
		Name:        "testlabel",
		Query:       "select 1;",
		Platform:    "darwin",
		Description: "test label description",
	})
	require.NoError(t, err)

	// Create new app config for test
	config, err := ds.NewAppConfig(context.Background(), &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName:    "Test",
			OrgLogoURL: "localhost:8080/logo.png",
		},
	})
	require.NoError(t, err)
	config.HostSettings.EnableSoftwareInventory = false
	config.HostSettings.EnableHostUsers = false
	config.VulnerabilitySettings.DatabasesPath = ""
	config.WebhookSettings.HostStatusWebhook.Enable = true

	err = ds.SaveAppConfig(context.Background(), config)
	require.NoError(t, err)

	time.Sleep(1100 * time.Millisecond) // ensure the DB timestamp is not in the same second

	license := &fleet.LicenseInfo{Tier: "premium"}

	// First time running, we send statistics
	stats, shouldSend, err := ds.ShouldSendStatistics(context.Background(), fleet.StatisticsFrequency, license)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.NotEmpty(t, stats.AnonymousIdentifier)
	assert.NotEmpty(t, stats.FleetVersion)
	assert.Equal(t, stats.LicenseTier, "premium")
	assert.Equal(t, stats.NumHostsEnrolled, 1)
	assert.Equal(t, stats.NumUsers, 2)
	assert.Equal(t, stats.NumTeams, 1)
	assert.Equal(t, stats.NumPolicies, 1)
	assert.Equal(t, stats.NumLabels, 1)
	assert.Equal(t, stats.SoftwareInventoryEnabled, false)
	assert.Equal(t, stats.SystemUsersEnabled, false)
	assert.Equal(t, stats.VulnDetectionEnabled, false)
	assert.Equal(t, stats.HostsStatusWebHookEnabled, true)
	assert.Equal(t, stats.NumWeeklyActiveUsers, 1)

	firstIdentifier := stats.AnonymousIdentifier

	err = ds.RecordStatisticsSent(context.Background())
	require.NoError(t, err)

	// If we try right away, it shouldn't ask to send
	stats, shouldSend, err = ds.ShouldSendStatistics(context.Background(), fleet.StatisticsFrequency, license)
	require.NoError(t, err)
	assert.False(t, shouldSend)

	time.Sleep(1100 * time.Millisecond) // ensure the DB timestamp is not in the same second

	// create a few more hosts, with platforms and os versions
	_, err = ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		OsqueryHostID:   "S",
		Platform:        "rhel",
		OSVersion:       "Fedora 35",
	})
	require.NoError(t, err)

	_, err = ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "3",
		UUID:            "3",
		Hostname:        "foo.local3",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "40-65-EC-6F-C4-59",
		OsqueryHostID:   "T",
		Platform:        "rhel",
		OSVersion:       "Fedora 35",
	})
	require.NoError(t, err)

	_, err = ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "4",
		UUID:            "4",
		Hostname:        "foo.local4",
		PrimaryIP:       "192.168.1.4",
		PrimaryMac:      "50-65-EC-6F-C4-59",
		OsqueryHostID:   "U",
		Platform:        "macos",
		OSVersion:       "10.11.12",
	})
	require.NoError(t, err)

	_, err = ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "5",
		UUID:            "5",
		Hostname:        "foo.local5",
		PrimaryIP:       "192.168.1.5",
		PrimaryMac:      "60-65-EC-6F-C4-59",
		OsqueryHostID:   "V",
		Platform:        "rhel",
		OSVersion:       "Fedora 36",
	})
	require.NoError(t, err)

	// Lower the frequency to trigger an "outdated" sent
	stats, shouldSend, err = ds.ShouldSendStatistics(context.Background(), time.Millisecond, license)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.Equal(t, firstIdentifier, stats.AnonymousIdentifier)
	assert.Equal(t, stats.NumHostsEnrolled, 5)
	assert.Equal(t, stats.NumUsers, 2)
	assert.Equal(t, stats.NumWeeklyActiveUsers, 0)          // no active user since last stats were sent
	require.Len(t, stats.HostsEnrolledByOperatingSystem, 3) // empty platform, rhel and macos
	require.ElementsMatch(t, []fleet.HostsCountByOSVersion{
		{Version: "Fedora 35", NumEnrolled: 2},
		{Version: "Fedora 36", NumEnrolled: 1},
	}, stats.HostsEnrolledByOperatingSystem["rhel"])
	require.ElementsMatch(t, []fleet.HostsCountByOSVersion{
		{Version: "10.11.12", NumEnrolled: 1},
	}, stats.HostsEnrolledByOperatingSystem["macos"])
	require.ElementsMatch(t, []fleet.HostsCountByOSVersion{
		{Version: "", NumEnrolled: 1},
	}, stats.HostsEnrolledByOperatingSystem[""])

	// Create multiple new sessions for a single user
	_, err = ds.NewSession(context.Background(), u1.ID, "session_key2")
	require.NoError(t, err)
	_, err = ds.NewSession(context.Background(), u1.ID, "session_key3")
	require.NoError(t, err)
	_, err = ds.NewSession(context.Background(), u1.ID, "session_key4")
	require.NoError(t, err)

	// wait a bit and resend statistics
	time.Sleep(1100 * time.Millisecond) // ensure the DB timestamp is not in the same second

	stats, shouldSend, err = ds.ShouldSendStatistics(context.Background(), time.Millisecond, license)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.Equal(t, firstIdentifier, stats.AnonymousIdentifier)
	assert.Equal(t, stats.NumHostsEnrolled, 5)
	assert.Equal(t, stats.NumUsers, 2)
	assert.Equal(t, stats.NumWeeklyActiveUsers, 1)
}
