package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"example.com/backend/domain"
)

type Principal struct {
	UserID domain.ID
	Role   domain.AccountRole
}

// Authenticator проверяет bearer-токен и выдаёт новые токены после входа.
type Authenticator interface {
	Authenticate(r *http.Request) (Principal, error)
	IssueToken(accountID domain.ID, role domain.AccountRole) (string, error)
}

type Service interface {
	Authenticate(r *http.Request, login, password string) (domain.Account, bool, string, error)
	AcceptConsent(r *http.Request, patientID domain.ID, policyVersion string) error

	CreateRelationship(r *http.Request, doctorID domain.ID, req CreateRelationshipRequest) (domain.CareRelationship, error)
	EndRelationship(r *http.Request, doctorID, relationshipID domain.ID) error

	ListPatients(r *http.Request, doctorID domain.ID) ([]PatientSummary, error)
	GetPatientDetails(r *http.Request, doctorID, patientID domain.ID) (PatientDetails, error)
	MyDoctor(r *http.Request, patientID domain.ID) (DoctorSummary, error)
	ListJournalEntries(r *http.Request, patientID domain.ID) ([]domain.JournalEntry, error)
	GetForest(r *http.Request, patientID domain.ID) (domain.ForestState, error)
	UpdateAnamnesis(r *http.Request, patientID domain.ID, notes string) error

	ListAssignments(r *http.Request, userID domain.ID, role domain.AccountRole, patientID domain.ID) ([]domain.Assignment, error)
	CreateAssignment(r *http.Request, doctorID domain.ID, req CreateAssignmentRequest) (domain.Assignment, error)
	ChangeAssignmentStatus(r *http.Request, doctorID, assignmentID domain.ID, action string) error
	CompleteAssignment(r *http.Request, patientID, assignmentID domain.ID, req CompleteAssignmentRequest) (domain.Completion, error)

	CreateJournalEntry(r *http.Request, patientID domain.ID, req CreateJournalEntryRequest) (domain.JournalEntry, error)
	CreateMoodCheckIn(r *http.Request, patientID domain.ID, req CreateMoodCheckInRequest) (domain.MoodCheckIn, error)

	ListNotifications(r *http.Request, recipientID domain.ID) ([]domain.Notification, error)
	MarkNotificationRead(r *http.Request, recipientID, notificationID domain.ID) error
	ListMedicationGroups(r *http.Request) ([]MedicationGroup, error)
	ListSupportContents(r *http.Request) ([]domain.SupportContent, error)
}

type API struct {
	auth Authenticator
	svc  Service
	mux  *http.ServeMux
}

func New(auth Authenticator, svc Service) *API {
	a := &API{auth: auth, svc: svc, mux: http.NewServeMux()}
	a.routes()
	return a
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	a.mux.ServeHTTP(w, r)
}

func (a *API) routes() {
	a.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	a.mux.HandleFunc("POST /v1/auth/login", a.login)
	a.mux.HandleFunc("POST /v1/consent", a.acceptConsent)
	a.mux.HandleFunc("POST /v1/relationships", a.createRelationship)
	a.mux.HandleFunc("POST /v1/relationships/{id}/end", a.endRelationship)

	a.mux.HandleFunc("GET /v1/patients", a.listPatients)
	a.mux.HandleFunc("GET /v1/patients/{id}", a.getPatientDetails)
	a.mux.HandleFunc("GET /v1/my-doctor", a.myDoctor)
	a.mux.HandleFunc("GET /v1/notifications", a.listNotifications)
	a.mux.HandleFunc("POST /v1/notifications/{id}/read", a.markNotificationRead)
	a.mux.HandleFunc("GET /v1/medications", a.listMedicationGroups)
	a.mux.HandleFunc("GET /v1/support/messages", a.listSupportContents)
	a.mux.HandleFunc("GET /v1/forest", a.getForest)
	a.mux.HandleFunc("PUT /v1/anamnesis", a.updateAnamnesis)

	a.mux.HandleFunc("GET /v1/assignments", a.listAssignments)
	a.mux.HandleFunc("POST /v1/assignments", a.createAssignment)
	a.mux.HandleFunc("POST /v1/assignments/{id}/pause", a.assignmentAction("pause"))
	a.mux.HandleFunc("POST /v1/assignments/{id}/resume", a.assignmentAction("resume"))
	a.mux.HandleFunc("POST /v1/assignments/{id}/close", a.assignmentAction("close"))
	a.mux.HandleFunc("POST /v1/assignments/{id}/cancel", a.assignmentAction("cancel"))
	a.mux.HandleFunc("POST /v1/assignments/{id}/complete", a.completeAssignment)

	a.mux.HandleFunc("POST /v1/journal-entries", a.createJournalEntry)
	a.mux.HandleFunc("GET /v1/journal-entries", a.listJournalEntries)
	a.mux.HandleFunc("POST /v1/mood-check-ins", a.createMoodCheckIn)
}

