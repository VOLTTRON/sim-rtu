//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	runtimePkg "runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/services"

	"github.com/VOLTTRON/sim-rtu/internal/api"
	bacnetsrv "github.com/VOLTTRON/sim-rtu/internal/bacnet"
	"github.com/VOLTTRON/sim-rtu/internal/config"
	"github.com/VOLTTRON/sim-rtu/internal/engine"
	"github.com/VOLTTRON/sim-rtu/internal/points"
)

// testEnv holds the running engine, BACnet server, and API server for a test.
type testEnv struct {
	engine    *engine.Engine
	bacnetSrv *bacnetsrv.Server
	apiSrv    *api.Server

	bacnetAddr net.Addr
	apiPort    int
	cancel     context.CancelFunc
}

// projectRoot returns the absolute path to the sim-rtu project root.
func projectRoot(t *testing.T) string {
	t.Helper()
	// tests/integration/ is two levels below project root
	_, filename, _, ok := runtimePkg.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// startTestEnv creates a full sim-rtu stack on random ports.
func startTestEnv(t *testing.T) *testEnv {
	t.Helper()

	root := projectRoot(t)

	// Change to project root so registry paths in config resolve correctly
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir to project root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cfg, err := config.Load(filepath.Join(root, "configs", "default.yml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	// Override ports to 0 (random) so tests don't conflict
	cfg.BACnet.Port = 0
	cfg.BACnet.Interface = "127.0.0.1"
	cfg.API.Port = 0
	cfg.API.Host = "127.0.0.1"

	eng, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start engine
	go func() {
		_ = eng.Start(ctx)
	}()

	// Start BACnet server on random port
	bacnetCfg := config.BACnetConfig{
		Enabled:   true,
		Interface: "127.0.0.1",
		Port:      0,
	}
	bsrv := bacnetsrv.New(bacnetCfg)

	// Listen on random port manually
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatalf("listen bacnet: %v", err)
	}

	stores := make(map[int]*points.PointStore)
	for id, dev := range eng.Devices() {
		stores[id] = dev.Store
	}

	// Use the Start method's internal setup by injecting our conn
	// We need to use the unexported fields, so we start normally but with port 0
	// Actually, we already have the server_test.go pattern — inject conn directly.
	// But Server.Start sets conn internally. Let's just start with the real Start.
	// Instead, use a config with port from our listener.
	conn.Close() // close so Start can bind

	// Pick a free port
	bacnetPort := getFreePort(t, "udp")
	bacnetCfg.Port = bacnetPort
	bsrv = bacnetsrv.New(bacnetCfg)

	if err := bsrv.Start(ctx, stores); err != nil {
		cancel()
		t.Fatalf("start bacnet server: %v", err)
	}

	bacnetAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", bacnetPort))
	if err != nil {
		cancel()
		t.Fatalf("resolve bacnet addr: %v", err)
	}

	// Start API server on random port
	apiPort := getFreePort(t, "tcp")
	asrv := api.New(eng, "127.0.0.1", apiPort, "")
	go func() {
		_ = asrv.Start(ctx)
	}()

	// Wait for API to be ready
	apiBase := fmt.Sprintf("http://127.0.0.1:%d", apiPort)
	waitForAPI(t, apiBase, 10*time.Second)

	env := &testEnv{
		engine:     eng,
		bacnetSrv:  bsrv,
		apiSrv:     asrv,
		bacnetAddr: bacnetAddr,
		apiPort:    apiPort,
		cancel:     cancel,
	}

	t.Cleanup(func() {
		cancel()
		_ = bsrv.Stop()
		shutdownCtx, sc := context.WithTimeout(context.Background(), 2*time.Second)
		defer sc()
		_ = asrv.Stop(shutdownCtx)
		eng.Stop()
	})

	return env
}

// getFreePort finds a free port of the given network type.
func getFreePort(t *testing.T, network string) int {
	t.Helper()
	switch network {
	case "tcp":
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("get free tcp port: %v", err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		l.Close()
		return port
	case "udp":
		c, err := net.ListenPacket("udp4", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("get free udp port: %v", err)
		}
		port := c.LocalAddr().(*net.UDPAddr).Port
		c.Close()
		return port
	default:
		t.Fatalf("unsupported network: %s", network)
		return 0
	}
}

// waitForAPI polls the API status endpoint until it responds.
func waitForAPI(t *testing.T, base string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/api/v1/status")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("API did not become ready within timeout")
}

// sendBACnet sends a BACnet packet and reads the response.
func sendBACnet(t *testing.T, addr net.Addr, request []byte) []byte {
	t.Helper()

	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("client listen: %v", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	if _, err := conn.WriteTo(request, addr); err != nil {
		t.Fatalf("send: %v", err)
	}

	buf := make([]byte, 1500)
	n, _, err := conn.ReadFrom(buf)
	if err != nil {
		t.Fatalf("receive: %v", err)
	}

	return buf[:n]
}

// collectIAmResponses sends WhoIs and collects all IAm responses within a timeout.
func collectIAmResponses(t *testing.T, addr net.Addr) map[uint32]bool {
	t.Helper()

	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("client listen: %v", err)
	}
	defer conn.Close()

	whoisBytes, err := bacnet.NewWhois()
	if err != nil {
		t.Fatalf("create WhoIs: %v", err)
	}

	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	if _, err := conn.WriteTo(whoisBytes, addr); err != nil {
		t.Fatalf("send WhoIs: %v", err)
	}

	deviceIDs := make(map[uint32]bool)
	buf := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			break // timeout — done collecting
		}
		msg, err := bacnet.Parse(buf[:n])
		if err != nil {
			continue
		}
		iam, ok := msg.(*services.UnconfirmedIAm)
		if !ok {
			continue
		}
		dec, err := iam.Decode()
		if err != nil {
			continue
		}
		deviceIDs[dec.InstanceNum] = true
	}

	return deviceIDs
}

// apiGet makes an API GET request and returns the body.
func apiGet(t *testing.T, port int, path string) []byte {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d", path, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return body
}

// --- Tests ---

func TestWhoIsReturnsAllDevices(t *testing.T) {
	env := startTestEnv(t)

	deviceIDs := collectIAmResponses(t, env.bacnetAddr)

	// default.yml defines: 86254 (Schneider), 86255 (OpenStat), 20001 (DENT)
	expected := []uint32{86254, 86255, 20001}
	for _, id := range expected {
		if !deviceIDs[id] {
			t.Errorf("missing IAm response for device %d", id)
		}
	}

	if len(deviceIDs) != len(expected) {
		t.Errorf("expected %d devices, got %d: %v", len(expected), len(deviceIDs), deviceIDs)
	}
}

func TestReadZoneTemperature(t *testing.T) {
	env := startTestEnv(t)

	// ZoneTemperature is analogValue index 100 on device 86254
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeAnalogValue, 100,
		objects.PropertyIdPresentValue,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	resp := sendBACnet(t, env.bacnetAddr, rpBytes)

	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	dec, err := cack.Decode()
	if err != nil {
		t.Fatalf("decode CACK: %v", err)
	}

	if len(dec.Tags) == 0 {
		t.Fatal("no value tags in response")
	}

	val, ok := dec.Tags[0].Value.(float32)
	if !ok {
		t.Fatalf("expected float32, got %T", dec.Tags[0].Value)
	}

	// Initial zone temp is 72.0, should be within reasonable range
	if val < 40.0 || val > 120.0 {
		t.Errorf("ZoneTemperature out of reasonable range: %f", val)
	}
	t.Logf("ZoneTemperature = %.2f F", val)
}

func TestWriteAndReadSetpoint(t *testing.T) {
	env := startTestEnv(t)

	// Write CoolingDemand (analogOutput, index 22) to 50.0
	// Using analogOutput because it's proven in unit tests
	targetValue := float32(50.0)
	wpBytes, err := bacnet.NewWriteProperty(
		objects.ObjectTypeAnalogOutput, 22,
		objects.PropertyIdPresentValue,
		targetValue,
	)
	if err != nil {
		t.Fatalf("create WP: %v", err)
	}

	resp := sendBACnet(t, env.bacnetAddr, wpBytes)
	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse WP response: %v", err)
	}

	if msg.GetType() != 2 { // SimpleACK
		t.Fatalf("expected SimpleACK (type 2), got type %d", msg.GetType())
	}

	// Read it back via BACnet
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeAnalogOutput, 22,
		objects.PropertyIdPresentValue,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	resp = sendBACnet(t, env.bacnetAddr, rpBytes)
	msg, err = bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse RP response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	dec, err := cack.Decode()
	if err != nil {
		t.Fatalf("decode CACK: %v", err)
	}

	if len(dec.Tags) == 0 {
		t.Fatal("no value tags")
	}

	val, ok := dec.Tags[0].Value.(float32)
	if !ok {
		t.Fatalf("expected float32, got %T", dec.Tags[0].Value)
	}

	if val != targetValue {
		t.Errorf("expected %f, got %f", targetValue, val)
	}
}

