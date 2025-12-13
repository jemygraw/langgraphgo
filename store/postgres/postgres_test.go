package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/smallnest/langgraphgo/graph"
	"github.com/stretchr/testify/assert"
)

func TestPostgresCheckpointStore_Save(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cp := &graph.Checkpoint{
		ID:        "cp-1",
		NodeName:  "node-a",
		State:     map[string]any{"foo": "bar"},
		Timestamp: time.Now(),
		Version:   1,
		Metadata: map[string]any{
			"execution_id": "exec-1",
		},
	}

	stateJSON, _ := json.Marshal(cp.State)
	metadataJSON, _ := json.Marshal(cp.Metadata)

	// Expect INSERT
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO checkpoints")).
		WithArgs(
			cp.ID,
			"exec-1",
			cp.NodeName,
			stateJSON,
			metadataJSON,
			cp.Timestamp,
			cp.Version,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = store.Save(context.Background(), cp)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_Load(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cpID := "cp-1"
	timestamp := time.Now()
	state := map[string]any{"foo": "bar"}
	metadata := map[string]any{"execution_id": "exec-1"}

	stateJSON, _ := json.Marshal(state)
	metadataJSON, _ := json.Marshal(metadata)

	rows := pgxmock.NewRows([]string{"id", "node_name", "state", "metadata", "timestamp", "version"}).
		AddRow(cpID, "node-a", stateJSON, metadataJSON, timestamp, 1)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, node_name, state, metadata, timestamp, version FROM checkpoints WHERE id = $1")).
		WithArgs(cpID).
		WillReturnRows(rows)

	loaded, err := store.Load(context.Background(), cpID)
	assert.NoError(t, err)
	assert.Equal(t, cpID, loaded.ID)
	assert.Equal(t, "node-a", loaded.NodeName)
	assert.Equal(t, 1, loaded.Version)

	// Check state
	loadedState, ok := loaded.State.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "bar", loadedState["foo"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_Save_WithoutExecutionID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cp := &graph.Checkpoint{
		ID:        "cp-1",
		NodeName:  "node-a",
		State:     map[string]any{"foo": "bar"},
		Timestamp: time.Now(),
		Version:   1,
		Metadata:  map[string]any{}, // No execution_id
	}

	stateJSON, _ := json.Marshal(cp.State)
	metadataJSON, _ := json.Marshal(cp.Metadata)

	// Expect INSERT with empty execution_id
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO checkpoints")).
		WithArgs(
			cp.ID,
			"", // empty execution_id
			cp.NodeName,
			stateJSON,
			metadataJSON,
			cp.Timestamp,
			cp.Version,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = store.Save(context.Background(), cp)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_Save_MarshalError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	// Create invalid state that cannot be marshaled
	cp := &graph.Checkpoint{
		ID:        "cp-1",
		NodeName:  "node-a",
		State:     make(chan int), // channels cannot be marshaled to JSON
		Timestamp: time.Now(),
		Version:   1,
	}

	err = store.Save(context.Background(), cp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal state")
}

func TestPostgresCheckpoint_Load_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cpID := "non-existent"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, node_name, state, metadata, timestamp, version FROM checkpoints WHERE id = $1")).
		WithArgs(cpID).
		WillReturnError(pgx.ErrNoRows)

	loaded, err := store.Load(context.Background(), cpID)
	assert.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "checkpoint not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpoint_Load_DatabaseError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cpID := "cp-1"
	dbError := errors.New("database connection failed")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, node_name, state, metadata, timestamp, version FROM checkpoints WHERE id = $1")).
		WithArgs(cpID).
		WillReturnError(dbError)

	loaded, err := store.Load(context.Background(), cpID)
	assert.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "failed to load checkpoint")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpoint_Load_InvalidStateJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cpID := "cp-1"
	timestamp := time.Now()

	// Create row with invalid JSON
	rows := pgxmock.NewRows([]string{"id", "node_name", "state", "metadata", "timestamp", "version"}).
		AddRow(cpID, "node-a", []byte("{invalid json"), []byte("{}"), timestamp, 1)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, node_name, state, metadata, timestamp, version FROM checkpoints WHERE id = $1")).
		WithArgs(cpID).
		WillReturnRows(rows)

	loaded, err := store.Load(context.Background(), cpID)
	assert.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "failed to unmarshal state")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpoint_Load_InvalidMetadataJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cpID := "cp-1"
	timestamp := time.Now()
	state := map[string]any{"foo": "bar"}
	stateJSON, _ := json.Marshal(state)

	// Create row with invalid metadata JSON
	rows := pgxmock.NewRows([]string{"id", "node_name", "state", "metadata", "timestamp", "version"}).
		AddRow(cpID, "node-a", stateJSON, []byte("{invalid metadata json"), timestamp, 1)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, node_name, state, metadata, timestamp, version FROM checkpoints WHERE id = $1")).
		WithArgs(cpID).
		WillReturnRows(rows)

	loaded, err := store.Load(context.Background(), cpID)
	assert.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "failed to unmarshal metadata")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpoint_Load_NilMetadata(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cpID := "cp-1"
	timestamp := time.Now()
	state := map[string]any{"foo": "bar"}
	stateJSON, _ := json.Marshal(state)

	// Create row with nil metadata
	rows := pgxmock.NewRows([]string{"id", "node_name", "state", "metadata", "timestamp", "version"}).
		AddRow(cpID, "node-a", stateJSON, nil, timestamp, 1)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, node_name, state, metadata, timestamp, version FROM checkpoints WHERE id = $1")).
		WithArgs(cpID).
		WillReturnRows(rows)

	loaded, err := store.Load(context.Background(), cpID)
	assert.NoError(t, err)
	assert.Equal(t, cpID, loaded.ID)
	assert.Equal(t, "node-a", loaded.NodeName)
	assert.Equal(t, 1, loaded.Version)
	assert.NotNil(t, loaded.State)
	// Metadata should be nil when not present in DB (not initialized)
	assert.Nil(t, loaded.Metadata)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	executionID := "exec-1"
	timestamp := time.Now()

	// Create checkpoint data
	checkpoints := []struct {
		id       string
		nodeName string
		state    map[string]any
		metadata map[string]any
		version  int
	}{
		{
			id:       "cp-1",
			nodeName: "node-a",
			state:    map[string]any{"step": 1},
			metadata: map[string]any{"execution_id": executionID},
			version:  1,
		},
		{
			id:       "cp-2",
			nodeName: "node-b",
			state:    map[string]any{"step": 2},
			metadata: map[string]any{"execution_id": executionID},
			version:  2,
		},
	}

	rows := pgxmock.NewRows([]string{"id", "node_name", "state", "metadata", "timestamp", "version"})
	for _, cp := range checkpoints {
		stateJSON, _ := json.Marshal(cp.state)
		metadataJSON, _ := json.Marshal(cp.metadata)
		rows.AddRow(cp.id, cp.nodeName, stateJSON, metadataJSON, timestamp, cp.version)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, node_name, state, metadata, timestamp, version FROM checkpoints WHERE execution_id = $1 ORDER BY timestamp ASC")).
		WithArgs(executionID).
		WillReturnRows(rows)

	loaded, err := store.List(context.Background(), executionID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(loaded))

	// Check first checkpoint
	assert.Equal(t, "cp-1", loaded[0].ID)
	assert.Equal(t, "node-a", loaded[0].NodeName)
	assert.Equal(t, 1, loaded[0].Version)

	// Check second checkpoint
	assert.Equal(t, "cp-2", loaded[1].ID)
	assert.Equal(t, "node-b", loaded[1].NodeName)
	assert.Equal(t, 2, loaded[1].Version)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_List_EmptyResult(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	executionID := "exec-empty"

	rows := pgxmock.NewRows([]string{"id", "node_name", "state", "metadata", "timestamp", "version"})

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, node_name, state, metadata, timestamp, version FROM checkpoints WHERE execution_id = $1 ORDER BY timestamp ASC")).
		WithArgs(executionID).
		WillReturnRows(rows)

	loaded, err := store.List(context.Background(), executionID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(loaded))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_List_DatabaseError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	executionID := "exec-1"
	dbError := errors.New("database connection failed")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, node_name, state, metadata, timestamp, version FROM checkpoints WHERE execution_id = $1 ORDER BY timestamp ASC")).
		WithArgs(executionID).
		WillReturnError(dbError)

	loaded, err := store.List(context.Background(), executionID)
	assert.Error(t, err)
	assert.Nil(t, loaded)
	assert.Contains(t, err.Error(), "failed to list checkpoints")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_List_ScanError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	executionID := "exec-1"

	rows := pgxmock.NewRows([]string{"id", "node_name", "state", "metadata", "timestamp", "version"}).
		AddRow("cp-1", "node-a", []byte("{invalid"), []byte("{}"), time.Now(), 1).
		AddRow("cp-2", "node-b", []byte("{}"), []byte("{}"), time.Now(), 2)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, node_name, state, metadata, timestamp, version FROM checkpoints WHERE execution_id = $1 ORDER BY timestamp ASC")).
		WithArgs(executionID).
		WillReturnRows(rows)

	loaded, err := store.List(context.Background(), executionID)
	assert.Error(t, err)
	assert.Nil(t, loaded)
	// The error occurs during JSON unmarshaling, not scanning
	assert.Contains(t, err.Error(), "failed to unmarshal state")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	checkpointID := "cp-1"

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM checkpoints WHERE id = $1")).
		WithArgs(checkpointID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = store.Delete(context.Background(), checkpointID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_Delete_DatabaseError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	checkpointID := "cp-1"
	dbError := errors.New("database connection failed")

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM checkpoints WHERE id = $1")).
		WithArgs(checkpointID).
		WillReturnError(dbError)

	err = store.Delete(context.Background(), checkpointID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete checkpoint")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_Clear(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	executionID := "exec-1"

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM checkpoints WHERE execution_id = $1")).
		WithArgs(executionID).
		WillReturnResult(pgxmock.NewResult("DELETE", 5)) // 5 rows deleted

	err = store.Clear(context.Background(), executionID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_Clear_DatabaseError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	executionID := "exec-1"
	dbError := errors.New("database connection failed")

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM checkpoints WHERE execution_id = $1")).
		WithArgs(executionID).
		WillReturnError(dbError)

	err = store.Clear(context.Background(), executionID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clear checkpoints")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_InitSchema(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	mock.ExpectExec(regexp.QuoteMeta(`
		CREATE TABLE IF NOT EXISTS checkpoints (
			id TEXT PRIMARY KEY,
			execution_id TEXT NOT NULL,
			node_name TEXT NOT NULL,
			state JSONB NOT NULL,
			metadata JSONB,
			timestamp TIMESTAMPTZ NOT NULL,
			version INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_checkpoints_execution_id ON checkpoints (execution_id);
	`)).
		WillReturnResult(pgxmock.NewResult("CREATE", 0))

	err = store.InitSchema(context.Background())
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_InitSchema_CustomTable(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	tableName := "custom_checkpoints"
	store := NewPostgresCheckpointStoreWithPool(mock, tableName)

	mock.ExpectExec(regexp.QuoteMeta(`
		CREATE TABLE IF NOT EXISTS custom_checkpoints (
			id TEXT PRIMARY KEY,
			execution_id TEXT NOT NULL,
			node_name TEXT NOT NULL,
			state JSONB NOT NULL,
			metadata JSONB,
			timestamp TIMESTAMPTZ NOT NULL,
			version INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_custom_checkpoints_execution_id ON custom_checkpoints (execution_id);
	`)).
		WillReturnResult(pgxmock.NewResult("CREATE", 0))

	err = store.InitSchema(context.Background())
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_InitSchema_DatabaseError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	dbError := errors.New("database connection failed")
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS checkpoints")).
		WillReturnError(dbError)

	err = store.InitSchema(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create schema")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_Close(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	// This should not panic
	assert.NotPanics(t, func() {
		store.Close()
	})
}

func TestNewPostgresCheckpointStoreWithPool_DefaultTableName(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	// Pass empty table name, should default to "checkpoints"
	store := NewPostgresCheckpointStoreWithPool(mock, "")

	assert.NotNil(t, store)
	assert.Equal(t, "checkpoints", store.tableName)
	assert.Equal(t, mock, store.pool)
}

func TestPostgresCheckpointStore_Save_Conflict(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cp := &graph.Checkpoint{
		ID:        "cp-1",
		NodeName:  "node-a",
		State:     map[string]any{"foo": "bar"},
		Timestamp: time.Now(),
		Version:   2, // Different version
		Metadata: map[string]any{
			"execution_id": "exec-1",
		},
	}

	stateJSON, _ := json.Marshal(cp.State)
	metadataJSON, _ := json.Marshal(cp.Metadata)

	// Expect UPDATE due to conflict
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO checkpoints")).
		WithArgs(
			cp.ID,
			"exec-1",
			cp.NodeName,
			stateJSON,
			metadataJSON,
			cp.Timestamp,
			cp.Version,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = store.Save(context.Background(), cp)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_Save_DatabaseError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cp := &graph.Checkpoint{
		ID:        "cp-1",
		NodeName:  "node-a",
		State:     map[string]any{"foo": "bar"},
		Timestamp: time.Now(),
		Version:   1,
		Metadata: map[string]any{
			"execution_id": "exec-1",
		},
	}

	stateJSON, _ := json.Marshal(cp.State)
	metadataJSON, _ := json.Marshal(cp.Metadata)

	dbError := errors.New("database connection failed")
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO checkpoints")).
		WithArgs(
			cp.ID,
			"exec-1",
			cp.NodeName,
			stateJSON,
			metadataJSON,
			cp.Timestamp,
			cp.Version,
		).
		WillReturnError(dbError)

	err = store.Save(context.Background(), cp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save checkpoint")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_Save_MarshalMetadataError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cp := &graph.Checkpoint{
		ID:        "cp-1",
		NodeName:  "node-a",
		State:     map[string]any{"foo": "bar"},
		Timestamp: time.Now(),
		Version:   1,
		Metadata: map[string]any{
			"invalid": make(chan int), // channels cannot be marshaled
		},
	}

	err = store.Save(context.Background(), cp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal metadata")
}

func TestNewPostgresCheckpointStore_InvalidConnection(t *testing.T) {
	ctx := context.Background()
	opts := PostgresOptions{
		ConnString: "invalid://connection-string",
		TableName:  "test_checkpoints",
	}

	// This should return an error due to invalid connection string
	_, err := NewPostgresCheckpointStore(ctx, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to create connection pool")
}
