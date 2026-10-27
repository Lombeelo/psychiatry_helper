package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"example.com/backend/domain"
)

// LoginAccount — учётная запись вместе с данными для входа.
type LoginAccount struct {
	Account         domain.Account
	PasswordHash    string
	PasswordSalt    string
	ConsentAccepted bool
	Notes           string
	Specialty       string
}

// LoginAccountByLogin находит учётную запись по логину.
func (s *Store) LoginAccountByLogin(ctx context.Context, login string) (LoginAccount, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.code_value, a.role, a.display_name, a.date_of_birth, a.created_at,
		       COALESCE(c.password_hash, ''), COALESCE(c.password_salt, ''),
		       COALESCE(p.notes, ''), COALESCE(p.consent_accepted, 0),
		       COALESCE(d.specialty, '')
		FROM accounts a
		JOIN credentials c ON c.account_id = a.id
		LEFT JOIN patients p ON p.account_id = a.id
		LEFT JOIN doctors d ON d.account_id = a.id
		WHERE c.login = ?`,
		login,
	)

	account, err := scanLoginAccount(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LoginAccount{}, domain.ErrNotFound
		}
		return LoginAccount{}, fmt.Errorf("find account by login: %w", err)
	}
	return account, nil
}

// LoginAccountByID возвращает учётную запись по её идентификатору.
func (s *Store) LoginAccountByID(ctx context.Context, id domain.ID) (LoginAccount, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.code_value, a.role, a.display_name, a.date_of_birth, a.created_at,
		       COALESCE(c.password_hash, ''), COALESCE(c.password_salt, ''),
		       COALESCE(p.notes, ''), COALESCE(p.consent_accepted, 0),
		       COALESCE(d.specialty, '')
		FROM accounts a
		LEFT JOIN credentials c ON c.account_id = a.id
		LEFT JOIN patients p ON p.account_id = a.id
		LEFT JOIN doctors d ON d.account_id = a.id
		WHERE a.id = ?`,
		string(id),
	)

	account, err := scanLoginAccount(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LoginAccount{}, domain.ErrNotFound
		}
		return LoginAccount{}, fmt.Errorf("find account by id: %w", err)
	}
	return account, nil
}

func scanLoginAccount(row *sql.Row) (LoginAccount, error) {
	var (
		account      LoginAccount
		id           string
		code         string
		role         string
		displayName  string
		dateOfBirth  string
		createdAt    string
		passwordHash string
		passwordSalt string
		consent      int
		specialty    string
	)

	if err := row.Scan(
		&id, &code, &role, &displayName, &dateOfBirth, &createdAt,
		&passwordHash, &passwordSalt,
		&account.Notes, &consent,
		&specialty,
	); err != nil {
		return LoginAccount{}, err
	}

	dob, err := parseTime(dateOfBirth)
	if err != nil {
		return LoginAccount{}, err
	}
	created, err := parseTime(createdAt)
	if err != nil {
		return LoginAccount{}, err
	}

	account.Account = domain.Account{
		ID:   domain.ID(id),
		Code: domain.UserCode{Value: code, Role: domain.AccountRole(role)},
		Personal: domain.PersonalData{
			DisplayName: displayName,
			DateOfBirth: dob,
		},
		CreatedAt: created,
	}
	account.PasswordHash = passwordHash
	account.PasswordSalt = passwordSalt
	account.ConsentAccepted = consent == 1
	account.Specialty = specialty
	return account, nil
}

// SetCredentials сохраняет логин и хеш пароля для учётной записи.
func (s *Store) SetCredentials(
	ctx context.Context,
	accountID domain.ID,
	login, passwordHash, passwordSalt string,
) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO credentials (account_id, login, password_hash, password_salt)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(account_id) DO UPDATE SET
			login = excluded.login,
			password_hash = excluded.password_hash,
			password_salt = excluded.password_salt`,
		string(accountID), login, passwordHash, passwordSalt,
	)
	if err != nil {
		return fmt.Errorf("save credentials: %w", err)
	}
	return nil
}

// AcceptConsent фиксирует согласие пациента с политикой.
func (s *Store) AcceptConsent(
	ctx context.Context,
	patientID domain.ID,
	policyVersion string,
	acceptedAt time.Time,
) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE patients
		SET consent_accepted = 1, policy_version = ?, accepted_at = ?
		WHERE account_id = ?`,
		policyVersion, formatTime(acceptedAt), string(patientID),
	)
	if err != nil {
		return fmt.Errorf("accept consent: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check accepted consent: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UpdatePatientNotes сохраняет анамнез в заметках пациента.
func (s *Store) UpdatePatientNotes(
	ctx context.Context,
	patientID domain.ID,
	notes string,
) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE patients SET notes = ? WHERE account_id = ?`,
		notes, string(patientID),
	)
	if err != nil {
		return fmt.Errorf("update patient notes: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check updated patient notes: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListPatientsForDoctor возвращает пациентов, привязанных к врачу.
func (s *Store) ListPatientsForDoctor(
	ctx context.Context,
	doctorID domain.ID,
) ([]domain.Patient, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.code_value, a.role, a.display_name, a.date_of_birth, a.created_at,
		       p.notes, p.consent_accepted, p.policy_version, p.accepted_at
		FROM care_relationships r
		JOIN patients p ON p.account_id = r.patient_id
		JOIN accounts a ON a.id = p.account_id
		WHERE r.doctor_id = ? AND r.status = ?
		ORDER BY a.display_name`,
		string(doctorID), string(domain.RelationshipActive),
	)
	if err != nil {
		return nil, fmt.Errorf("list doctor patients: %w", err)
	}
	defer rows.Close()

	var patients []domain.Patient
	for rows.Next() {
		patient, err := scanPatient(rows)
		if err != nil {
			return nil, err
		}
		patients = append(patients, patient)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate patients: %w", err)
	}
	return patients, nil
}

