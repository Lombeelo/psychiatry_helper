package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"example.com/backend/domain"
)

// MedicationGroupWithMedications — группа справочника вместе с препаратами.
type MedicationGroupWithMedications struct {
	Group       domain.MedicationGroup
	Medications []domain.Medication
}

func (s *Store) ListJournalEntriesForPatient(
	ctx context.Context,
	patientID domain.ID,
) ([]domain.JournalEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, patient_id, text, reflection_prompt, created_at, updated_at
		FROM journal_entries
		WHERE patient_id = ?
		ORDER BY created_at DESC`,
		string(patientID),
	)
	if err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}
	defer rows.Close()

	var entries []domain.JournalEntry
	for rows.Next() {
		var (
			entry            domain.JournalEntry
			id, patient      string
			createdAt, updAt string
		)
		if err := rows.Scan(
			&id, &patient, &entry.Text, &entry.ReflectionPrompt,
			&createdAt, &updAt,
		); err != nil {
			return nil, fmt.Errorf("scan journal entry: %w", err)
		}
		entry.ID = domain.ID(id)
		entry.PatientID = domain.ID(patient)
		if entry.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		if entry.UpdatedAt, err = parseTime(updAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate journal entries: %w", err)
	}
	return entries, nil
}

func (s *Store) ListMoodCheckInsForPatient(
	ctx context.Context,
	patientID domain.ID,
) ([]domain.MoodCheckIn, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, patient_id, answers, comment, created_at
		FROM mood_check_ins
		WHERE patient_id = ?
		ORDER BY created_at DESC`,
		string(patientID),
	)
	if err != nil {
		return nil, fmt.Errorf("list mood check-ins: %w", err)
	}
	defer rows.Close()

	var checkIns []domain.MoodCheckIn
	for rows.Next() {
		var (
			checkIn   domain.MoodCheckIn
			id        string
			patientID string
			answers   string
			createdAt string
		)
		if err := rows.Scan(&id, &patientID, &answers, &checkIn.Comment, &createdAt); err != nil {
			return nil, fmt.Errorf("scan mood check-in: %w", err)
		}
		checkIn.ID = domain.ID(id)
		checkIn.PatientID = domain.ID(patientID)
		if err := json.Unmarshal([]byte(answers), &checkIn.Answers); err != nil {
			return nil, fmt.Errorf("decode mood answers: %w", err)
		}
		if checkIn.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		checkIns = append(checkIns, checkIn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mood check-ins: %w", err)
	}
	return checkIns, nil
}