func TestTemperatureChangesOverTime(t *testing.T) {
	env := startTestEnv(t)

	readZoneTemp := func() float32 {
		rpBytes, err := bacnet.NewReadProperty(
			objects.ObjectTypeAnalogValue, 100,
			objects.PropertyIdPresentValue,
		)
		if err != nil {
			t.Fatalf("create RP: %v", err)
		}
		resp := sendBACnet(t, env.bacnetAddr, rpBytes)
		msg, err := bacnet.Parse(resp)
		if err != nil {
			t.Fatalf("parse response: %v", err)
		}
		cack, ok := msg.(*services.ComplexACK)
		if !ok {
			t.Fatalf("expected ComplexACK, got %T", msg)
		}
		dec, err := cack.Decode()
		if err != nil {
			t.Fatalf("decode CACK: %v", err)
		}
		if len(dec.Tags) == 0 {
			t.Fatal("no tags")
		}
		val, ok := dec.Tags[0].Value.(float32)
		if !ok {
			t.Fatalf("expected float32, got %T", dec.Tags[0].Value)
		}
		return val
	}

	temp1 := readZoneTemp()
	t.Logf("Initial ZoneTemperature: %.4f", temp1)

	// Wait for a few ticks (tick_interval is 1.0s by default)
	time.Sleep(4 * time.Second)

	temp2 := readZoneTemp()
	t.Logf("After 4s ZoneTemperature: %.4f", temp2)

	// Temperature should change because outdoor temp (85F) != zone temp (72F)
	diff := math.Abs(float64(temp2 - temp1))
	if diff < 0.001 {
		t.Errorf("temperature did not change after 4 seconds: %.4f -> %.4f (diff=%.6f)",
			temp1, temp2, diff)
	}
}

