// internal/appservice/sqlite_service.go
package appservice

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"example.com/backend/domain"
	"example.com/backend/httpapi"
	sqlitestore "example.com/backend/sqlite"
)

type SQLiteService struct {
	store *sqlitestore.Store
	now   func() time.Time
}

func NewSQLiteService(store *sqlitestore.Store) *SQLiteService {
	return &SQLiteService{
		store: store,
		now:   time.Now,
	}
}

// Проверка соответствия интерфейсу выполняется компилятором.
var _ httpapi.Service = (*SQLiteService)(nil)

func (s *SQLiteService) CreatePatient(
	r *http.Request,
	userID domain.ID,
	req httpapi.CreatePatientRequest,
) (domain.Patient, error) {
	ctx := r.Context()

	dateOfBirth, err := parseDate(req.DateOfBirth)
	if err != nil {
		return domain.Patient{}, err
	}
	if !req.Accepted || strings.TrimSpace(req.PolicyVersion) == "" {
		return domain.Patient{}, domain.ErrInvalidInput
	}

	exists, err := s.store.AccountExists(ctx, userID)
	if err != nil {
		return domain.Patient{}, err
	}
	if exists {
		return domain.Patient{}, domain.ErrAccountExists
	}

	code, err := s.newUserCode(ctx, domain.RolePatient)
	if err != nil {
		return domain.Patient{}, err
	}

	now := s.now().UTC()
	acceptedAt := now
	patient := domain.Patient{
		Account: domain.Account{
			ID:   userID,
			Code: code,
			Personal: domain.PersonalData{
				DisplayName: strings.TrimSpace(req.DisplayName),
				DateOfBirth: dateOfBirth,
			},
			CreatedAt: now,
		},
		Profile: domain.PatientProfile{},
		Consent: domain.Consent{
			Accepted:      true,
			PolicyVersion: strings.TrimSpace(req.PolicyVersion),
			AcceptedAt:    &acceptedAt,
		},
	}

	if err := patient.Personal.Validate(); err != nil {
		return domain.Patient{}, err
	}
	if err := patient.Consent.Validate(); err != nil {
		return domain.Patient{}, err
	}
	if err := patient.Code.Validate(); err != nil {
		return domain.Patient{}, err
	}
	if err := s.store.CreatePatient(ctx, patient); err != nil {
		return domain.Patient{}, err
	}
	return patient, nil
}

func (s *SQLiteService) CreateDoctor(
	r *http.Request,
	userID domain.ID,
	req httpapi.CreateDoctorRequest,
) (domain.Doctor, error) {
	ctx := r.Context()

	dateOfBirth, err := parseDate(req.DateOfBirth)
	if err != nil {
		return domain.Doctor{}, err
	}

	exists, err := s.store.AccountExists(ctx, userID)
	if err != nil {
		return domain.Doctor{}, err
	}
	if exists {
		return domain.Doctor{}, domain.ErrAccountExists
	}

	code, err := s.newUserCode(ctx, domain.RoleDoctor)
	if err != nil {
		return domain.Doctor{}, err
	}

	doctor := domain.Doctor{
		Account: domain.Account{
			ID:   userID,
			Code: code,
			Personal: domain.PersonalData{
				DisplayName: strings.TrimSpace(req.DisplayName),
				DateOfBirth: dateOfBirth,
			},
			CreatedAt: s.now().UTC(),
		},
		Profile: domain.DoctorProfile{Specialty: strings.TrimSpace(req.Specialty)},
	}

	if err := doctor.Personal.Validate(); err != nil {
		return domain.Doctor{}, err
	}
	if err := doctor.Code.Validate(); err != nil {
		return domain.Doctor{}, err
	}
	if err := s.store.CreateDoctor(ctx, doctor); err != nil {
		return domain.Doctor{}, err
	}
	return doctor, nil
}

func parseDate(value string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: date must use YYYY-MM-DD", domain.ErrInvalidInput)
	}
	return date, nil
}

// newUserCode создаёт уникальный код вида PXXXXXX или VXXXXXX.
func (s *SQLiteService) newUserCode(
	ctx context.Context,
	role domain.AccountRole,
) (domain.UserCode, error) {
	prefix := "P"
	if role == domain.RoleDoctor {
		prefix = "V"
	}

	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const maxAttempts = 20

	for attempt := 0; attempt < maxAttempts; attempt++ {
		raw := make([]byte, 6)
		if _, err := rand.Read(raw); err != nil {
			return domain.UserCode{}, fmt.Errorf("generate user code: %w", err)
		}
		suffix := make([]byte, len(raw))
		for i, value := range raw {
			suffix[i] = alphabet[int(value)%len(alphabet)]
		}

		code := domain.UserCode{Value: prefix + string(suffix), Role: role}
		taken, err := s.store.CodeExists(ctx, code.Value)
		if err != nil {
			return domain.UserCode{}, err
		}
		if !taken {
			return code, nil
		}
	}
	return domain.UserCode{}, errors.New("failed to generate a unique user code")
}

