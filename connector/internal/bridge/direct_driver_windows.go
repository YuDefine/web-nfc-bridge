//go:build windows

package bridge

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// DirectDriver bypasses SCardSvr and talks directly to smart card readers
// via SetupAPI enumeration and DeviceIoControl IOCTLs. This is needed on
// Windows machines where SCardSvr is broken or returns only generic smart card
// drivers (observed on both ARM64 and some x86 systems).
type DirectDriver struct {
	events          chan Event
	stop            chan struct{}
	opMu            sync.Mutex
	lastReaderSet   string
	lastCardPresent bool
	// devicePaths caches enumerated reader paths; refreshed by monitor
	devicePaths []string
}

func NewDirectDriver() (*DirectDriver, error) {
	d := &DirectDriver{
		events: make(chan Event, 32),
		stop:   make(chan struct{}),
	}

	// Initial enumeration to verify we can find readers
	paths, err := enumerateSmartCardReaders()
	if err != nil {
		return nil, fmt.Errorf("direct driver init: %w", err)
	}
	d.devicePaths = paths
	log.Printf("direct: found %d reader(s)", len(paths))
	for i, p := range paths {
		log.Printf("direct: reader[%d] = %s (%s)", i, friendlyReaderName(p), p)
	}

	go d.monitor()
	return d, nil
}

func (d *DirectDriver) DriverName() string {
	return "direct"
}

func (d *DirectDriver) Health(context.Context) map[string]any {
	d.opMu.Lock()
	defer d.opMu.Unlock()

	paths, err := enumerateSmartCardReaders()
	if err != nil {
		return map[string]any{
			"status": "degraded",
			"driver": d.DriverName(),
			"error":  err.Error(),
		}
	}
	d.devicePaths = paths

	status := "ok"
	if len(paths) == 0 {
		status = "degraded"
	}
	return map[string]any{
		"status":  status,
		"driver":  d.DriverName(),
		"readers": len(paths),
	}
}

func (d *DirectDriver) ListReaders(context.Context) ([]Reader, error) {
	d.opMu.Lock()
	defer d.opMu.Unlock()
	return d.listReadersLocked()
}

func (d *DirectDriver) listReadersLocked() ([]Reader, error) {
	paths, err := enumerateSmartCardReaders()
	if err != nil {
		return []Reader{}, nil
	}
	d.devicePaths = paths

	items := make([]Reader, 0, len(paths))
	for _, path := range paths {
		name := friendlyReaderName(path)
		cardPresent := d.probeCardPresent(path)
		items = append(items, Reader{
			Name:       name,
			Driver:     d.DriverName(),
			CardPresent: cardPresent,
		})
	}
	return items, nil
}

func (d *DirectDriver) ConnectSession(_ context.Context, readerName string) (*Session, error) {
	d.opMu.Lock()
	defer d.opMu.Unlock()

	devicePath, err := d.resolveReaderLocked(readerName)
	if err != nil {
		return nil, err
	}

	handle, err := openSmartCardReader(devicePath)
	if err != nil {
		return nil, fmt.Errorf("open reader: %w", err)
	}
	defer procCloseHandle.Call(uintptr(handle))

	// POWER + SET_PROTOCOL to verify card is present and in a usable state.
	// POWER is non-blocking (unlike IS_PRESENT which blocks on contactless).
	_, powerErr := smartCardPower(handle, scardColdReset)
	if powerErr != nil {
		return nil, errors.New("no card present")
	}
	// SET_PROTOCOL transitions card from "negotiable" (state 5) to "specific" (state 6),
	// which is required before TRANSMIT will work.
	_, _ = smartCardSetProtocol(handle, scardProtocolT0|scardProtocolT1)

	name := friendlyReaderName(devicePath)
	session := NewSession(name, d.DriverName())

	d.emit(Event{
		Type:      "reader.status",
		Status:    "ready",
		Reader:    &Reader{Name: name, Driver: d.DriverName(), CardPresent: true},
		SessionID: session.ID,
	})
	return session, nil
}

