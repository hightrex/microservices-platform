package service

import (
	"crypto/sha256"
	"fmt"

	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
)

// GenerateHash generates a SHA-256 hash for an audit log entry, incorporating the previous hash
// to form a hash chain that ensures tamper detection.
//
// Hash format: SHA-256 of "{previous_hash}|{tenant_id}|{timestamp}|{event_type}|{actor_id}|{resource_id}|{action}|{outcome}"
func GenerateHash(prevHash *string, log *models.AuditLog) string {
	prev := ""
	if prevHash != nil {
		prev = *prevHash
	}

	resourceID := ""
	if log.ResourceID != nil {
		resourceID = *log.ResourceID
	}

	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		prev,
		log.TenantID.String(),
		log.Timestamp.UTC().Format("2006-01-02T15:04:05.000000Z"),
		log.EventType,
		log.ActorID,
		resourceID,
		log.Action,
		string(log.Outcome),
	)

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// VerifyHash recalculates the hash for a single log entry and compares it to the stored hash.
func VerifyHash(log *models.AuditLog) bool {
	recalculated := GenerateHash(log.PreviousHash, log)
	return recalculated == log.CurrentHash
}

// VerifyChain verifies the integrity of a sequence of audit logs.
// Logs must be in chronological order (ascending).
func VerifyChain(logs []models.AuditLog) *models.VerificationReport {
	report := &models.VerificationReport{
		TotalLogs: len(logs),
		Valid:     true,
	}

	if len(logs) == 0 {
		return report
	}

	for i, log := range logs {
		// Verify each log's hash is correct
		if !VerifyHash(&log) {
			report.Valid = false
			report.BrokenAt = &i
			report.ErrorDetail = fmt.Sprintf("Hash mismatch at log %s (index %d): stored=%s", log.ID.String(), i, log.CurrentHash)
			return report
		}

		// Verify chain continuity (each log's previous_hash matches prior log's current_hash)
		if i > 0 {
			prevLog := logs[i-1]
			if log.PreviousHash == nil || *log.PreviousHash != prevLog.CurrentHash {
				report.Valid = false
				report.BrokenAt = &i
				report.ErrorDetail = fmt.Sprintf("Chain broken at log %s (index %d): expected previous_hash=%s", log.ID.String(), i, prevLog.CurrentHash)
				return report
			}
		}

		report.Verified++
	}

	return report
}
