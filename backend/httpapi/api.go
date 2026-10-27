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

// В production реализация должна проверять подписанный токен,
// получать пользователя из базы и проверять его статус.
type Authenticator interface {
	Authenticate(r *http.Request) (Principal, error)
}

type Service interface {
	CreatePatient(r *http.Request, userID domain.ID, req CreatePatientRequest) (domain.Patient, error)
	CreateRelationship(r *http.Request, doctorID domain.ID, req CreateRelationshipRequest) (domain.CareRelationship, error)
	EndRelationship(r *http.Request, doctorID, relationshipID domain.ID) error

	ListAssignments(r *http.Request, userID domain.ID, role domain.AccountRole) ([]domain.Assignment, error)
	CreateAssignment(r *http.Request, doctorID domain.ID, req CreateAssignmentRequest) (domain.Assignment, error)
	ChangeAssignmentStatus(r *http.Request, doctorID, assignmentID domain.ID, action string) error
	CompleteAssignment(r *http.Request, patientID, assignmentID domain.ID, req CompleteAssignmentRequest) (domain.Completion, error)

	CreateJournalEntry(r *http.Request, patientID domain.ID, req CreateJournalEntryRequest) (domain.JournalEntry, error)
	CreateMoodCheckIn(r *http.Request, patientID domain.ID, req CreateMoodCheckInRequest) (domain.MoodCheckIn, error)
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

	a.mux.HandleFunc("POST /v1/patients", a.createPatient)
	a.mux.HandleFunc("POST /v1/relationships", a.createRelationship)
	a.mux.HandleFunc("POST /v1/relationships/{id}/end", a.endRelationship)

	a.mux.HandleFunc("GET /v1/assignments", a.listAssignments)
	a.mux.HandleFunc("POST /v1/assignments", a.createAssignment)
	a.mux.HandleFunc("POST /v1/assignments/{id}/pause", a.assignmentAction("pause"))
	a.mux.HandleFunc("POST /v1/assignments/{id}/resume", a.assignmentAction("resume"))
	a.mux.HandleFunc("POST /v1/assignments/{id}/complete", a.completeAssignment)

	a.mux.HandleFunc("POST /v1/journal-entries", a.createJournalEntry)
	a.mux.HandleFunc("POST /v1/mood-check-ins", a.createMoodCheckIn)
}

type CreatePatientRequest struct {
	DisplayName   string `json:"display_name"`
	DateOfBirth   string `json:"date_of_birth"` // YYYY-MM-DD
	PolicyVersion string `json:"policy_version"`
	Accepted      bool   `json:"accepted"`
}

type CreateRelationshipRequest struct {
	PatientID domain.ID `json:"patient_id"`
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
	Comment string `json:"comment"`
	Result  string `json:"result"`
}

type CreateJournalEntryRequest struct {
	Text             string `json:"text"`
	ReflectionPrompt string `json:"reflection_prompt"`
}

type CreateMoodCheckInRequest struct {
	Answers []domain.MoodAnswer `json:"answers"`
	Comment string              `json:"comment"`
}

func (a *API) createPatient(w http.ResponseWriter, r *http.Request) {
	p, ok := a.requireRole(w, r, domain.RolePatient)
	if !ok {
		return
	}
	var req CreatePatientRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_date_of_birth", "date_of_birth must use YYYY-MM-DD")
		return
	}
	if !req.Accepted || strings.TrimSpace(req.PolicyVersion) == "" {
		writeError(w, http.StatusBadRequest, "consent_required", "accepted consent and policy_version are required")
		return
	}

	out, err := a.svc.CreatePatient(r, p.UserID, req)
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	_ = dob // Передайте распарсенную дату в DTO сервиса, если DTO хранит time.Time.
	writeJSON(w, http.StatusCreated, out)
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
	if !req.PatientID.Valid() {
		writeError(w, http.StatusBadRequest, "invalid_patient_id", "patient_id is required")
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
	out, err := a.svc.ListAssignments(r, p.UserID, p.Role)
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
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return Principal{}, false
	}
	if p.Role != role {
		writeError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
		return Principal{}, false
	}
	return p, true
}

func (a *API) requireAnyRole(w http.ResponseWriter, r *http.Request, roles ...domain.AccountRole) (Principal, bool) {
	p, err := a.auth.Authenticate(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return Principal{}, false
	}
	for _, role := range roles {
		if p.Role == role {
			return p, true
		}
	}
	writeError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
	return Principal{}, false
}

func (a *API) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrInvalidTransition),
		errors.Is(err, domain.ErrAlreadyEnded):
		writeError(w, http.StatusConflict, "invalid_transition", err.Error())
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
