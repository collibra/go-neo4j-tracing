package neo4j_tracing

import (
	"context"
	"time"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j/db"
)

// mockDriver is a mock implementation of the neo4j.Driver interface for testing.
type mockDriver struct {
	neo4j.Driver

	newSessionFunc           func(ctx context.Context, config neo4j.SessionConfig) neo4j.Session
	verifyConnectivityFunc   func(ctx context.Context) error
	verifyAuthenticationFunc func(ctx context.Context, auth *neo4j.AuthToken) error
	getServerInfoFunc        func(ctx context.Context) (neo4j.ServerInfo, error)
	closeFunc                func(ctx context.Context) error
}

func (m *mockDriver) NewSession(_ context.Context, _ neo4j.SessionConfig) neo4j.Session {
	if m.newSessionFunc != nil {
		return m.newSessionFunc(context.Background(), neo4j.SessionConfig{})
	}
	return &mockSession{}
}

func (m *mockDriver) VerifyConnectivity(ctx context.Context) error {
	if m.verifyConnectivityFunc != nil {
		return m.verifyConnectivityFunc(ctx)
	}
	return nil
}

func (m *mockDriver) VerifyAuthentication(ctx context.Context, auth *neo4j.AuthToken) error {
	if m.verifyAuthenticationFunc != nil {
		return m.verifyAuthenticationFunc(ctx, auth)
	}
	return nil
}

func (m *mockDriver) GetServerInfo(ctx context.Context) (neo4j.ServerInfo, error) {
	if m.getServerInfoFunc != nil {
		return m.getServerInfoFunc(ctx)
	}
	return &mockServerInfo{}, nil
}

func (m *mockDriver) Close(ctx context.Context) error {
	if m.closeFunc != nil {
		return m.closeFunc(ctx)
	}
	return nil
}

// mockSession is a mock implementation of the neo4j.Session interface for testing.
type mockSession struct {
	neo4j.Session

	beginTransactionFunc func(ctx context.Context, configurers ...func(config *neo4j.TransactionConfig)) (neo4j.ExplicitTransaction, error)
	executeReadFunc      func(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig)) (any, error)
	executeWriteFunc     func(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig)) (any, error)
	runFunc              func(ctx context.Context, cypher string, params map[string]any, configurers ...func(config *neo4j.TransactionConfig)) (neo4j.Result, error)
	lastBookmarksFunc    func() neo4j.Bookmarks
	closeFunc            func(ctx context.Context) error
}

func (m *mockSession) BeginTransaction(ctx context.Context, configurers ...func(config *neo4j.TransactionConfig)) (neo4j.ExplicitTransaction, error) {
	if m.beginTransactionFunc != nil {
		return m.beginTransactionFunc(ctx, configurers...)
	}
	return &mockExplicitTransaction{}, nil
}

func (m *mockSession) ExecuteRead(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig)) (any, error) {
	if m.executeReadFunc != nil {
		return m.executeReadFunc(ctx, work, configurers...)
	}
	return nil, nil
}

func (m *mockSession) ExecuteWrite(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(config *neo4j.TransactionConfig)) (any, error) {
	if m.executeWriteFunc != nil {
		return m.executeWriteFunc(ctx, work, configurers...)
	}
	return nil, nil
}

func (m *mockSession) Run(ctx context.Context, cypher string, params map[string]any, configurers ...func(config *neo4j.TransactionConfig)) (neo4j.Result, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, cypher, params, configurers...)
	}
	return &mockResult{}, nil
}

func (m *mockSession) LastBookmarks() neo4j.Bookmarks {
	if m.lastBookmarksFunc != nil {
		return m.lastBookmarksFunc()
	}
	return nil
}

func (m *mockSession) Close(ctx context.Context) error {
	if m.closeFunc != nil {
		return m.closeFunc(ctx)
	}
	return nil
}

// mockResult is a mock implementation of the neo4j.Result interface for testing.
type mockResult struct {
	neo4j.Result

	nextRecordFunc func(ctx context.Context, record **neo4j.Record) bool
	nextFunc       func(ctx context.Context) bool
	peekRecordFunc func(ctx context.Context, record **neo4j.Record) bool
	peekFunc       func(ctx context.Context) bool
	collectFunc    func(ctx context.Context) ([]*neo4j.Record, error)
	singleFunc     func(ctx context.Context) (*neo4j.Record, error)
	consumeFunc    func(ctx context.Context) (neo4j.ResultSummary, error)
	errFunc        func() error
	recordFunc     func() *neo4j.Record
	keysFunc       func() ([]string, error)
	isOpenFunc     func() bool
}

func (m *mockResult) NextRecord(ctx context.Context, record **neo4j.Record) bool {
	if m.nextRecordFunc != nil {
		return m.nextRecordFunc(ctx, record)
	}
	return false
}

func (m *mockResult) Next(ctx context.Context) bool {
	if m.nextFunc != nil {
		return m.nextFunc(ctx)
	}
	return false
}

func (m *mockResult) PeekRecord(ctx context.Context, record **neo4j.Record) bool {
	if m.peekRecordFunc != nil {
		return m.peekRecordFunc(ctx, record)
	}
	return false
}