func (d *DirectDriver) ReadCard(_ context.Context, session *Session, operation string) (*CardReadResult, error) {
	d.opMu.Lock()
	defer d.opMu.Unlock()

	devicePath, err := d.resolveReaderLocked(session.ReaderName)
	if err != nil {
		return nil, err
	}

	handle, err := openSmartCardReader(devicePath)
	if err != nil {
		return nil, fmt.Errorf("open reader: %w", err)
	}

	// Power on with cold reset, then negotiate protocol
	atr, err := smartCardPower(handle, scardColdReset)
	if err != nil {
		procCloseHandle.Call(uintptr(handle))
		return nil, fmt.Errorf("power on: %w", err)
	}

	negotiated, err := smartCardSetProtocol(handle, scardProtocolT0|scardProtocolT1)
	if err != nil {
		procCloseHandle.Call(uintptr(handle))
		return nil, fmt.Errorf("set protocol: %w", err)
	}
	if negotiated == 0 {
		negotiated = scardProtocolT1 // fallback
	}

	card := &directCard{handle: handle, protocol: negotiated}
	defer card.Close()

	// Get UID
	uidResponse, err := card.Transmit([]byte{0xFF, 0xCA, 0x00, 0x00, 0x00})
	if err != nil {
		return nil, fmt.Errorf("get uid: %w", err)
	}
	uid := parseUID(uidResponse)

	result := &CardReadResult{
		SessionID: session.ID,
		Reader:    session.ReaderName,
		Operation: operation,
		UID:       uid,
		ATR:       strings.ToUpper(hex.EncodeToString(atr)),
		Details: map[string]string{
			"driver":   d.DriverName(),
			"protocol": fmt.Sprintf("%d", scardProtocolT1),
		},
	}

	ndefResult := readNDEFWithRetry(card)
	result.MediaType = ndefResult.MediaType
	result.Payload = ndefResult.Payload
	if ndefResult.Err != nil {
		result.Details["ndefReadError"] = ndefResult.Err.Error()
	}

	d.emit(Event{
		Type:      "card.read.complete",
		Status:    "ok",
		Reader:    &Reader{Name: session.ReaderName, Driver: d.DriverName(), CardPresent: true},
		SessionID: session.ID,
		Payload: map[string]any{
			"uid":       uid,
			"mediaType": result.MediaType,
			"payload":   result.Payload,
		},
	})

	return result, nil
}