type CreatePatientRequest struct {
	DisplayName   string `json:"display_name"`
	DateOfBirth   string `json:"date_of_birth"` // YYYY-MM-DD
	PolicyVersion string `json:"policy_version"`
	Accepted      bool   `json:"accepted"`
}

type CreateDoctorRequest struct {
	DisplayName string `json:"display_name"`
	DateOfBirth string `json:"date_of_birth"` // YYYY-MM-DD
	Specialty   string `json:"specialty"`
}

type CreateRelationshipRequest struct {
	PatientCode string `json:"patient_code"`
}

type CreateAssignmentRequest struct {
	PatientID  domain.ID             `json:"patient_id"`
	Title      string                `json:"title"`
	StartsAt   time.Time             `json:"starts_at"`
	EndsAt     time.Time             `json:"ends_at"`
	Frequency  string                `json:"frequency"`
	DaysOfWeek []time.Weekday        `json:"days_of_week"`
	TimesOfDay []string              `json:"times_of_day"`
	Type       domain.AssignmentType `json:"type"`
	Details    json.RawMessage       `json:"details"`
}

type CompleteAssignmentRequest struct {
	Comment          string                   `json:"comment"`
	Result           string                   `json:"result"`
	MedicationIntake *MedicationIntakeRequest `json:"medication_intake,omitempty"`
	SideEffects      *SideEffectReportRequest `json:"side_effects,omitempty"`
}

type MedicationIntakeRequest struct {
	TakenAt         time.Time `json:"taken_at"`
	ReportedAsTaken bool      `json:"reported_as_taken"`
	DosageTaken     string    `json:"dosage_taken"`
}

type SideEffectReportRequest struct {
	Effects    []string  `json:"effects"`
	Comment    string    `json:"comment"`
	ReportedAt time.Time `json:"reported_at"`
}

type CreateJournalEntryRequest struct {
	Text             string `json:"text"`
	ReflectionPrompt string `json:"reflection_prompt"`
}

type CreateMoodCheckInRequest struct {
	Answers []domain.MoodAnswer `json:"answers"`
	Comment string              `json:"comment"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type AccountSummary struct {
	AccountID   string `json:"account_id"`
	Role        string `json:"role"`
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
}

type LoginResponse struct {
	Token           string         `json:"token"`
	Account         AccountSummary `json:"account"`
	ConsentAccepted bool           `json:"consent_accepted"`
	Anamnesis       string         `json:"anamnesis"`
}

type StatusReason struct {
	Kind          string `json:"kind"`
	Description   string `json:"description"`
	ObservedValue string `json:"observed_value"`
}

type StatusSummary struct {
	Color          string         `json:"color"`
	Reasons        []StatusReason `json:"reasons"`
	CalculatedAt   time.Time      `json:"calculated_at"`
	DataSufficient bool           `json:"data_sufficient"`
}

type PatientSummary struct {
	PatientID   string        `json:"patient_id"`
	Code        string        `json:"code"`
	DisplayName string        `json:"display_name"`
	Notes       string        `json:"notes"`
	Status      StatusSummary `json:"status"`
}

type PatientDetails struct {
	Patient        PatientSummary        `json:"patient"`
	Assignments    []domain.Assignment   `json:"assignments"`
	JournalEntries []domain.JournalEntry `json:"journal_entries"`
	MoodCheckIns   []domain.MoodCheckIn  `json:"mood_check_ins"`
}

type DoctorSummary struct {
	DoctorID    string `json:"doctor_id"`
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
	Specialty   string `json:"specialty"`
}

type Medication struct {
	MedicationID    string `json:"medication_id"`
	Name            string `json:"name"`
	InstructionText string `json:"instruction_text"`
}

type MedicationGroup struct {
	GroupID     string       `json:"group_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Medications []Medication `json:"medications"`
}

