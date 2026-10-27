// internal/storage/sqlite/store.go
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"example.com/backend/domain"
)

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite допускает только одного писателя одновременно.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	store := &Store{db: db}
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	const schema = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS accounts (
    id            TEXT PRIMARY KEY,
    code_value    TEXT NOT NULL UNIQUE,
    role          TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    date_of_birth TEXT NOT NULL,
    created_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS patients (
    account_id     TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    notes          TEXT NOT NULL DEFAULT '',
    consent_accepted INTEGER NOT NULL DEFAULT 0,
    policy_version TEXT NOT NULL DEFAULT '',
    accepted_at    TEXT
);

CREATE TABLE IF NOT EXISTS doctors (
    account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    specialty  TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS care_relationships (
    id         TEXT PRIMARY KEY,
    patient_id TEXT NOT NULL REFERENCES patients(account_id),
    doctor_id  TEXT NOT NULL REFERENCES doctors(account_id),
    created_at TEXT NOT NULL,
    ended_at   TEXT,
    status     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_relationships_patient
    ON care_relationships(patient_id, status);
CREATE INDEX IF NOT EXISTS idx_relationships_doctor
    ON care_relationships(doctor_id, status);

CREATE TABLE IF NOT EXISTS assignments (
    id          TEXT PRIMARY KEY,
    patient_id  TEXT NOT NULL REFERENCES patients(account_id),
    doctor_id   TEXT NOT NULL REFERENCES doctors(account_id),
    type        TEXT NOT NULL,
    title       TEXT NOT NULL,
    starts_at   TEXT NOT NULL,
    ends_at     TEXT NOT NULL,
    frequency   TEXT NOT NULL,
    days_of_week TEXT NOT NULL,
    times_of_day TEXT NOT NULL,
    status      TEXT NOT NULL,
    details     TEXT NOT NULL,
    created_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_assignments_patient
    ON assignments(patient_id, created_at);

CREATE TABLE IF NOT EXISTS completions (
    id            TEXT PRIMARY KEY,
    assignment_id TEXT NOT NULL REFERENCES assignments(id),
    patient_id    TEXT NOT NULL REFERENCES patients(account_id),
    completed_at  TEXT NOT NULL,
    comment       TEXT NOT NULL DEFAULT '',
    result        TEXT NOT NULL DEFAULT '',
    intake        TEXT,
    side_effects  TEXT
);

CREATE INDEX IF NOT EXISTS idx_completions_assignment
    ON completions(assignment_id);

CREATE TABLE IF NOT EXISTS journal_entries (
    id                TEXT PRIMARY KEY,
    patient_id        TEXT NOT NULL REFERENCES patients(account_id),
    text              TEXT NOT NULL,
    reflection_prompt TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_journal_patient_created
    ON journal_entries(patient_id, created_at);

CREATE TABLE IF NOT EXISTS mood_check_ins (
    id         TEXT PRIMARY KEY,
    patient_id TEXT NOT NULL REFERENCES patients(account_id),
    answers    TEXT NOT NULL,
    comment    TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mood_patient_created
    ON mood_check_ins(patient_id, created_at);

CREATE TABLE IF NOT EXISTS credentials (
    account_id    TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    login         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    password_salt TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS medication_groups (
    group_id    TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS medications (
    medication_id    TEXT PRIMARY KEY,
    group_id         TEXT NOT NULL REFERENCES medication_groups(group_id),
    name             TEXT NOT NULL,
    instruction_text TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_medications_group
    ON medications(group_id, name);

CREATE TABLE IF NOT EXISTS notifications (
    id           TEXT PRIMARY KEY,
    recipient_id TEXT NOT NULL REFERENCES accounts(id),
    patient_id   TEXT NOT NULL REFERENCES patients(account_id),
    type         TEXT NOT NULL,
    message      TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    read_at      TEXT
);

CREATE INDEX IF NOT EXISTS idx_notifications_recipient
    ON notifications(recipient_id, created_at);

CREATE TABLE IF NOT EXISTS support_contents (
    content_id   TEXT PRIMARY KEY,
    trigger      TEXT NOT NULL,
    message      TEXT NOT NULL,
    content_type TEXT NOT NULL
);
`

	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("migrate sqlite: %w", err)
	}
	return nil
}

func (s *Store) ListAssignmentsForDoctor(
	ctx context.Context,
	doctorID domain.ID,
) ([]domain.Assignment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM assignments WHERE doctor_id = ? ORDER BY created_at DESC`,
		string(doctorID),
	)
	if err != nil {
		return nil, fmt.Errorf("list doctor assignments: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan assignment id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assignments: %w", err)
	}
	// Store uses a single SQLite connection, so rows must be closed before
	// loading assignments; otherwise GetAssignment waits for the connection.
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close assignment rows: %w", err)
	}

	result := make([]domain.Assignment, 0, len(ids))
	for _, id := range ids {
		assignment, err := s.GetAssignment(ctx, domain.ID(id))
		if err != nil {
			return nil, err
		}
		result = append(result, assignment)
	}
	return result, nil
}

func (s *Store) ListAssignmentsForDoctorAndPatient(
	ctx context.Context,
	doctorID, patientID domain.ID,
) ([]domain.Assignment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM assignments
		 WHERE doctor_id = ? AND patient_id = ?
		 ORDER BY created_at DESC`,
		string(doctorID),
		string(patientID),
	)
	if err != nil {
		return nil, fmt.Errorf("list patient assignments for doctor: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan assignment id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assignments: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close assignment rows: %w", err)
	}

	result := make([]domain.Assignment, 0, len(ids))
	for _, id := range ids {
		assignment, err := s.GetAssignment(ctx, domain.ID(id))
		if err != nil {
			return nil, err
		}
		result = append(result, assignment)
	}
	return result, nil
}

func (s *Store) UpdateAssignmentStatus(
	ctx context.Context,
	id domain.ID,
	status domain.AssignmentStatus,
) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE assignments SET status = ? WHERE id = ?`,
		string(status),
		string(id),
	)
	if err != nil {
		return fmt.Errorf("update assignment status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check updated assignment: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// CreateAccount сохраняет общую часть учётной записи.
// Данные роли пациента или врача сохраняйте отдельно через CreatePatient/CreateDoctor.
func (s *Store) CreateAccount(ctx context.Context, account domain.Account) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (id, code_value, role, display_name, date_of_birth, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		string(account.ID),
		account.Code.Value,
		string(account.Code.Role),
		account.Personal.DisplayName,
		formatTime(account.Personal.DateOfBirth),
		formatTime(account.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	return nil
}

func (s *Store) CreatePatient(ctx context.Context, patient domain.Patient) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create patient: %w", err)
	}
	defer tx.Rollback()

	if err := insertAccount(ctx, tx, patient.Account); err != nil {
		return err
	}

	var acceptedAt any
	if patient.Consent.AcceptedAt != nil {
		acceptedAt = formatTime(*patient.Consent.AcceptedAt)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO patients (account_id, notes, consent_accepted, policy_version, accepted_at)
		VALUES (?, ?, ?, ?, ?)`,
		string(patient.ID),
		patient.Profile.Notes,
		boolInt(patient.Consent.Accepted),
		patient.Consent.PolicyVersion,
		acceptedAt,
	)
	if err != nil {
		return fmt.Errorf("create patient: %w", err)
	}

	return tx.Commit()
}

func (s *Store) CreateDoctor(ctx context.Context, doctor domain.Doctor) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create doctor: %w", err)
	}
	defer tx.Rollback()

	if err := insertAccount(ctx, tx, doctor.Account); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO doctors (account_id, specialty) VALUES (?, ?)`,
		string(doctor.ID), doctor.Profile.Specialty)
	if err != nil {
		return fmt.Errorf("create doctor: %w", err)
	}

	return tx.Commit()
}

func insertAccount(ctx context.Context, tx *sql.Tx, account domain.Account) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO accounts (id, code_value, role, display_name, date_of_birth, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		string(account.ID),
		account.Code.Value,
		string(account.Code.Role),
		account.Personal.DisplayName,
		formatTime(account.Personal.DateOfBirth),
		formatTime(account.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert account: %w", err)
	}
	return nil
}

func (s *Store) CreateRelationship(ctx context.Context, relationship domain.CareRelationship) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO care_relationships (id, patient_id, doctor_id, created_at, ended_at, status)
		VALUES (?, ?, ?, ?, ?, ?)`,
		string(relationship.ID),
		string(relationship.PatientID),
		string(relationship.DoctorID),
		formatTime(relationship.CreatedAt),
		nullableTime(relationship.EndedAt),
		string(relationship.Status),
	)
	if err != nil {
		return fmt.Errorf("create relationship: %w", err)
	}
	return nil
}

func (s *Store) EndRelationship(
	ctx context.Context,
	id, doctorID domain.ID,
	now time.Time,
) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE care_relationships
		SET status = ?, ended_at = ?
		WHERE id = ? AND doctor_id = ? AND status = ?`,
		string(domain.RelationshipEnded),
		formatTime(now),
		string(id),
		string(doctorID),
		string(domain.RelationshipActive),
	)
	if err != nil {
		return fmt.Errorf("end relationship: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check ended relationship: %w", err)
	}
	if rows == 0 {
		// Связь не найдена, уже завершена или принадлежит другому врачу.
		return domain.ErrNotFound
	}
	return nil
}

// AccountExists проверяет, существует ли учётная запись.
func (s *Store) AccountExists(ctx context.Context, id domain.ID) (bool, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM accounts WHERE id = ?)`,
		string(id),
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check account: %w", err)
	}
	return exists == 1, nil
}

// CodeExists проверяет уникальность пользовательского кода.
func (s *Store) CodeExists(ctx context.Context, code string) (bool, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM accounts WHERE code_value = ?)`,
		code,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check user code: %w", err)
	}
	return exists == 1, nil
}

// PatientIDByCode находит пациента по коду вида PXXXXXX.
func (s *Store) PatientIDByCode(ctx context.Context, code string) (domain.ID, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
		SELECT p.account_id
		FROM patients p
		JOIN accounts a ON a.id = p.account_id
		WHERE a.code_value = ? AND a.role = ?`,
		code,
		string(domain.RolePatient),
	).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("find patient by code: %w", err)
	}
	return domain.ID(id), nil
}

func (s *Store) HasActiveRelationship(ctx context.Context, patientID, doctorID domain.ID) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM care_relationships
			WHERE patient_id = ? AND doctor_id = ? AND status = ?
		)`,
		string(patientID),
		string(doctorID),
		string(domain.RelationshipActive),
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check active relationship: %w", err)
	}
	return exists == 1, nil
}

func (s *Store) CreateAssignment(ctx context.Context, assignment domain.Assignment) error {
	details, err := marshalDetails(assignment.Details)
	if err != nil {
		return err
	}

	days, err := json.Marshal(assignment.Schedule.DaysOfWeek)
	if err != nil {
		return fmt.Errorf("marshal schedule days: %w", err)
	}
	times, err := json.Marshal(assignment.Schedule.TimesOfDay)
	if err != nil {
		return fmt.Errorf("marshal schedule times: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO assignments (
			id, patient_id, doctor_id, type, title, starts_at, ends_at,
			frequency, days_of_week, times_of_day, status, details, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(assignment.ID),
		string(assignment.PatientID),
		string(assignment.DoctorID),
		string(assignment.Type),
		assignment.Title,
		formatTime(assignment.Period.StartsAt),
		formatTime(assignment.Period.EndsAt),
		assignment.Schedule.Frequency,
		string(days),
		string(times),
		string(assignment.Status),
		string(details),
		formatTime(assignment.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("create assignment: %w", err)
	}
	return nil
}

func (s *Store) GetAssignment(ctx context.Context, id domain.ID) (domain.Assignment, error) {
	var a domain.Assignment
	var patientID, doctorID, assignmentType, status string
	var startsAt, endsAt, createdAt string
	var daysJSON, timesJSON, detailsJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, patient_id, doctor_id, type, title, starts_at, ends_at,
		       frequency, days_of_week, times_of_day, status, details, created_at
		FROM assignments WHERE id = ?`,
		string(id),
	).Scan(
		&a.ID, &patientID, &doctorID, &assignmentType, &a.Title,
		&startsAt, &endsAt, &a.Schedule.Frequency,
		&daysJSON, &timesJSON, &status, &detailsJSON, &createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Assignment{}, sql.ErrNoRows
		}
		return domain.Assignment{}, fmt.Errorf("get assignment: %w", err)
	}

	a.PatientID = domain.ID(patientID)
	a.DoctorID = domain.ID(doctorID)
	a.Type = domain.AssignmentType(assignmentType)
	a.Status = domain.AssignmentStatus(status)

	if a.Period.StartsAt, err = parseTime(startsAt); err != nil {
		return domain.Assignment{}, err
	}
	if a.Period.EndsAt, err = parseTime(endsAt); err != nil {
		return domain.Assignment{}, err
	}
	if a.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Assignment{}, err
	}
	if err := json.Unmarshal([]byte(daysJSON), &a.Schedule.DaysOfWeek); err != nil {
		return domain.Assignment{}, fmt.Errorf("decode schedule days: %w", err)
	}
	if err := json.Unmarshal([]byte(timesJSON), &a.Schedule.TimesOfDay); err != nil {
		return domain.Assignment{}, fmt.Errorf("decode schedule times: %w", err)
	}
	if a.Details, err = unmarshalDetails(a.Type, []byte(detailsJSON)); err != nil {
		return domain.Assignment{}, err
	}

	return a, nil
}

func (s *Store) ListAssignmentsForPatient(ctx context.Context, patientID domain.ID) ([]domain.Assignment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM assignments WHERE patient_id = ? ORDER BY created_at DESC`,
		string(patientID))
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan assignment id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assignments: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close assignment rows: %w", err)
	}

	result := make([]domain.Assignment, 0, len(ids))
	for _, id := range ids {
		assignment, err := s.GetAssignment(ctx, domain.ID(id))
		if err != nil {
			return nil, err
		}
		result = append(result, assignment)
	}
	return result, nil
}

func (s *Store) CreateCompletion(ctx context.Context, completion domain.Completion) error {
	var intake, sideEffects any

	if completion.Intake != nil {
		raw, err := json.Marshal(completion.Intake)
		if err != nil {
			return fmt.Errorf("marshal medication intake: %w", err)
		}
		intake = string(raw)
	}
	if completion.SideEffects != nil {
		raw, err := json.Marshal(completion.SideEffects)
		if err != nil {
			return fmt.Errorf("marshal side effects: %w", err)
		}
		sideEffects = string(raw)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO completions
			(id, assignment_id, patient_id, completed_at, comment, result, intake, side_effects)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		string(completion.ID),
		string(completion.AssignmentID),
		string(completion.PatientID),
		formatTime(completion.CompletedAt),
		completion.Comment,
		completion.Result,
		intake,
		sideEffects,
	)
	if err != nil {
		return fmt.Errorf("create completion: %w", err)
	}
	return nil
}

func (s *Store) CreateJournalEntry(ctx context.Context, entry domain.JournalEntry) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO journal_entries (id, patient_id, text, reflection_prompt, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		string(entry.ID),
		string(entry.PatientID),
		entry.Text,
		entry.ReflectionPrompt,
		formatTime(entry.CreatedAt),
		formatTime(entry.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create journal entry: %w", err)
	}
	return nil
}

func (s *Store) CreateMoodCheckIn(ctx context.Context, checkIn domain.MoodCheckIn) error {
	answers, err := json.Marshal(checkIn.Answers)
	if err != nil {
		return fmt.Errorf("marshal mood answers: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO mood_check_ins (id, patient_id, answers, comment, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		string(checkIn.ID),
		string(checkIn.PatientID),
		string(answers),
		checkIn.Comment,
		formatTime(checkIn.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("create mood check-in: %w", err)
	}
	return nil
}

func marshalDetails(details domain.AssignmentDetails) ([]byte, error) {
	if details == nil {
		return nil, fmt.Errorf("%w: assignment details are required", domain.ErrInvalidInput)
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return nil, fmt.Errorf("marshal assignment details: %w", err)
	}
	return raw, nil
}

func unmarshalDetails(kind domain.AssignmentType, raw []byte) (domain.AssignmentDetails, error) {
	var details domain.AssignmentDetails

	switch kind {
	case domain.AssignmentMedication:
		details = &domain.MedicationDetails{}
	case domain.AssignmentQuestionnaire:
		details = &domain.QuestionnaireDetails{}
	case domain.AssignmentAppCheckIn:
		details = &domain.AppCheckInDetails{}
	case domain.AssignmentJournalTask:
		details = &domain.JournalTaskDetails{}
	default:
		return nil, fmt.Errorf("%w: unknown assignment type %q", domain.ErrInvalidInput, kind)
	}

	if err := json.Unmarshal(raw, details); err != nil {
		return nil, fmt.Errorf("unmarshal assignment details: %w", err)
	}
	return details, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse stored time: %w", err)
	}
	return t, nil
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