func (s *SQLiteService) CreateRelationship(
	r *http.Request,
	doctorID domain.ID,
	req httpapi.CreateRelationshipRequest,
) (domain.CareRelationship, error) {
	ctx := r.Context()

	code := domain.UserCode{
		Value: strings.TrimSpace(req.PatientCode),
		Role:  domain.RolePatient,
	}
	if err := code.Validate(); err != nil {
		return domain.CareRelationship{}, err
	}

	patientID, err := s.store.PatientIDByCode(ctx, code.Value)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.CareRelationship{}, domain.ErrUserNotFound
		}
		return domain.CareRelationship{}, err
	}

	doctorExists, err := s.store.AccountExists(ctx, doctorID)
	if err != nil {
		return domain.CareRelationship{}, err
	}
	if !doctorExists {
		return domain.CareRelationship{}, domain.ErrUserNotFound
	}

	alreadyActive, err := s.store.HasActiveRelationship(ctx, patientID, doctorID)
	if err != nil {
		return domain.CareRelationship{}, err
	}
	if alreadyActive {
		return domain.CareRelationship{}, domain.ErrDuplicateRelationship
	}

	relationship, err := domain.NewCareRelationship(
		domain.ID(uuid.NewString()),
		patientID,
		doctorID,
		s.now().UTC(),
	)
	if err != nil {
		return domain.CareRelationship{}, err
	}

	if err := s.store.CreateRelationship(ctx, *relationship); err != nil {
		return domain.CareRelationship{}, err
	}
	return *relationship, nil
}

func (s *SQLiteService) EndRelationship(
	r *http.Request,
	doctorID, relationshipID domain.ID,
) error {
	// Завершать связь может только врач, указанный в самой связи:
	// doctor_id входит в условие UPDATE, поэтому чужую связь завершить нельзя.
	return s.store.EndRelationship(
		r.Context(),
		relationshipID,
		doctorID,
		s.now().UTC(),
	)
}

func (s *SQLiteService) ListAssignments(
	r *http.Request,
	userID domain.ID,
	role domain.AccountRole,
	patientID domain.ID,
) ([]domain.Assignment, error) {
	switch role {
	case domain.RolePatient:
		return s.store.ListAssignmentsForPatient(r.Context(), userID)
	case domain.RoleDoctor:
		if patientID.Valid() {
			return s.store.ListAssignmentsForDoctorAndPatient(
				r.Context(),
				userID,
				patientID,
			)
		}
		return s.store.ListAssignmentsForDoctor(r.Context(), userID)
	default:
		return nil, errors.New("unsupported role")
	}
}

func (s *SQLiteService) CreateAssignment(
	r *http.Request,
	doctorID domain.ID,
	req httpapi.CreateAssignmentRequest,
) (domain.Assignment, error) {
	details, err := decodeAssignmentDetails(req.Type, req.Details)
	if err != nil {
		return domain.Assignment{}, err
	}

	now := s.now().UTC()
	assignment, err := domain.NewAssignment(
		domain.ID(uuid.NewString()),
		req.PatientID,
		doctorID,
		req.Title,
		domain.Period{
			StartsAt: req.StartsAt,
			EndsAt:   req.EndsAt,
		},
		domain.Schedule{
			Frequency:  req.Frequency,
			DaysOfWeek: req.DaysOfWeek,
			TimesOfDay: req.TimesOfDay,
		},
		details,
		now,
	)
	if err != nil {
		return domain.Assignment{}, err
	}

	// Важно: до сохранения нужно проверить активную связь doctorID ↔ PatientID.
	active, err := s.store.HasActiveRelationship(r.Context(), req.PatientID, doctorID)
	if err != nil {
		return domain.Assignment{}, err
	}
	if !active {
		return domain.Assignment{}, domain.ErrInactiveRelation
	}

	if err := s.store.CreateAssignment(r.Context(), *assignment); err != nil {
		return domain.Assignment{}, err
	}
	return *assignment, nil
}

