package bridge

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Reader struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	CardPresent bool   `json:"cardPresent"`
}

type CardReadResult struct {
	SessionID string            `json:"sessionId"`
	Reader    string            `json:"reader"`
	Operation string            `json:"operation"`
	UID       string            `json:"uid,omitempty"`
	ATR       string            `json:"atr,omitempty"`
	MediaType string            `json:"mediaType,omitempty"`
	Payload   map[string]any    `json:"payload,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

type CardWriteResult struct {
	SessionID string            `json:"sessionId"`
	Reader    string            `json:"reader"`
	Operation string            `json:"operation"`
	Accepted  bool              `json:"accepted"`
	Details   map[string]string `json:"details,omitempty"`
}

type WriteRequest struct {
	Operation      string         `json:"operation"`
	Profile        string         `json:"profile"`
	MediaType      string         `json:"mediaType"`
	PayloadType    string         `json:"payloadType"`
	Payload        map[string]any `json:"payload"`
	EncodedPayload []byte         `json:"-"`
}

type Event struct {
	Type      string         `json:"type"`
	Status    string         `json:"status,omitempty"`
	Reader    *Reader        `json:"reader,omitempty"`
	SessionID string         `json:"sessionId,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	At        string         `json:"at"`
}

type Driver interface {
	DriverName() string
	Health(context.Context) map[string]any
	ListReaders(context.Context) ([]Reader, error)
	ConnectSession(context.Context, string) (*Session, error)
	ReadCard(context.Context, *Session, string) (*CardReadResult, error)
	WriteCard(context.Context, *Session, *WriteRequest) (*CardWriteResult, error)
	Events() <-chan Event
	Close() error
}

type Session struct {
	ID         string    `json:"id"`
	ReaderName string    `json:"readerName"`
	Driver     string    `json:"driver"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Service struct {
	driver   Driver
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewService(driver Driver) *Service {
	return &Service{
		driver:   driver,
		sessions: map[string]*Session{},
	}
}

func (s *Service) DriverName() string {
	return s.driver.DriverName()
}

func (s *Service) Health(ctx context.Context) map[string]any {
	return s.driver.Health(ctx)
}

func (s *Service) Readers(ctx context.Context) ([]Reader, error) {
	return s.driver.ListReaders(ctx)
}

func (s *Service) OpenSession(ctx context.Context, readerName string) (*Session, error) {
	session, err := s.driver.ConnectSession(ctx, readerName)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return session, nil
}

func (s *Service) Read(ctx context.Context, sessionID string, operation string) (*CardReadResult, error) {
	session, err := s.session(sessionID)
	if err != nil {
		return nil, err
	}

	return s.driver.ReadCard(ctx, session, operation)
}

func (s *Service) Write(ctx context.Context, sessionID string, operation string, payload map[string]any) (*CardWriteResult, error) {
	session, err := s.session(sessionID)
	if err != nil {
		return nil, err
	}

	request, err := ValidateWriteRequest(operation, payload)
	if err != nil {
		return nil, err
	}

	return s.driver.WriteCard(ctx, session, request)
}

func (s *Service) Events() <-chan Event {
	return s.driver.Events()
}

func (s *Service) Close() error {
	return s.driver.Close()
}

func (s *Service) session(sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	return session, nil
}

func NewSession(readerName string, driverName string) *Session {
	return &Session{
		ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
		ReaderName: readerName,
		Driver:     driverName,
		CreatedAt:  time.Now().UTC(),
	}
}

// HasNFCCapableReader checks whether a driver's listed readers include at least
// one that looks like a real NFC/contactless reader (e.g. ACR1252 PICC interface)
// rather than a generic Windows smart card driver.
func HasNFCCapableReader(d Driver) bool {
	readers, err := d.ListReaders(context.Background())
	if err != nil || len(readers) == 0 {
		return false
	}
	for _, r := range readers {
		upper := strings.ToUpper(r.Name)
		// Known NFC reader indicators
		if strings.Contains(upper, "PICC") ||
			strings.Contains(upper, "CONTACTLESS") ||
			strings.Contains(upper, "ACR") ||
			strings.Contains(upper, "NFC") {
			return true
		}
	}
	return false
}

func firstReader(readers []Reader) *Reader {
	if len(readers) == 0 {
		return nil
	}
	reader := readers[0]
	return &reader
}