func (s *Store) ListCompletionsForPatient(
	ctx context.Context,
	patientID domain.ID,
) ([]domain.Completion, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, assignment_id, patient_id, completed_at, comment, result, intake, side_effects
		FROM completions
		WHERE patient_id = ?
		ORDER BY completed_at DESC`,
		string(patientID),
	)
	if err != nil {
		return nil, fmt.Errorf("list completions: %w", err)
	}
	defer rows.Close()

	var completions []domain.Completion
	for rows.Next() {
		var (
			completion   domain.Completion
			id           string
			assignmentID string
			patient      string
			completedAt  string
			intake       sql.NullString
			sideEffects  sql.NullString
		)
		if err := rows.Scan(
			&id, &assignmentID, &patient, &completedAt,
			&completion.Comment, &completion.Result, &intake, &sideEffects,
		); err != nil {
			return nil, fmt.Errorf("scan completion: %w", err)
		}
		completion.ID = domain.ID(id)
		completion.AssignmentID = domain.ID(assignmentID)
		completion.PatientID = domain.ID(patient)
		if completion.CompletedAt, err = parseTime(completedAt); err != nil {
			return nil, err
		}
		if intake.Valid {
			var value domain.MedicationIntake
			if err := json.Unmarshal([]byte(intake.String), &value); err != nil {
				return nil, fmt.Errorf("decode medication intake: %w", err)
			}
			completion.Intake = &value
		}
		if sideEffects.Valid {
			var value domain.SideEffectReport
			if err := json.Unmarshal([]byte(sideEffects.String), &value); err != nil {
				return nil, fmt.Errorf("decode side effects: %w", err)
			}
			completion.SideEffects = &value
		}
		completions = append(completions, completion)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate completions: %w", err)
	}
	return completions, nil
}

func (s *Store) ListNotificationsForRecipient(
	ctx context.Context,
	recipientID domain.ID,
) ([]domain.Notification, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, recipient_id, patient_id, type, message, created_at, read_at
		FROM notifications
		WHERE recipient_id = ?
		ORDER BY created_at DESC`,
		string(recipientID),
	)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []domain.Notification
	for rows.Next() {
		var (
			notification domain.Notification
			id           string
			recipient    string
			patient      string
			kind         string
			createdAt    string
			readAt       sql.NullString
		)
		if err := rows.Scan(
			&id, &recipient, &patient, &kind, &notification.Message,
			&createdAt, &readAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		notification.ID = domain.ID(id)
		notification.RecipientID = domain.ID(recipient)
		notification.PatientID = domain.ID(patient)
		notification.Type = domain.NotificationType(kind)
		if notification.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		if readAt.Valid {
			parsed, err := parseTime(readAt.String)
			if err != nil {
				return nil, err
			}
			notification.ReadAt = &parsed
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}
	return notifications, nil
}

func (s *Store) CreateNotification(
	ctx context.Context,
	notification domain.Notification,
) error {
	var readAt any
	if notification.ReadAt != nil {
		readAt = formatTime(*notification.ReadAt)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications
			(id, recipient_id, patient_id, type, message, created_at, read_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		string(notification.ID),
		string(notification.RecipientID),
		string(notification.PatientID),
		string(notification.Type),
		notification.Message,
		formatTime(notification.CreatedAt),
		readAt,
	)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

func (s *Store) MarkNotificationRead(
	ctx context.Context,
	recipientID, notificationID domain.ID,
	readAt time.Time,
) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = ?
		WHERE id = ? AND recipient_id = ?`,
		formatTime(readAt), string(notificationID), string(recipientID),
	)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check notification update: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListSupportContents(ctx context.Context) ([]domain.SupportContent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT content_id, trigger, message, content_type
		FROM support_contents
		ORDER BY content_id`)
	if err != nil {
		return nil, fmt.Errorf("list support contents: %w", err)
	}
	defer rows.Close()

	var contents []domain.SupportContent
	for rows.Next() {
		var (
			content domain.SupportContent
			id      string
			kind    string
		)
		if err := rows.Scan(&id, &content.Trigger, &content.Message, &kind); err != nil {
			return nil, fmt.Errorf("scan support content: %w", err)
		}
		content.ID = domain.ID(id)
		content.ContentType = domain.SupportContentType(kind)
		contents = append(contents, content)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate support contents: %w", err)
	}
	return contents, nil
}

func (s *Store) CreateSupportContent(
	ctx context.Context,
	content domain.SupportContent,
) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO support_contents (content_id, trigger, message, content_type)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(content_id) DO UPDATE SET
			trigger = excluded.trigger,
			message = excluded.message,
			content_type = excluded.content_type`,
		string(content.ID), content.Trigger, content.Message, string(content.ContentType),
	)
	if err != nil {
		return fmt.Errorf("save support content: %w", err)
	}
	return nil
}

func (s *Store) ListMedicationGroups(
	ctx context.Context,
) ([]MedicationGroupWithMedications, error) {
	groupRows, err := s.db.QueryContext(ctx, `
		SELECT group_id, name, description
		FROM medication_groups
		ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list medication groups: %w", err)
	}
	defer groupRows.Close()

	var groups []MedicationGroupWithMedications
	for groupRows.Next() {
		var group domain.MedicationGroup
		if err := groupRows.Scan(&group.ID, &group.Name, &group.Description); err != nil {
			return nil, fmt.Errorf("scan medication group: %w", err)
		}
		groups = append(groups, MedicationGroupWithMedications{Group: group})
	}
	if err := groupRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate medication groups: %w", err)
	}
	if err := groupRows.Close(); err != nil {
		return nil, fmt.Errorf("close medication groups: %w", err)
	}

	for index := range groups {
		rows, err := s.db.QueryContext(ctx, `
			SELECT medication_id, name, group_id, instruction_text
			FROM medications
			WHERE group_id = ?
			ORDER BY name`,
			string(groups[index].Group.ID),
		)
		if err != nil {
			return nil, fmt.Errorf("list medications: %w", err)
		}
		for rows.Next() {
			var medication domain.Medication
			if err := rows.Scan(
				&medication.ID,
				&medication.Name,
				&medication.GroupID,
				&medication.InstructionText,
			); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan medication: %w", err)
			}
			groups[index].Medications = append(groups[index].Medications, medication)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate medications: %w", err)
		}
		if err := rows.Close(); err != nil {
			return nil, fmt.Errorf("close medications: %w", err)
		}
	}
	return groups, nil
}

func (s *Store) CreateMedicationGroup(
	ctx context.Context,
	group domain.MedicationGroup,
) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO medication_groups (group_id, name, description)
		VALUES (?, ?, ?)
		ON CONFLICT(group_id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description`,
		string(group.ID), group.Name, group.Description,
	)
	if err != nil {
		return fmt.Errorf("save medication group: %w", err)
	}
	return nil
}

func (s *Store) CreateMedication(
	ctx context.Context,
	medication domain.Medication,
) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO medications (medication_id, group_id, name, instruction_text)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(medication_id) DO UPDATE SET
			group_id = excluded.group_id,
			name = excluded.name,
			instruction_text = excluded.instruction_text`,
		string(medication.ID),
		string(medication.GroupID),
		medication.Name,
		medication.InstructionText,
	)
	if err != nil {
		return fmt.Errorf("save medication: %w", err)
	}
	return nil
}

// CountAccounts возвращает число учётных записей.
func (s *Store) CountAccounts(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count accounts: %w", err)
	}
	return count, nil
}