func (s *SQLiteService) ChangeAssignmentStatus(
	r *http.Request,
	doctorID, assignmentID domain.ID,
	action string,
) error {
	ctx := r.Context()

	assignment, err := s.store.GetAssignment(ctx, assignmentID)
	if err != nil {
		return notFoundAsDomainError(err)
	}
	if assignment.DoctorID != doctorID {
		return domain.ErrInactiveRelation
	}

	active, err := s.store.HasActiveRelationship(
		ctx,
		assignment.PatientID,
		doctorID,
	)
	if err != nil {
		return err
	}
	if !active {
		return domain.ErrInactiveRelation
	}

	switch action {
	case "pause":
		err = assignment.Pause()
	case "resume":
		err = assignment.Resume()
	case "close":
		err = assignment.Close()
	case "cancel":
		err = assignment.Cancel()
	default:
		return domain.ErrInvalidInput
	}
	if err != nil {
		return err
	}
	return s.store.UpdateAssignmentStatus(ctx, assignment.ID, assignment.Status)
}

func (s *SQLiteService) CompleteAssignment(
	r *http.Request,
	patientID, assignmentID domain.ID,
	req httpapi.CompleteAssignmentRequest,
) (domain.Completion, error) {
	ctx := r.Context()

	assignment, err := s.store.GetAssignment(ctx, assignmentID)
	if err != nil {
		return domain.Completion{}, notFoundAsDomainError(err)
	}

	now := s.now().UTC()
	completion, err := domain.NewCompletion(
		domain.ID(uuid.NewString()),
		assignment,
		patientID,
		now,
	)
	if err != nil {
		return domain.Completion{}, err
	}
	completion.Comment = req.Comment
	completion.Result = req.Result

	if assignment.Type == domain.AssignmentMedication {
		if req.MedicationIntake == nil {
			return domain.Completion{}, fmt.Errorf(
				"%w: medication assignment completion requires medication_intake",
				domain.ErrInvalidInput,
			)
		}
		intake := domain.MedicationIntake{
			TakenAt:         req.MedicationIntake.TakenAt,
			ReportedAsTaken: req.MedicationIntake.ReportedAsTaken,
			DosageTaken:     req.MedicationIntake.DosageTaken,
		}
		if intake.TakenAt.IsZero() {
			intake.TakenAt = now
		}
		if err := intake.Validate(); err != nil {
			return domain.Completion{}, err
		}
		completion.Intake = &intake
	}

	if req.SideEffects != nil {
		report := domain.SideEffectReport{
			Effects:    req.SideEffects.Effects,
			Comment:    req.SideEffects.Comment,
			ReportedAt: req.SideEffects.ReportedAt,
		}
		if report.ReportedAt.IsZero() {
			report.ReportedAt = now
		}
		if err := report.Validate(); err != nil {
			return domain.Completion{}, err
		}
		completion.SideEffects = &report
	}

	if err := s.store.CreateCompletion(ctx, *completion); err != nil {
		return domain.Completion{}, err
	}
	return *completion, nil
}

func notFoundAsDomainError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func (s *SQLiteService) CreateJournalEntry(
	r *http.Request,
	patientID domain.ID,
	req httpapi.CreateJournalEntryRequest,
) (domain.JournalEntry, error) {
	now := s.now().UTC()
	entry := domain.JournalEntry{
		ID:               domain.ID(uuid.NewString()),
		PatientID:        patientID,
		Text:             strings.TrimSpace(req.Text),
		ReflectionPrompt: req.ReflectionPrompt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if entry.Text == "" {
		return domain.JournalEntry{}, domain.ErrInvalidInput
	}
	if err := s.store.CreateJournalEntry(r.Context(), entry); err != nil {
		return domain.JournalEntry{}, err
	}
	return entry, nil
}

func (s *SQLiteService) CreateMoodCheckIn(
	r *http.Request,
	patientID domain.ID,
	req httpapi.CreateMoodCheckInRequest,
) (domain.MoodCheckIn, error) {
	checkIn := domain.MoodCheckIn{
		ID:        domain.ID(uuid.NewString()),
		PatientID: patientID,
		Answers:   req.Answers,
		Comment:   req.Comment,
		CreatedAt: s.now().UTC(),
	}
	if err := checkIn.Validate(); err != nil {
		return domain.MoodCheckIn{}, err
	}
	if err := s.store.CreateMoodCheckIn(r.Context(), checkIn); err != nil {
		return domain.MoodCheckIn{}, err
	}
	return checkIn, nil
}

func decodeAssignmentDetails(
	kind domain.AssignmentType,
	raw json.RawMessage,
) (domain.AssignmentDetails, error) {
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
		return nil, domain.ErrInvalidInput
	}

	if len(raw) == 0 {
		return nil, domain.ErrInvalidInput
	}
	if err := json.Unmarshal(raw, details); err != nil {
		return nil, domain.ErrInvalidInput
	}
	if err := details.Validate(); err != nil {
		return nil, err
	}
	return details, nil
}

// Защита от случайного использования контекста без дедлайна —
// все методы передают r.Context() в SQLite.
var _ context.Context