// GetPatientByID возвращает пациента по идентификатору.
func (s *Store) GetPatientByID(ctx context.Context, id domain.ID) (domain.Patient, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.code_value, a.role, a.display_name, a.date_of_birth, a.created_at,
		       p.notes, p.consent_accepted, p.policy_version, p.accepted_at
		FROM patients p
		JOIN accounts a ON a.id = p.account_id
		WHERE p.account_id = ?`,
		string(id),
	)

	patient, err := scanPatient(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Patient{}, domain.ErrNotFound
		}
		return domain.Patient{}, fmt.Errorf("get patient: %w", err)
	}
	return patient, nil
}

// GetDoctorByID возвращает врача по идентификатору.
func (s *Store) GetDoctorByID(ctx context.Context, id domain.ID) (domain.Doctor, error) {
	var (
		doctor      domain.Doctor
		code        string
		role        string
		displayName string
		dateOfBirth string
		createdAt   string
		specialty   string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.code_value, a.role, a.display_name, a.date_of_birth, a.created_at,
		       d.specialty
		FROM doctors d
		JOIN accounts a ON a.id = d.account_id
		WHERE d.account_id = ?`,
		string(id),
	).Scan(
		&doctor.ID, &code, &role, &displayName, &dateOfBirth, &createdAt, &specialty,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Doctor{}, domain.ErrNotFound
		}
		return domain.Doctor{}, fmt.Errorf("get doctor: %w", err)
	}

	dob, err := parseTime(dateOfBirth)
	if err != nil {
		return domain.Doctor{}, err
	}
	created, err := parseTime(createdAt)
	if err != nil {
		return domain.Doctor{}, err
	}

	doctor.Account = domain.Account{
		ID:   domain.ID(doctor.ID),
		Code: domain.UserCode{Value: code, Role: domain.AccountRole(role)},
		Personal: domain.PersonalData{
			DisplayName: displayName,
			DateOfBirth: dob,
		},
		CreatedAt: created,
	}
	doctor.Profile = domain.DoctorProfile{Specialty: specialty}
	return doctor, nil
}

// DoctorForPatient возвращает врача по активной связи.
func (s *Store) DoctorForPatient(
	ctx context.Context,
	patientID domain.ID,
) (domain.Doctor, error) {
	var doctorID string
	err := s.db.QueryRowContext(ctx, `
		SELECT doctor_id FROM care_relationships
		WHERE patient_id = ? AND status = ?
		ORDER BY created_at DESC
		LIMIT 1`,
		string(patientID), string(domain.RelationshipActive),
	).Scan(&doctorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Doctor{}, domain.ErrNotFound
		}
		return domain.Doctor{}, fmt.Errorf("find doctor for patient: %w", err)
	}
	return s.GetDoctorByID(ctx, domain.ID(doctorID))
}

func scanPatient(row interface {
	Scan(dest ...any) error
}) (domain.Patient, error) {
	var (
		patient     domain.Patient
		id          string
		code        string
		role        string
		displayName string
		dateOfBirth string
		createdAt   string
		consent     int
		policy      string
		acceptedAt  sql.NullString
	)
	if err := row.Scan(
		&id, &code, &role, &displayName, &dateOfBirth, &createdAt,
		&patient.Profile.Notes, &consent, &policy, &acceptedAt,
	); err != nil {
		return domain.Patient{}, err
	}

	dob, err := parseTime(dateOfBirth)
	if err != nil {
		return domain.Patient{}, err
	}
	created, err := parseTime(createdAt)
	if err != nil {
		return domain.Patient{}, err
	}

	patient.Account = domain.Account{
		ID:   domain.ID(id),
		Code: domain.UserCode{Value: code, Role: domain.AccountRole(role)},
		Personal: domain.PersonalData{
			DisplayName: displayName,
			DateOfBirth: dob,
		},
		CreatedAt: created,
	}
	patient.Consent = domain.Consent{
		Accepted:      consent == 1,
		PolicyVersion: policy,
	}
	if acceptedAt.Valid {
		parsed, err := parseTime(acceptedAt.String)
		if err != nil {
			return domain.Patient{}, err
		}
		patient.Consent.AcceptedAt = &parsed
	}
	return patient, nil
}