type ConsentRequest struct {
	PolicyVersion string `json:"policy_version"`
}

type AnamnesisRequest struct {
	Notes string `json:"notes"`
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Login) == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid_login", "Укажите логин и пароль")
		return
	}

	account, consentAccepted, anamnesis, err := a.svc.Authenticate(
		r,
		req.Login,
		req.Password,
	)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}

	requestedRole := domain.AccountRole(strings.ToUpper(strings.TrimSpace(req.Role)))
	if requestedRole != "" && account.Code.Role != requestedRole {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Неверный логин или пароль")
		return
	}

	token, err := a.auth.IssueToken(account.ID, account.Code.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_error", "could not issue token")
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Token: token,
		Account: AccountSummary{
			AccountID:   string(account.ID),
			Role:        string(account.Code.Role),
			Code:        account.Code.Value,
			DisplayName: account.Personal.DisplayName,
		},
		ConsentAccepted: consentAccepted,
		Anamnesis:       anamnesis,
	})
}

func (a *API) acceptConsent(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RolePatient)
	if !ok {
		return
	}
	var req ConsentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.PolicyVersion) == "" {
		req.PolicyVersion = "v1"
	}
	if err := a.svc.AcceptConsent(r, p.UserID, req.PolicyVersion); err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"consent_accepted": true,
		"policy_version":   req.PolicyVersion,
	})
}

func (a *API) listPatients(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RoleDoctor)
	if !ok {
		return
	}
	out, err := a.svc.ListPatients(r, p.UserID)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"patients": out})
}

func (a *API) getPatientDetails(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RoleDoctor)
	if !ok {
		return
	}
	patientID := domain.ID(r.PathValue("id"))
	if !patientID.Valid() {
		writeError(w, http.StatusBadRequest, "invalid_patient_id", "patient id is required")
		return
	}
	out, err := a.svc.GetPatientDetails(r, p.UserID, patientID)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) myDoctor(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RolePatient)
	if !ok {
		return
	}
	out, err := a.svc.MyDoctor(r, p.UserID)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) listNotifications(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireAnyRole(w, r, domain.RoleDoctor, domain.RolePatient)
	if !ok {
		return
	}
	out, err := a.svc.ListNotifications(r, p.UserID)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notifications": out})
}

func (a *API) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireAnyRole(w, r, domain.RoleDoctor, domain.RolePatient)
	if !ok {
		return
	}
	notificationID := domain.ID(r.PathValue("id"))
	if !notificationID.Valid() {
		writeError(w, http.StatusBadRequest, "invalid_id", "notification id is required")
		return
	}
	if err := a.svc.MarkNotificationRead(r, p.UserID, notificationID); err != nil {
		a.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listMedicationGroups(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireRole(w, r, domain.RoleDoctor); !ok {
		return
	}
	out, err := a.svc.ListMedicationGroups(r)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": out})
}

func (a *API) listSupportContents(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireRole(w, r, domain.RolePatient); !ok {
		return
	}
	out, err := a.svc.ListSupportContents(r)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": out})
}

func (a *API) getForest(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RolePatient)
	if !ok {
		return
	}
	out, err := a.svc.GetForest(r, p.UserID)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"forest": out})
}

func (a *API) updateAnamnesis(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RolePatient)
	if !ok {
		return
	}
	var req AnamnesisRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := a.svc.UpdateAnamnesis(r, p.UserID, req.Notes); err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"notes": strings.TrimSpace(req.Notes)})
}

func (a *API) listJournalEntries(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RolePatient)
	if !ok {
		return
	}
	out, err := a.svc.ListJournalEntries(r, p.UserID)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"journal_entries": out})
}