func (d *DirectDriver) WriteCard(_ context.Context, session *Session, request *WriteRequest) (*CardWriteResult, error) {
	d.opMu.Lock()
	defer d.opMu.Unlock()

	devicePath, err := d.resolveReaderLocked(session.ReaderName)
	if err != nil {
		return nil, err
	}

	handle, err := openSmartCardReader(devicePath)
	if err != nil {
		return nil, fmt.Errorf("open reader: %w", err)
	}

	_, err = smartCardPower(handle, scardColdReset)
	if err != nil {
		procCloseHandle.Call(uintptr(handle))
		return nil, fmt.Errorf("power on: %w", err)
	}

	negotiated, err := smartCardSetProtocol(handle, scardProtocolT0|scardProtocolT1)
	if err != nil {
		procCloseHandle.Call(uintptr(handle))
		return nil, fmt.Errorf("set protocol: %w", err)
	}
	if negotiated == 0 {
		negotiated = scardProtocolT1
	}

	card := &directCard{handle: handle, protocol: negotiated}

	capability, err := readType2Capability(card)
	if err != nil {
		card.Close()
		return &CardWriteResult{
			SessionID: session.ID,
			Reader:    session.ReaderName,
			Operation: request.Operation,
			Accepted:  false,
			Details: map[string]string{
				"driver":      d.DriverName(),
				"profile":     request.Profile,
				"mediaType":   request.MediaType,
				"payloadType": request.PayloadType,
				"reason":      err.Error(),
			},
		}, nil
	}
	if capability.ReadOnly {
		card.Close()
		return &CardWriteResult{
			SessionID: session.ID,
			Reader:    session.ReaderName,
			Operation: request.Operation,
			Accepted:  false,
			Details: map[string]string{
				"driver":      d.DriverName(),
				"profile":     request.Profile,
				"mediaType":   request.MediaType,
				"payloadType": request.PayloadType,
				"reason":      "card is NDEF read-only",
			},
		}, nil
	}

	ndefMessage := buildNDEFMessage(request)
	tlv := buildType2TLV(ndefMessage)
	if len(tlv) > capability.DataAreaBytes {
		card.Close()
		return &CardWriteResult{
			SessionID: session.ID,
			Reader:    session.ReaderName,
			Operation: request.Operation,
			Accepted:  false,
			Details: map[string]string{
				"driver":      d.DriverName(),
				"profile":     request.Profile,
				"mediaType":   request.MediaType,
				"payloadType": request.PayloadType,
				"reason":      fmt.Sprintf("payload requires %d bytes but card data area is %d bytes", len(tlv), capability.DataAreaBytes),
			},
		}, nil
	}

	if err := writeType2Pages(card, type2UserDataPage, tlv); err != nil {
		var writeResponseErr *type2WriteResponseError
		if errors.As(err, &writeResponseErr) {
			card.Close()
			return &CardWriteResult{
				SessionID: session.ID,
				Reader:    session.ReaderName,
				Operation: request.Operation,
				Accepted:  false,
				Details: map[string]string{
					"driver":       d.DriverName(),
					"profile":      request.Profile,
					"mediaType":    request.MediaType,
					"payloadType":  request.PayloadType,
					"reason":       writeResponseErr.Error(),
					"rejectedPage": fmt.Sprintf("%d", writeResponseErr.Page),
					"response":     strings.ToUpper(hex.EncodeToString(writeResponseErr.Response)),
				},
			}, nil
		}
		card.Close()
		return nil, err
	}

	// Close card, wait, re-open for verify
	card.Close()
	time.Sleep(120 * time.Millisecond)

	handle2, err := openSmartCardReader(devicePath)
	if err != nil {
		return nil, fmt.Errorf("reopen for verify: %w", err)
	}
	_, err = smartCardPower(handle2, scardColdReset)
	if err != nil {
		procCloseHandle.Call(uintptr(handle2))
		return nil, fmt.Errorf("power on for verify: %w", err)
	}
	verifyProto, err := smartCardSetProtocol(handle2, scardProtocolT0|scardProtocolT1)
	if err != nil {
		procCloseHandle.Call(uintptr(handle2))
		return nil, fmt.Errorf("set protocol for verify: %w", err)
	}
	if verifyProto == 0 {
		verifyProto = scardProtocolT1
	}

	verifyCard := &directCard{handle: handle2, protocol: verifyProto}
	defer verifyCard.Close()

	if err := verifyType2Write(verifyCard, tlv); err != nil {
		return nil, err
	}

	result := &CardWriteResult{
		SessionID: session.ID,
		Reader:    session.ReaderName,
		Operation: request.Operation,
		Accepted:  true,
		Details: map[string]string{
			"driver":       d.DriverName(),
			"profile":      request.Profile,
			"mediaType":    request.MediaType,
			"payloadType":  request.PayloadType,
			"ndefBytes":    fmt.Sprintf("%d", len(ndefMessage)),
			"tlvBytes":     fmt.Sprintf("%d", len(tlv)),
			"pagesWritten": fmt.Sprintf("%d", requiredType2Pages(len(tlv))),
		},
	}

	d.emit(Event{
		Type:      "card.write.complete",
		Status:    "ok",
		Reader:    &Reader{Name: session.ReaderName, Driver: d.DriverName(), CardPresent: true},
		SessionID: session.ID,
		Payload: map[string]any{
			"operation":   request.Operation,
			"profile":     request.Profile,
			"payloadType": request.PayloadType,
		},
	})
	return result, nil
}

func (d *DirectDriver) Events() <-chan Event {
	return d.events
}

func (d *DirectDriver) Close() error {
	close(d.stop)
	return nil
}