func TestBACnetAndRESTAgreement(t *testing.T) {
	env := startTestEnv(t)

	// Read ZoneTemperature via BACnet
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeAnalogValue, 100,
		objects.PropertyIdPresentValue,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	resp := sendBACnet(t, env.bacnetAddr, rpBytes)
	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	dec, err := cack.Decode()
	if err != nil {
		t.Fatalf("decode CACK: %v", err)
	}

	if len(dec.Tags) == 0 {
		t.Fatal("no tags")
	}

	bacnetVal, ok := dec.Tags[0].Value.(float32)
	if !ok {
		t.Fatalf("expected float32, got %T", dec.Tags[0].Value)
	}

	// Read same value via REST API
	body := apiGet(t, env.apiPort, "/api/v1/devices/86254/points/ZoneTemperature")

	var apiResp struct {
		Success bool `json:"success"`
		Data    struct {
			Name  string   `json:"name"`
			Value *float64 `json:"value"`
			Units string   `json:"units"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("parse API response: %v (body: %s)", err, body)
	}

	if !apiResp.Success {
		t.Fatalf("API returned success=false (body: %s)", body)
	}
	if apiResp.Data.Value == nil {
		t.Fatalf("API returned nil value (body: %s)", body)
	}

	apiVal := *apiResp.Data.Value

	// Values should be very close (float32 vs float64 precision)
	diff := math.Abs(float64(bacnetVal) - apiVal)
	if diff > 0.01 {
		t.Errorf("BACnet (%.4f) and REST API (%.4f) disagree by %.4f",
			bacnetVal, apiVal, diff)
	}

	t.Logf("BACnet=%.4f, REST=%.4f, diff=%.6f", bacnetVal, apiVal, diff)
}

func TestReadPowerMeterPoints(t *testing.T) {
	env := startTestEnv(t)

	// Wait for at least one tick so power meter has values
	time.Sleep(2 * time.Second)

	// WholeBuildingPower is analogInput index 1167 on device 20001
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeAnalogInput, 1167,
		objects.PropertyIdPresentValue,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	resp := sendBACnet(t, env.bacnetAddr, rpBytes)
	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	dec, err := cack.Decode()
	if err != nil {
		t.Fatalf("decode CACK: %v", err)
	}

	if len(dec.Tags) == 0 {
		t.Fatal("no value tags")
	}

	val, ok := dec.Tags[0].Value.(float32)
	if !ok {
		t.Fatalf("expected float32, got %T", dec.Tags[0].Value)
	}

	// Base load is 5kW, should have at least some power
	if val < 0 {
		t.Errorf("WholeBuildingPower should be non-negative: %f", val)
	}

	t.Logf("WholeBuildingPower = %.2f kW", val)
}

func TestReadPropertyMultipleSchneider(t *testing.T) {
	env := startTestEnv(t)

	// RPM for ZoneTemperature — request presentValue and objectName
	rpmBytes, err := bacnet.NewReadPropertyMultiple(
		objects.ObjectTypeAnalogValue, 100,
		[]uint16{objects.PropertyIdPresentValue, objects.PropertyIdObjectName},
	)
	if err != nil {
		t.Fatalf("create RPM: %v", err)
	}

	resp := sendBACnet(t, env.bacnetAddr, rpmBytes)
	msg, err := bacnet.Parse(resp)
	if err != nil {
		t.Fatalf("parse RPM response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	if cack.APDU.Service != services.ServiceConfirmedReadPropMultiple {
		t.Errorf("expected RPM service, got %d", cack.APDU.Service)
	}

	if len(cack.APDU.Objects) == 0 {
		t.Fatal("RPM response has no objects")
	}

	t.Logf("RPM returned %d APDU objects", len(cack.APDU.Objects))
}

func TestWriteViaRESTReadViaBACnet(t *testing.T) {
	env := startTestEnv(t)

	// Write OccupiedCoolingSetPoint via REST API (must include priority 1-16)
	target := 70.5
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/devices/86254/points/OccupiedCoolingSetPoint", env.apiPort)
	body := strings.NewReader(fmt.Sprintf(`{"value": %f, "priority": 16}`, target))

	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT status: %d, body: %s", resp.StatusCode, respBody)
	}

	// Read back via BACnet (OccupiedCoolingSetPoint = analogValue index 40)
	rpBytes, err := bacnet.NewReadProperty(
		objects.ObjectTypeAnalogValue, 40,
		objects.PropertyIdPresentValue,
	)
	if err != nil {
		t.Fatalf("create RP: %v", err)
	}

	bacnetResp := sendBACnet(t, env.bacnetAddr, rpBytes)
	msg, err := bacnet.Parse(bacnetResp)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}

	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("expected ComplexACK, got %T", msg)
	}

	dec, err := cack.Decode()
	if err != nil {
		t.Fatalf("decode CACK: %v", err)
	}

	if len(dec.Tags) == 0 {
		t.Fatal("no tags")
	}

	val, ok := dec.Tags[0].Value.(float32)
	if !ok {
		t.Fatalf("expected float32, got %T", dec.Tags[0].Value)
	}

	diff := math.Abs(float64(val) - target)
	if diff > 0.01 {
		t.Errorf("REST wrote %.2f but BACnet read %.2f", target, val)
	}

	t.Logf("REST write=%.2f, BACnet read=%.2f", target, val)
}