func (a *API) createRelationship(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RoleDoctor)
	if !ok {
		return
	}
	var req CreateRelationshipRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	code := domain.UserCode{
		Value: req.PatientCode,
		Role:  domain.RolePatient,
	}
	if err := code.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_patient_code", "patient_code must match PXXXXXX")
		return
	}
	out, err := a.svc.CreateRelationship(r, p.UserID, req)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *API) endRelationship(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RoleDoctor)
	if !ok {
		return
	}
	id := domain.ID(r.PathValue("id"))
	if !id.Valid() {
		writeError(w, http.StatusBadRequest, "invalid_id", "relationship id is required")
		return
	}
	if err := a.svc.EndRelationship(r, p.UserID, id); err != nil {
		a.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listAssignments(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireAnyRole(w, r, domain.RolePatient, domain.RoleDoctor)
	if !ok {
		return
	}
	patientID := domain.ID(strings.TrimSpace(r.URL.Query().Get("patient_id")))
	out, err := a.svc.ListAssignments(r, p.UserID, p.Role, patientID)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) createAssignment(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RoleDoctor)
	if !ok {
		return
	}
	var req CreateAssignmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !req.PatientID.Valid() {
		writeError(w, http.StatusBadRequest, "invalid_patient_id", "patient_id is required")
		return
	}
	out, err := a.svc.CreateAssignment(r, p.UserID, req)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *API) assignmentAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := a.requireRole(w, r, domain.RoleDoctor)
		if !ok {
			return
		}
		id := domain.ID(r.PathValue("id"))
		if !id.Valid() {
			writeError(w, http.StatusBadRequest, "invalid_id", "assignment id is required")
			return
		}
		if err := a.svc.ChangeAssignmentStatus(r, p.UserID, id, action); err != nil {
			a.writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (a *API) completeAssignment(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RolePatient)
	if !ok {
		return
	}
	id := domain.ID(r.PathValue("id"))
	if !id.Valid() {
		writeError(w, http.StatusBadRequest, "invalid_id", "assignment id is required")
		return
	}
	var req CompleteAssignmentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	out, err := a.svc.CompleteAssignment(r, p.UserID, id, req)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *API) createJournalEntry(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RolePatient)
	if !ok {
		return
	}
	var req CreateJournalEntryRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		writeError(w, http.StatusBadRequest, "empty_text", "text is required")
		return
	}
	out, err := a.svc.CreateJournalEntry(r, p.UserID, req)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *API) createMoodCheckIn(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RolePatient)
	if !ok {
		return
	}
	var req CreateMoodCheckInRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Answers) == 0 {
		writeError(w, http.StatusBadRequest, "answers_required", "at least one answer is required")
		return
	}
	out, err := a.svc.CreateMoodCheckIn(r, p.UserID, req)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *API) requireRole(w http.ResponseWriter, r *http.Request, role domain.AccountRole) (Principal, bool) {
	p, err := a.auth.Authenticate(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return Principal{}, false
	}
	if p.Role != role {
		writeError(w, http.StatusForbidden, "forbidden", "Недостаточно прав")
		return Principal{}, false
	}
	return p, true
}

func (a *API) requireAnyRole(w http.ResponseWriter, r *http.Request, roles ...domain.AccountRole) (Principal, bool) {
	p, err := a.auth.Authenticate(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return Principal{}, false
	}
	for _, role := range roles {
		if p.Role == role {
			return p, true
		}
	}
	writeError(w, http.StatusForbidden, "forbidden", "Недостаточно прав")
	return Principal{}, false
}

func (a *API) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Неверный логин или пароль")
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrInvalidTransition),
		errors.Is(err, domain.ErrAlreadyEnded):
		writeError(w, http.StatusConflict, "invalid_transition", err.Error())
	case errors.Is(err, domain.ErrAlreadyCompleted):
		writeError(w, http.StatusConflict, "already_completed", "На сегодня выполнение уже отмечено")
	case errors.Is(err, domain.ErrDuplicateRelationship):
		writeError(w, http.StatusConflict, "relationship_exists", "Активная связь уже существует")
	case errors.Is(err, domain.ErrAccountExists):
		writeError(w, http.StatusConflict, "account_exists", "Учётная запись уже существует")
	case errors.Is(err, domain.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user_not_found", "Пользователь с таким кодом не найден")
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Объект не найден")
	case errors.Is(err, domain.ErrInactiveRelation),
		errors.Is(err, domain.ErrAssignmentMismatch):
		writeError(w, http.StatusForbidden, "access_denied", "operation is not allowed")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return false
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json", "request must contain one JSON object")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}
