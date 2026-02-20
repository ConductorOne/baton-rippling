package connector

import (
	"testing"

	"github.com/conductorone/baton-rippling/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestUserResource_NoWorker(t *testing.T) {
	user := client.User{
		ID:        "user-1",
		Username:  "alice",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-01-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Alice Smith"},
		Emails:    []client.Email{{Value: "alice@example.com"}},
	}

	r, err := userResource(user, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Alice Smith", r.DisplayName)
	assert.Equal(t, "user-1", r.Id.Resource)

	profile := r.GetAnnotations()
	assert.NotNil(t, profile)
}

func TestUserResource_WithFullWorker(t *testing.T) {
	user := client.User{
		ID:        "user-2",
		Username:  "bob",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-06-15T12:00:00Z",
		Name:      client.Name{DisplayName: "Bob Jones"},
		Emails:    []client.Email{{Value: "bob@example.com"}},
	}
	worker := &client.Worker{
		ID:        "worker-2",
		UserID:    "user-2",
		Title:     "Senior Engineer",
		Status:    "ACTIVE",
		StartDate: "2023-01-15",
		EndDate:   "",
		WorkEmail: "bob@company.com",
		Country:   "US",
		IsManager: true,
		ManagerID: "worker-1",
		EmploymentType: &client.EmploymentType{
			Label: "Full-time",
			Name:  "FULL_TIME",
			Type:  "employee",
		},
		Department: &client.Department{
			Name: "Engineering",
		},
		Level: &client.Level{
			Name: "Senior",
		},
		Location: &client.Location{
			WorkLocationID: "loc-sf",
		},
	}

	r, err := userResource(user, worker)
	assert.NoError(t, err)
	assert.Equal(t, "Bob Jones", r.DisplayName)
	assert.Equal(t, "user-2", r.Id.Resource)
}

func TestUserResource_WorkerWithNilNestedStructs(t *testing.T) {
	user := client.User{
		ID:        "user-3",
		Username:  "carol",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-03-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Carol Lee"},
		Emails:    []client.Email{{Value: "carol@example.com"}},
	}
	worker := &client.Worker{
		ID:             "worker-3",
		UserID:         "user-3",
		Title:          "Product Manager",
		Status:         "ACTIVE",
		IsManager:      false,
		EmploymentType: nil,
		Department:     nil,
		Level:          nil,
		Location:       nil,
	}

	r, err := userResource(user, worker)
	assert.NoError(t, err)
	assert.Equal(t, "Carol Lee", r.DisplayName)
}

func TestUserResource_WorkerEmptyStringsNotIncluded(t *testing.T) {
	user := client.User{
		ID:        "user-4",
		Username:  "dave",
		Active:    false,
		Locale:    "en-US",
		CreatedAt: "2024-02-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Dave Kim"},
		Emails:    []client.Email{{Value: "dave@example.com"}},
	}
	worker := &client.Worker{
		ID:     "worker-4",
		UserID: "user-4",
		Title:  "",
		Status: "TERMINATED",
		EmploymentType: &client.EmploymentType{
			Label: "",
			Name:  "",
			Type:  "",
		},
		Department: &client.Department{Name: ""},
		Level:      &client.Level{Name: ""},
		Location:   &client.Location{WorkLocationID: ""},
	}

	r, err := userResource(user, worker)
	assert.NoError(t, err)
	assert.Equal(t, "Dave Kim", r.DisplayName)
}

func TestUserResource_NoEmails(t *testing.T) {
	user := client.User{
		ID:        "user-5",
		Username:  "eve",
		Active:    true,
		Locale:    "en-US",
		CreatedAt: "2024-01-01T00:00:00Z",
		Name:      client.Name{DisplayName: "Eve Wu"},
		Emails:    nil,
	}

	r, err := userResource(user, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Eve Wu", r.DisplayName)
}

func TestUserResource_InvalidCreatedAt(t *testing.T) {
	user := client.User{
		ID:        "user-6",
		Username:  "frank",
		Active:    true,
		CreatedAt: "not-a-date",
		Name:      client.Name{DisplayName: "Frank"},
	}

	_, err := userResource(user, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse created_at")
}
