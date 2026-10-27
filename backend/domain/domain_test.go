package domain

import (
	"errors"
	"testing"
	"time"
)

func TestUserCodeValidate(t *testing.T) {
	tests := []struct {
		name string
		code UserCode
		ok   bool
	}{
		{"patient code", UserCode{Value: "PABC123", Role: RolePatient}, true},
		{"doctor code", UserCode{Value: "VXYZ789", Role: RoleDoctor}, true},
		{"wrong prefix", UserCode{Value: "VABC123", Role: RolePatient}, false},
		{"too short", UserCode{Value: "PABC12", Role: RolePatient}, false},
		{"lowercase suffix", UserCode{Value: "Pabc123", Role: RolePatient}, false},
		{"admin has no code", UserCode{Value: "AABC123", Role: RoleAdmin}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.code.Validate()
			if test.ok && err != nil {
				t.Fatalf("expected valid code, got %v", err)
			}
			if !test.ok && !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestAssignmentTransitions(t *testing.T) {
	assignment, err := NewAssignment(
		"assignment-1",
		"patient-1",
		"doctor-1",
		"Приём лекарства",
		Period{
			StartsAt: time.Now().Add(-time.Hour),
			EndsAt:   time.Now().Add(time.Hour),
		},
		Schedule{Frequency: "DAILY", TimesOfDay: []string{"09:00"}},
		MedicationDetails{
			MedicationID: "medication-1",
			Dosage:       Dosage{Amount: 1, Unit: "мг"},
		},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	if err := assignment.Pause(); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if err := assignment.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if err := assignment.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := assignment.Cancel(); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("cancel completed assignment: expected ErrInvalidTransition, got %v", err)
	}
}

func TestNewCompletionRejectsOtherPatient(t *testing.T) {
	assignment := Assignment{
		ID:        "assignment-1",
		PatientID: "patient-1",
		Status:    AssignmentActive,
		Period: Period{
			StartsAt: time.Now().Add(-time.Hour),
			EndsAt:   time.Now().Add(time.Hour),
		},
	}

	_, err := NewCompletion("completion-1", assignment, "patient-2", time.Now())
	if !errors.Is(err, ErrAssignmentMismatch) {
		t.Fatalf("expected ErrAssignmentMismatch, got %v", err)
	}
}

func TestScoringRule(t *testing.T) {
	rule := ScoringRule{Description: "Сумма баллов"}
	answers := []Answer{
		{QuestionID: "q1", Score: 1},
		{QuestionID: "q2", Score: 2},
	}

	if score := rule.Calculate(answers); score != 3 {
		t.Fatalf("expected score 3, got %v", score)
	}
	if category := rule.Classify(3, 3); category != ResultLow {
		t.Fatalf("expected LOW, got %v", category)
	}
	if category := rule.Classify(4, 3); category != ResultOther {
		t.Fatalf("expected OTHER, got %v", category)
	}
}

func TestForestApplyActivity(t *testing.T) {
	forest := ForestState{ID: "forest-1", PatientID: "patient-1"}
	occurredAt := time.Now()

	if err := forest.ApplyActivity(ForestActivity{
		ActivityType: "JOURNAL_ENTRY",
		OccurredAt:   occurredAt,
		Points:       2,
	}); err != nil {
		t.Fatalf("apply activity: %v", err)
	}

	if forest.GrowthLevel != 2 {
		t.Fatalf("expected growth level 2, got %d", forest.GrowthLevel)
	}
	if !forest.UpdatedAt.Equal(occurredAt) {
		t.Fatalf("expected updated_at to be %v, got %v", occurredAt, forest.UpdatedAt)
	}
	if err := forest.ApplyActivity(ForestActivity{Points: -1}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