func (d *DirectDriver) monitor() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if !d.opMu.TryLock() {
				continue
			}
			readers, err := d.listReadersLocked()
			d.opMu.Unlock()
			if err != nil {
				d.emit(Event{Type: "error", Status: "error", Payload: map[string]any{"message": err.Error()}})
				continue
			}

			currentSet := ""
			if len(readers) > 0 {
				currentSet = readers[0].Name
			}

			if currentSet != d.lastReaderSet {
				status := "offline"
				var reader *Reader
				if len(readers) > 0 {
					status = "ready"
					reader = &readers[0]
				}
				d.emit(Event{Type: "reader.status", Status: status, Reader: reader})
				d.lastReaderSet = currentSet
			}

			cardPresent := len(readers) > 0 && readers[0].CardPresent
			if cardPresent != d.lastCardPresent {
				eventType := "card.removed"
				if cardPresent {
					eventType = "card.present"
				}
				d.emit(Event{Type: eventType, Status: map[bool]string{true: "ready", false: "idle"}[cardPresent], Reader: firstReader(readers)})
				d.lastCardPresent = cardPresent
			}
		case <-d.stop:
			close(d.events)
			return
		}
	}
}

func (d *DirectDriver) emit(event Event) {
	event.At = time.Now().UTC().Format(time.RFC3339)
	select {
	case d.events <- event:
	default:
	}
}

func (d *DirectDriver) resolveReaderLocked(readerName string) (string, error) {
	if len(d.devicePaths) == 0 {
		paths, err := enumerateSmartCardReaders()
		if err != nil {
			return "", fmt.Errorf("enumerate readers: %w", err)
		}
		d.devicePaths = paths
	}
	if len(d.devicePaths) == 0 {
		return "", errors.New("no readers available")
	}

	if readerName == "" {
		return d.devicePaths[0], nil
	}

	// Match by friendly name or exact device path
	for _, path := range d.devicePaths {
		if path == readerName || friendlyReaderName(path) == readerName {
			return path, nil
		}
	}

	// Fallback: if only one reader, use it
	if len(d.devicePaths) == 1 {
		log.Printf("direct: reader %q not found, using %s", readerName, friendlyReaderName(d.devicePaths[0]))
		return d.devicePaths[0], nil
	}

	return "", fmt.Errorf("reader %q not found", readerName)
}

// probeCardPresent checks if a card is in the reader's field by attempting
// POWER with cold reset. This is non-blocking, unlike IS_PRESENT which blocks
// indefinitely on contactless interfaces waiting for card insertion.
func (d *DirectDriver) probeCardPresent(devicePath string) bool {
	handle, err := openSmartCardReader(devicePath)
	if err != nil {
		return false
	}
	defer procCloseHandle.Call(uintptr(handle))
	_, powerErr := smartCardPower(handle, scardColdReset)
	return powerErr == nil
}

// friendlyReaderName extracts a readable name from a device path.
// Device paths look like: \\?\USB#VID_072F&PID_223B&MI_01#...#{guid}
// We extract VID/PID/MI portion and produce something like "USB Smart Card (072F:223B:01)"
func friendlyReaderName(devicePath string) string {
	upper := strings.ToUpper(devicePath)

	vid := extractToken(upper, "VID_", "&")
	pid := extractToken(upper, "PID_", "&")
	mi := extractToken(upper, "MI_", "#")

	if vid != "" && pid != "" {
		name := vid + ":" + pid
		if mi != "" {
			name += ":" + mi
		}
		return "Smart Card (" + name + ")"
	}

	// Fallback: return a truncated version of the path
	if len(devicePath) > 60 {
		return devicePath[:60] + "..."
	}
	return devicePath
}

func extractToken(s, prefix, suffix string) string {
	idx := strings.Index(s, prefix)
	if idx < 0 {
		return ""
	}
	start := idx + len(prefix)
	end := strings.Index(s[start:], suffix)
	if end < 0 {
		// Take until end of string
		return s[start:]
	}
	return s[start : start+end]
}


// Ensure DirectDriver implements Driver at compile time.
var _ Driver = (*DirectDriver)(nil)