func (m *mockResult) Peek(ctx context.Context) bool {
	if m.peekFunc != nil {
		return m.peekFunc(ctx)
	}
	return false
}

func (m *mockResult) Collect(ctx context.Context) ([]*neo4j.Record, error) {
	if m.collectFunc != nil {
		return m.collectFunc(ctx)
	}
	return nil, nil
}

func (m *mockResult) Single(ctx context.Context) (*neo4j.Record, error) {
	if m.singleFunc != nil {
		return m.singleFunc(ctx)
	}
	return nil, nil
}

func (m *mockResult) Consume(ctx context.Context) (neo4j.ResultSummary, error) {
	if m.consumeFunc != nil {
		return m.consumeFunc(ctx)
	}
	return &mockResultSummary{}, nil
}

func (m *mockResult) Err() error {
	if m.errFunc != nil {
		return m.errFunc()
	}
	return nil
}

func (m *mockResult) Record() *neo4j.Record {
	if m.recordFunc != nil {
		return m.recordFunc()
	}
	return nil
}

func (m *mockResult) Keys() ([]string, error) {
	if m.keysFunc != nil {
		return m.keysFunc()
	}
	return nil, nil
}

func (m *mockResult) IsOpen() bool {
	if m.isOpenFunc != nil {
		return m.isOpenFunc()
	}
	return false
}

// mockManagedTransaction is a mock implementation of the neo4j.ManagedTransaction interface for testing.
type mockManagedTransaction struct {
	neo4j.ManagedTransaction

	runFunc func(ctx context.Context, cypher string, params map[string]any) (neo4j.Result, error)
}

func (m *mockManagedTransaction) Run(ctx context.Context, cypher string, params map[string]any) (neo4j.Result, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, cypher, params)
	}
	return &mockResult{}, nil
}

// mockExplicitTransaction is a mock implementation of the neo4j.ExplicitTransaction interface for testing.
type mockExplicitTransaction struct {
	neo4j.ExplicitTransaction

	runFunc      func(ctx context.Context, cypher string, params map[string]any) (neo4j.Result, error)
	commitFunc   func(ctx context.Context) error
	rollbackFunc func(ctx context.Context) error
	closeFunc    func(ctx context.Context) error
}

func (m *mockExplicitTransaction) Run(ctx context.Context, cypher string, params map[string]any) (neo4j.Result, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, cypher, params)
	}
	return &mockResult{}, nil
}

func (m *mockExplicitTransaction) Commit(ctx context.Context) error {
	if m.commitFunc != nil {
		return m.commitFunc(ctx)
	}
	return nil
}

func (m *mockExplicitTransaction) Rollback(ctx context.Context) error {
	if m.rollbackFunc != nil {
		return m.rollbackFunc(ctx)
	}
	return nil
}

func (m *mockExplicitTransaction) Close(ctx context.Context) error {
	if m.closeFunc != nil {
		return m.closeFunc(ctx)
	}
	return nil
}

// mockServerInfo is a mock implementation of neo4j.ServerInfo.
type mockServerInfo struct {
	neo4j.ServerInfo
}

func (m *mockServerInfo) Address() string {
	return ""
}

func (m *mockServerInfo) Agent() string {
	return ""
}

func (m *mockServerInfo) ProtocolVersion() db.ProtocolVersion {
	return db.ProtocolVersion{}
}

func (m *mockServerInfo) Database() neo4j.DatabaseInfo {
	return &mockDatabaseInfo{}
}

// mockResultSummary is a mock implementation of the neo4j.ResultSummary interface for testing.
type mockResultSummary struct {
	neo4j.ResultSummary
}

func (m *mockResultSummary) Server() neo4j.ServerInfo {
	return &mockServerInfo{}
}

func (m *mockResultSummary) Query() neo4j.Query {
	return nil
}

func (m *mockResultSummary) Counters() neo4j.Counters {
	return nil
}

func (m *mockResultSummary) QueryType() neo4j.QueryType {
	return neo4j.QueryTypeUnknown
}

func (m *mockResultSummary) Plan() neo4j.Plan {
	return nil
}

func (m *mockResultSummary) Profile() neo4j.ProfiledPlan {
	return nil
}

func (m *mockResultSummary) Notifications() []neo4j.Notification { //nolint:staticcheck
	return nil
}

func (m *mockResultSummary) ResultAvailableAfter() time.Duration {
	return 0
}

func (m *mockResultSummary) ResultConsumedAfter() time.Duration {
	return 0
}

func (m *mockResultSummary) Database() neo4j.DatabaseInfo {
	return &mockDatabaseInfo{}
}

func (m *mockResultSummary) LastBookmark() string {
	return ""
}

func (m *mockResultSummary) LastBookmarks() []string {
	return nil
}

// mockDatabaseInfo is a mock implementation of the neo4j.DatabaseInfo interface for testing.
type mockDatabaseInfo struct {
	neo4j.DatabaseInfo
}

func (m *mockDatabaseInfo) Name() string {
	return ""
}
