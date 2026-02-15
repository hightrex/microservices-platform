package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateHash_Deterministic(t *testing.T) {
	log := createTestLog()

	hash1 := GenerateHash(nil, log)
	hash2 := GenerateHash(nil, log)

	assert.Equal(t, hash1, hash2, "Hash should be deterministic for same input")
	assert.Len(t, hash1, 64, "SHA-256 hex digest should be 64 chars")
}

func TestGenerateHash_DifferentWithPreviousHash(t *testing.T) {
	log := createTestLog()

	hashNoPrev := GenerateHash(nil, log)
	prevHash := "abc123"
	hashWithPrev := GenerateHash(&prevHash, log)

	assert.NotEqual(t, hashNoPrev, hashWithPrev, "Hash should differ when previous hash is included")
}

func TestGenerateHash_DifferentForDifferentData(t *testing.T) {
	log1 := createTestLog()
	log2 := createTestLog()
	log2.Action = "different_action"

	hash1 := GenerateHash(nil, log1)
	hash2 := GenerateHash(nil, log2)

	assert.NotEqual(t, hash1, hash2, "Hash should differ for different action")
}

func TestVerifyHash_Valid(t *testing.T) {
	log := createTestLog()
	log.CurrentHash = GenerateHash(nil, log)

	assert.True(t, VerifyHash(log), "Correctly generated hash should verify")
}

func TestVerifyHash_Tampered(t *testing.T) {
	log := createTestLog()
	log.CurrentHash = GenerateHash(nil, log)

	// Tamper with the data
	log.Action = "tampered_action"

	assert.False(t, VerifyHash(log), "Tampered log should fail verification")
}

func TestVerifyChain_ValidChain(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()

	// Build a valid chain of 5 logs
	var logs []models.AuditLog
	var prevHash *string

	for i := 0; i < 5; i++ {
		log := &models.AuditLog{
			ID:            uuid.New(),
			TenantID:      tenantID,
			EventType:     "test.event",
			EventCategory: models.CategorySystem,
			ActorID:       "user-123",
			ActorType:     models.ActorUser,
			Action:        "test",
			Outcome:       models.OutcomeSuccess,
			Timestamp:     now.Add(time.Duration(i) * time.Second),
			PreviousHash:  prevHash,
		}
		log.CurrentHash = GenerateHash(prevHash, log)
		logs = append(logs, *log)
		prevHash = &log.CurrentHash
	}

	report := VerifyChain(logs)

	require.True(t, report.Valid, "Valid chain should pass verification")
	assert.Equal(t, 5, report.TotalLogs)
	assert.Equal(t, 5, report.Verified)
	assert.Nil(t, report.BrokenAt)
}

func TestVerifyChain_BrokenChain(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()

	var logs []models.AuditLog
	var prevHash *string

	for i := 0; i < 5; i++ {
		log := &models.AuditLog{
			ID:            uuid.New(),
			TenantID:      tenantID,
			EventType:     "test.event",
			EventCategory: models.CategorySystem,
			ActorID:       "user-123",
			ActorType:     models.ActorUser,
			Action:        "test",
			Outcome:       models.OutcomeSuccess,
			Timestamp:     now.Add(time.Duration(i) * time.Second),
			PreviousHash:  prevHash,
		}
		log.CurrentHash = GenerateHash(prevHash, log)
		logs = append(logs, *log)
		prevHash = &log.CurrentHash
	}

	// Tamper with the 3rd log's hash
	logs[2].CurrentHash = "tampered_hash_value_that_does_not_match"

	report := VerifyChain(logs)

	assert.False(t, report.Valid, "Tampered chain should fail verification")
	require.NotNil(t, report.BrokenAt)
	assert.Equal(t, 2, *report.BrokenAt, "Should detect break at index 2")
}

func TestVerifyChain_EmptyLogs(t *testing.T) {
	report := VerifyChain([]models.AuditLog{})

	assert.True(t, report.Valid, "Empty log list should be valid")
	assert.Equal(t, 0, report.TotalLogs)
}

func TestVerifyChain_SingleLog(t *testing.T) {
	log := createTestLog()
	log.CurrentHash = GenerateHash(nil, log)

	report := VerifyChain([]models.AuditLog{*log})

	assert.True(t, report.Valid, "Single valid log should pass verification")
	assert.Equal(t, 1, report.Verified)
}

func createTestLog() *models.AuditLog {
	return &models.AuditLog{
		ID:            uuid.New(),
		TenantID:      uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		EventType:     "user.created",
		EventCategory: models.CategoryAuth,
		ActorID:       "actor-123",
		ActorType:     models.ActorUser,
		Action:        "create",
		Outcome:       models.OutcomeSuccess,
		Timestamp:     time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC),
	}
}
